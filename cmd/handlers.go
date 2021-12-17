package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/go-co-op/gocron"
	"github.com/labstack/echo/v4"
	"github.com/saisona/simba"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

func handleMigration(e *echo.Echo) bool {
	hasMigrationStr, exists := os.LookupEnv("APP_MIGRATE")
	var hasMigration bool = exists
	var err error

	if exists && hasMigrationStr != "" {
		if hasMigration, err = strconv.ParseBool(hasMigrationStr); err != nil {
			e.Logger.Warnf("APP_MIGRATE(%s) has been set but cannot be converted to bool : %s ", hasMigrationStr, err.Error())
			return false
		}
	}
	return hasMigration
}

func initApplication(e *echo.Echo, threadTS string) (string, *simba.Config, *gorm.DB, *slack.Client, *gocron.Scheduler, error) {
	slackSigningSecret, ok := os.LookupEnv("SLACK_SIGNING_SECRET")
	if !ok || slackSigningSecret == "" {
		err := fmt.Errorf("SLACK_SIGNING_SECRET does not exists and must be provided")
		return "", nil, nil, nil, nil, err
	}

	config, err := simba.InitConfig()
	if err != nil {
		err = fmt.Errorf("failed initConfig: %s", err.Error())
		return slackSigningSecret, nil, nil, nil, nil, err
	}
	dbClient := simba.InitDbClient(config.DB.Host, config.DB.Username, config.DB.Password, config.DB.Name, handleMigration(e))

	slackClient := slack.New(config.SLACK_API_TOKEN, slack.OptionDebug(true), slack.OptionLog(log.Default()))

	scheduler, _, err := simba.InitScheduler(dbClient, slackClient, config, threadTS)
	if err != nil {
		err = fmt.Errorf("failed initScheduler: %s", err.Error())
		return slackSigningSecret, nil, nil, nil, nil, err
	}
	return slackSigningSecret, config, dbClient, slackClient, scheduler, nil
}

func watchValueChanged(valueToChange *string, channel chan string, logger echo.Logger) {
	for {
		oldValue := *valueToChange
		val := <-channel
		*valueToChange = val
		logger.Infof("Changed slackTS from %s to %s", oldValue, *valueToChange)
	}
}

// func handleUpdateMood(config *simba.Config, slackClient *slack.Client, dbClient *gorm.DB, threadTS, channelId, userId, username string, action *slack.BlockAction, previousBlock []slack.Block) error {
// 	if _, err := simba.HandleAddDailyMood(dbClient, slackClient, channelId, userId, username, action.Value, threadTS); err != nil {
// 		log.Printf("Error HandleAddDailyMood = %s", err.Error())
// 		return err
// 	} else {
// 		log.Printf("Mood %s has been added for the daily for %s", action.Value, username)
// 		slackMessage := fmt.Sprintf("<@%s> has responded to the daily message with %s", userId, strings.ReplaceAll(action.Value, "_", " "))
// 		simba.SendSlackTSMessage(slackClient, config, slackMessage, threadTS)
// 		slackClient.AddReaction("robot_face", slack.ItemRef{Timestamp: threadTS, Channel: channelId})
// 		return nil
// 	}
// }

func handleSendKindMessage(slackClient *slack.Client, userId string, action *slack.BlockAction) error {
	log.Printf("Warning this has to be handled by another thing (blockId:%s, value:%s)", action.ActionID, action.Value)
	privateChannel, _, _, err := slackClient.OpenConversation(&slack.OpenConversationParameters{Users: []string{action.Value}})
	if err != nil {
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
		return err
	} else if err = simba.SendImage(slackClient, privateChannel.ID, filePath, title, comment); err != nil {
		return err
	}
	return nil
}
