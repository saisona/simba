/**
 * File              : slack.go
 * Author            : Alexandre Saison <alexandre.saison@inarix.com>
 * Date              : 08.11.2021
 * Last Modified Date: 14.11.2021
 * Last Modified By  : Alexandre Saison <alexandre.saison@inarix.com>
 */
package simba

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

func slackTextObject(text string) slack.MsgOption {
	return slack.MsgOptionText(text, false)
}

func SendImage(client *slack.Client, config *Config, filePath, title, comment string) error {
	if fileInfo, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		// path/to/whatever does not exist
		return fmt.Errorf("File %s does not exists", filePath)
	} else {
		sizeMo := fileInfo.Size() / 1e5
		sizeRest := fileInfo.Size() % 1e5
		log.Printf("Uploading %d,%dMo file", sizeMo, sizeRest)
	}

	file, err := client.UploadFile(slack.FileUploadParameters{File: filePath, Title: title, InitialComment: comment, Content: "", Channels: []string{config.CHANNEL_ID}})
	if err != nil {
		return err
	}
	log.Printf("mimeType=%s\n", file.Mimetype)
	return nil
}

func SendSlackTSMessage(client *slack.Client, config *Config, message string, ts string) (string, error) {
	_, threadTS, _, err := client.SendMessage(config.CHANNEL_ID, slackTextObject(message), slack.MsgOptionTS(ts))
	if err != nil {
		return "", err
	}
	return threadTS, nil
}

func SendSlackMessage(client *slack.Client, config *Config, message string) (string, error) {
	_, threadTS, _, err := client.SendMessage(config.CHANNEL_ID, slackTextObject(message))
	if err != nil {
		return "", err
	}
	return threadTS, nil
}

func SendSlackMessageToUser(client *slack.Client, userId, message string) (string, error) {
	_, threadTS, _, err := client.SendMessage(userId, slackTextObject(message))
	if err != nil {
		return "", err
	}
	return threadTS, nil
}

//@NotImplemented
func SendSlackBlocksToUser(client *slack.Client, userId string, blocks []slack.Block, dbClient *gorm.DB) (string, error) {
	return "", fmt.Errorf("#SendSlackBlocksToUser has not been implemented yet")
	//if blocks == nil || len(blocks) == 0 {
	//	blocksDefault := fromJsonToBlocks(dbClient, userId).Blocks.BlockSet
	//	_, threadTS, err := client.PostMessage(userId, slack.MsgOptionBlocks(blocksDefault...))
	//	if err != nil {
	//		return "", err
	//	}
	//	return threadTS, nil
	//} else {
	//	_, threadTS, err := client.PostMessage(userId, slack.MsgOptionBlocks(blocks...))
	//	if err != nil {
	//		return "", err
	//	}
	//	return threadTS, nil
	//}
}

func fetchQuoteOfTheDay() (string, string, error) {
	req, err := http.NewRequest("GET", "https://type.fit/api/quotes", nil)
	if err != nil {
		return "", "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	respJson := make([]map[string]string, 50)

	reader := json.NewDecoder(resp.Body)
	if err = reader.Decode(&respJson); err != nil {
		return "", "", err
	}

	min := 0
	max := len(respJson)
	index := (rand.Intn(max-min) + min)
	return respJson[index]["author"], respJson[index]["text"], nil
}

func AddingContextAuthor(authorName string) *slack.ContextBlock {
	blockId := fmt.Sprintf("author_context_qotd_%d", time.Now().UnixMilli())
	textBlock := slack.NewTextBlockObject("plain_text", "From "+authorName, true, false)
	return slack.NewContextBlock(blockId, textBlock)
}

func firstSectionBlock() (string, *slack.SectionBlock) {
	author, qotd, err := fetchQuoteOfTheDay()
	if err != nil {
		qotd = "Meow"
		author = "Simba"
		log.Printf("Failed fetch Quote of the Day = %s", err.Error())
	}
	quoteOfTheDay := fmt.Sprintf("Hey folks! What is your mood today:\nQuote of the Day: *%s*", qotd)
	firstLine := slack.NewTextBlockObject("mrkdwn", quoteOfTheDay, false, false)
	sectionBlock := slack.NewSectionBlock(firstLine, nil, nil)
	return author, sectionBlock
}

func secondSectionBlock(dbClient *gorm.DB, channelId string) *slack.SectionBlock {
	user, mood, err := FetchLastPersonInBadMood(dbClient, channelId)
	if err != nil {
		log.Printf("FetchLastPersonInBadMood failed: %s", err.Error())
		user = &User{Username: "John Snow"}
		mood = &DailyMood{Mood: "I know nothing"}
	}

	var badMoodText string
	if mood.Context == "" {
		badMoodText = fmt.Sprintf("*Last co-worker in %s:*\n%s", strings.ReplaceAll(mood.Mood, "_", " "), user.Username)
	} else {
		badMoodText = fmt.Sprintf("*Last co-worker in %s:*\n%s (context: %s)", strings.ReplaceAll(mood.Mood, "_", " "), user.Username, mood.Context)
	}
	secondSectionFirstBlock := slack.NewTextBlockObject("mrkdwn", badMoodText, false, false)
	if secondSectionFirstBlock.Validate() != nil {
		log.Printf("WARNING FAILED %s", secondSectionFirstBlock.Validate().Error())
		return nil
	}

	lastBadMoodDay := mood.CreatedAt.Format(time.ANSIC)
	secondSectionSecondBlock := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*When:*\n%s", lastBadMoodDay), false, false)
	if secondSectionSecondBlock.Validate() != nil {
		log.Printf("WARNING FAILED %s", secondSectionFirstBlock.Validate().Error())
		return nil
	}

	buttonText := slack.NewTextBlockObject("plain_text", "Send kind message :heart:", true, false)

	buttonBlockId := fmt.Sprintf("send_kind_message_%d", time.Now().UnixMilli())

	buttonAccessory := slack.NewButtonBlockElement(buttonBlockId, user.SlackUserID, buttonText)
	buttonAccessory.Style = slack.StyleDanger
	secondSectionAccessory := slack.NewAccessory(buttonAccessory)

	textFields := []*slack.TextBlockObject{secondSectionFirstBlock, secondSectionSecondBlock}

	return slack.NewSectionBlock(nil, textFields, secondSectionAccessory)
}

func actionSectionBlock() *slack.ActionBlock {
	timeNow := time.Now().UnixMilli()
	actionBlockId := fmt.Sprintf("action_block_mood_user_%d", timeNow)

	goodMoodButtonText := slack.NewTextBlockObject("plain_text", "Good Mood :heart:", true, false)
	goodMoodButton := slack.NewButtonBlockElement(fmt.Sprintf("mood_user_good_%d", timeNow), "good_mood", goodMoodButtonText)
	goodMoodButton.Style = slack.StylePrimary
	if goodMoodButtonText.Validate() != nil {
		log.Printf("WARNING goodMood button display failed: %s", goodMoodButtonText.Validate().Error())
		return nil
	}

	averageMoodButtonText := slack.NewTextBlockObject("plain_text", "Meow :yellow_heart:", true, false)
	averageMoodButton := slack.NewButtonBlockElement(fmt.Sprintf("mood_user_avg_%d", timeNow), "average_mood", averageMoodButtonText)
	averageMoodButton.Style = slack.StyleDefault
	if averageMoodButtonText.Validate() != nil {
		log.Printf("WARNING averageMood button display failed: %s", averageMoodButtonText.Validate().Error())
		return nil
	}

	badMoodButtonText := slack.NewTextBlockObject("plain_text", "Grr ! :black_heart:", true, false)
	badMoodButton := slack.NewButtonBlockElement(fmt.Sprintf("mood_user_bad_%d", timeNow), "bad_mood", badMoodButtonText)
	badMoodButton.Style = slack.StyleDanger
	if badMoodButtonText.Validate() != nil {
		log.Printf("WARNING baddMood button display failed: %s", badMoodButtonText.Validate().Error())
		return nil
	}

	return slack.NewActionBlock(actionBlockId, goodMoodButton, averageMoodButton, badMoodButton)
}

func fromJsonToBlocks(dbClient *gorm.DB, channelId string) slack.Message {
	authorName, slackFirstSection := firstSectionBlock()
	contextBlock := AddingContextAuthor(authorName)
	slackSecondSection := secondSectionBlock(dbClient, channelId)
	actions := actionSectionBlock()
	return slack.NewBlockMessage(slackFirstSection, contextBlock, slackSecondSection, actions)
}

func SendSlackBlocks(client *slack.Client, config *Config, blocks []slack.Block, dbClient *gorm.DB) (string, error) {
	if blocks == nil || len(blocks) == 0 {
		blocksDefault := fromJsonToBlocks(dbClient, config.CHANNEL_ID).Blocks.BlockSet
		_, threadTS, err := client.PostMessage(config.CHANNEL_ID, slack.MsgOptionBlocks(blocksDefault...))
		if err != nil {
			return "", err
		}
		return threadTS, nil
	} else {
		_, threadTS, err := client.PostMessage(config.CHANNEL_ID, slack.MsgOptionBlocks(blocks...))
		if err != nil {
			return "", err
		}
		return threadTS, nil
	}
}

func UpdateMessage(client *slack.Client, config *Config, threadTS string, message string, blocks []slack.Block) (string, error) {
	if message != "" {
		msgOptions := slack.MsgOptionText(message, false)
		_, newThreadTS, _, err := client.UpdateMessage(config.CHANNEL_ID, threadTS, msgOptions)
		if err != nil {
			return threadTS, err
		}
		return newThreadTS, nil
	} else {
		msgOptions := slack.MsgOptionBlocks(blocks...)

		_, newThreadTS, _, err := client.UpdateMessage(config.CHANNEL_ID, threadTS, msgOptions)
		if err != nil {
			return threadTS, err
		}
		return newThreadTS, nil
	}
}
