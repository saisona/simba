/**
 * File              : slack.go
 * Author            : Alexandre Saison <alexandre.saison@inarix.com>
 * Date              : 08.11.2021
 * Last Modified Date: 09.11.2021
 * Last Modified By  : Alexandre Saison <alexandre.saison@inarix.com>
 */
package simba

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/slack-go/slack"
)

func slackTextObject(text string) slack.MsgOption {
	return slack.MsgOptionText(text, false)
}

func SendSlackMessage(client *slack.Client, config *Config, message string) (string, error) {
	channelId, threadTS, _, err := client.SendMessage(config.CHANNEL_ID, slackTextObject(message))
	if err != nil {
		return "", err
	}
	log.Printf("ChannelId =%s", channelId)
	return threadTS, nil
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

func firstSectionBlock() *slack.SectionBlock {

	author, qotd, err := fetchQuoteOfTheDay()
	if err != nil {
		qotd = "Meow"
		log.Printf("Failed fetch Quote of the Day = %s", err.Error())
	}
	quoteOfTheDay := fmt.Sprintf("Hey folks! What is your mood today:\nQuote of the Day: *%s*\nFrom %s", qotd, author)
	firstLine := slack.NewTextBlockObject("mrkdwn", quoteOfTheDay, false, false)
	sectionBlock := slack.NewSectionBlock(firstLine, nil, nil)
	return sectionBlock
}

func secondSectionBlock() *slack.SectionBlock {
	//TODO: Handle fetch last person in bad mood (REPLACE XXX BY User itself)
	secondSectionFirstBlock := slack.NewTextBlockObject("mrkdwn", "*Last co-worker in bad mood:*\nXXXXX", false, false)
	if secondSectionFirstBlock.Validate() != nil {
		log.Printf("WARNING FAILED %s", secondSectionFirstBlock.Validate().Error())
		return nil
	}
	//TODO: Handle time of when that person was in the bad mood
	secondSectionSecondBlock := slack.NewTextBlockObject("mrkdwn", "*When:*\nYesterday", false, false)
	if secondSectionSecondBlock.Validate() != nil {
		log.Printf("WARNING FAILED %s", secondSectionFirstBlock.Validate().Error())
		return nil
	}

	buttonText := slack.NewTextBlockObject("plain_text", "Send kind message :heart:", true, false)

	buttonBlockId := fmt.Sprintf("send_kind_message_%d", time.Now().UnixMilli())

	buttonAccessory := slack.NewButtonBlockElement(buttonBlockId, "SLACK_USER_ID", buttonText)
	buttonAccessory.Style = slack.StyleDanger
	secondSectionAccessory := slack.NewAccessory(buttonAccessory)

	textFields := []*slack.TextBlockObject{secondSectionFirstBlock, secondSectionSecondBlock}

	return slack.NewSectionBlock(nil, textFields, secondSectionAccessory)
}

func actionSectionBlock() *slack.ActionBlock {
	timeNow := time.Now().UnixMilli()
	actionBlockId := fmt.Sprintf("mood_user_%d", timeNow)

	goodMoodButtonText := slack.NewTextBlockObject("plain_text", "Good Mood :heart:", true, false)
	goodMoodButton := slack.NewButtonBlockElement(fmt.Sprintf("good_mood_%d", timeNow), "good_mood", goodMoodButtonText)
	goodMoodButton.Style = slack.StylePrimary
	if goodMoodButtonText.Validate() != nil {
		log.Printf("WARNING FAILED %s", goodMoodButtonText.Validate().Error())
		return nil
	}

	averageMoodButtonText := slack.NewTextBlockObject("plain_text", "Meow :yellow_heart:", true, false)
	averageMoodButton := slack.NewButtonBlockElement(fmt.Sprintf("average_mood_%d", timeNow), "average_mood", averageMoodButtonText)
	averageMoodButton.Style = slack.StyleDefault
	if averageMoodButtonText.Validate() != nil {
		log.Printf("WARNING FAILED %s", averageMoodButtonText.Validate().Error())
		return nil
	}

	badMoodButtonText := slack.NewTextBlockObject("plain_text", "Grr ! :black_heart:", true, false)
	badMoodButton := slack.NewButtonBlockElement(fmt.Sprintf("bad_mood_%d", timeNow), "bad_mood", badMoodButtonText)
	badMoodButton.Style = slack.StyleDanger
	if badMoodButtonText.Validate() != nil {
		log.Printf("WARNING FAILED %s", badMoodButtonText.Validate().Error())
		return nil
	}

	return slack.NewActionBlock(actionBlockId, goodMoodButton, averageMoodButton, badMoodButton)
}

func fromJsonToBlocks() slack.Message {
	slackFirstSection := firstSectionBlock()
	slackSecondSection := secondSectionBlock()
	actions := actionSectionBlock()
	return slack.NewBlockMessage(slackFirstSection, slackSecondSection, actions)
}

func SendSlackBlocks(client *slack.Client, config *Config, blocks []slack.Block) (string, error) {
	if blocks == nil || len(blocks) == 0 {
		blocksDefault := fromJsonToBlocks().Blocks.BlockSet
		log.Printf("Blocks len %d", len(blocksDefault))
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
