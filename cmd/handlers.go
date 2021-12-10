package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/saisona/simba"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

func watchValueChanged(valueToChange *string, channel chan string, logger echo.Logger) {
	for {
		val, _ := <-channel
		logger.Debugf("received new value for ThreadTS=", val)
		if valueToChange != nil {
			logger.Debugf("OldValue=%s\n", *valueToChange)
			*valueToChange = val
			logger.Debugf("NewValue=%s\n", *valueToChange)
		} else {
			*valueToChange = val
		}
	}
}

func handleError(errChan chan error, c echo.Context) {
	for hasErr, more := <-errChan; more; {
		if hasErr != nil {
			c.Error(hasErr)
		}
		log.Printf("No error")
	}
}

func handleUpdateMood(config *simba.Config, slackClient *slack.Client, dbClient *gorm.DB, threadTS, channelId, userId, username string, action *slack.BlockAction) error {
	if err := simba.HandleAddDailyMood(dbClient, channelId, userId, username, action.Value, ""); err != nil {
		return err
	} else {
		log.Printf("Mood %s has been added for the daily for %s", action.Value, username)
		slackMessage := fmt.Sprintf("<@%s> has responded to the daily message with %s", userId, strings.ReplaceAll(action.Value, "_", " "))
		simba.SendSlackTSMessage(slackClient, config, slackMessage, threadTS)
		slackClient.AddReaction("robot_face", slack.ItemRef{Timestamp: threadTS, Channel: channelId})
		return nil
	}
}

func handleSendKindMessage(slackClient *slack.Client, errChan chan error, userId string, action *slack.BlockAction) error {
	log.Printf("Warning this has to be handled by another thing (blockId:%s, value:%s)", action.ActionID, action.Value)
	defer close(errChan)
	privateChannel, _, _, err := slackClient.OpenConversation(&slack.OpenConversationParameters{Users: []string{action.Value}})
	if err != nil {
		log.Printf("WARNING CANNOT OPEN PRIVATE CONV : %s\nPrivateChannel=%+v", err.Error(), privateChannel)
		return err
	}

	words := simba.GenerateBuzzWords()
	title, urlToDownload, err := simba.FetchRelatedGif(words[simba.GenerateRandomIndexBuzzWord(words)])
	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("/tmp/%s", title)
	comment := fmt.Sprintf("<@%s> thought about you :hugging_face: and wanted to send you some kind image", userId)
	if err := simba.DownloadFile(filePath, urlToDownload, false); err != nil {
		errChan <- err
		return err
	} else if err = simba.SendImage(slackClient, privateChannel.ID, filePath, title, comment); err != nil {
		errChan <- err
		return err
	}
	return nil
}
