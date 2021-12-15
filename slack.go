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

func SendImage(client *slack.Client, channelId, filePath, title, comment string) error {
	if fileInfo, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("file %s does not exists", filePath)
	} else {
		sizeMo := fileInfo.Size() / 1e5
		sizeRest := fileInfo.Size() % 1e5
		log.Printf("Uploading %d,%dMo file", sizeMo, sizeRest)
	}

	file, err := client.UploadFile(slack.FileUploadParameters{File: filePath, Title: title, InitialComment: comment, Content: "", Channels: []string{channelId}})
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
	secondSectionFirstBlock := slack.NewTextBlockObject(slack.MarkdownType, badMoodText, false, false)
	if secondSectionFirstBlock.Validate() != nil {
		log.Printf("WARNING FAILED %s", secondSectionFirstBlock.Validate().Error())
		return nil
	}

	lastBadMoodDay := mood.CreatedAt.Format(time.ANSIC)
	secondSectionSecondBlock := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*When:*\n%s", lastBadMoodDay), false, false)
	if secondSectionSecondBlock.Validate() != nil {
		log.Printf("WARNING FAILED %s", secondSectionFirstBlock.Validate().Error())
		return nil
	}

	buttonText := slack.NewTextBlockObject(slack.PlainTextType, "Send kind message :heart:", true, false)

	buttonBlockId := fmt.Sprintf("send_kind_message_%d", time.Now().UnixMilli())

	buttonAccessory := slack.NewButtonBlockElement(buttonBlockId, user.SlackUserID, buttonText)
	buttonAccessory.Style = slack.StyleDanger
	secondSectionAccessory := slack.NewAccessory(buttonAccessory)

	textFields := []*slack.TextBlockObject{secondSectionFirstBlock, secondSectionSecondBlock}

	return slack.NewSectionBlock(nil, textFields, secondSectionAccessory)
}

func FetchUserById(slackClient *slack.Client, userId string) (*slack.User, error) {
	return slackClient.GetUserInfo(userId)
}

func actionSectionBlock() *slack.ActionBlock {
	timeNow := time.Now().UnixMilli()
	actionBlockId := fmt.Sprintf("action_block_mood_user_%d", timeNow)

	goodMoodButtonText := slack.NewTextBlockObject(slack.PlainTextType, "Good Mood :heart:", true, false)
	goodMoodButton := slack.NewButtonBlockElement(fmt.Sprintf("mood_user_good_%d", timeNow), "good_mood", goodMoodButtonText)
	goodMoodButton.Style = slack.StylePrimary
	if goodMoodButtonText.Validate() != nil {
		log.Printf("WARNING goodMood button display failed: %s", goodMoodButtonText.Validate().Error())
		return nil
	}

	averageMoodButtonText := slack.NewTextBlockObject(slack.PlainTextType, "Meow :yellow_heart:", true, false)
	averageMoodButton := slack.NewButtonBlockElement(fmt.Sprintf("mood_user_avg_%d", timeNow), "average_mood", averageMoodButtonText)
	averageMoodButton.Style = slack.StyleDefault
	if averageMoodButtonText.Validate() != nil {
		log.Printf("WARNING averageMood button display failed: %s", averageMoodButtonText.Validate().Error())
		return nil
	}

	badMoodButtonText := slack.NewTextBlockObject(slack.PlainTextType, "Grr ! :black_heart:", true, false)
	badMoodButton := slack.NewButtonBlockElement(fmt.Sprintf("mood_user_bad_%d", timeNow), "bad_mood", badMoodButtonText)
	badMoodButton.Style = slack.StyleDanger
	if badMoodButtonText.Validate() != nil {
		log.Printf("WARNING baddMood button display failed: %s", badMoodButtonText.Validate().Error())
		return nil
	}

	return slack.NewActionBlock(actionBlockId, goodMoodButton, averageMoodButton, badMoodButton)
}

func fromJsonToBlocks(dbClient *gorm.DB, channelId string, previousBlock []slack.Block, firstPrint bool, slackBlocksChan chan []slack.Block) slack.Message {
	var blockMessage slack.Message = slack.NewBlockMessage(previousBlock...)
	if len(previousBlock) < 1 {
		authorName, slackFirstSection := firstSectionBlock()
		contextBlock := AddingContextAuthor(authorName)
		slackSecondSection := secondSectionBlock(dbClient, channelId)
		actions := actionSectionBlock()
		blockMessage.Blocks.BlockSet = append(blockMessage.Blocks.BlockSet, slackFirstSection, contextBlock, slackSecondSection, actions)
	}

	inputBlock := ContextInputText()

	actionId := fmt.Sprintf("mood_ctxt_cancel_%d", time.Now().UnixMilli())
	blockId := fmt.Sprintf("block_mood_ctxt_cancel_%d", time.Now().UnixMilli())
	buttonBlock := slack.NewButtonBlockElement(actionId, "cancel_"+channelId, slack.NewTextBlockObject(slack.PlainTextType, "Cancel", false, false))
	buttonBlock.WithStyle(slack.StyleDefault)
	actionBlock := slack.NewActionBlock(blockId, buttonBlock)

	blockMessage.Blocks.BlockSet = append(blockMessage.Blocks.BlockSet, slack.NewDividerBlock())
	blockMessage.Blocks.BlockSet = append(blockMessage.Blocks.BlockSet, inputBlock)
	blockMessage.Blocks.BlockSet = append(blockMessage.Blocks.BlockSet, actionBlock)

	slackBlocksChan <- blockMessage.Blocks.BlockSet

	return blockMessage
}

func ContextInputText() *slack.InputBlock {
	blockId := fmt.Sprintf("bloc_mood_ctxt_%d", time.Now().UnixMilli())
	actionId := fmt.Sprintf("mood_ctxt_%d", time.Now().UnixMilli())

	//If does not work use true as emoji
	inputBlockElem := slack.NewPlainTextInputBlockElement(slack.NewTextBlockObject(slack.PlainTextType, "Add Context", true, false), actionId)
	inputBlock := slack.NewInputBlock(blockId, slack.NewTextBlockObject(slack.PlainTextType, "Context", true, false), inputBlockElem)

	//Modifiers
	inputBlock.DispatchAction = false
	inputBlock.Optional = true
	inputBlock.Hint = slack.NewTextBlockObject(slack.PlainTextType, "Optionnal: enter some context", true, false)

	return inputBlock
}

func SendSlackBlocks(client *slack.Client, config *Config, dbClient *gorm.DB, previousBlocks []slack.Block, firstPrint bool, slackBlockChan chan []slack.Block) (string, error) {
	blockMessage := fromJsonToBlocks(dbClient, config.CHANNEL_ID, previousBlocks, firstPrint, slackBlockChan)
	_, threadTS, err := client.PostMessage(config.CHANNEL_ID, slack.MsgOptionBlocks(blockMessage.Blocks.BlockSet...))
	if err != nil {
		return "", err
	}
	return threadTS, nil
}

func UpdateMessage(client *slack.Client, config *Config, dbClient *gorm.DB, threadTS string, previousBlock []slack.Block, firstPrint bool) (string, error) {
	var slackMessage slack.Message
	slackMessage = fromJsonToBlocks(dbClient, config.CHANNEL_ID, previousBlock, firstPrint, config.SLACK_PREVIOUS_BLOCK)
	_, newThreadTS, _, err := client.UpdateMessage(config.CHANNEL_ID, threadTS, slack.MsgOptionBlocks(slackMessage.Blocks.BlockSet...))
	if err != nil {
		return threadTS, err
	}
	return newThreadTS, nil
}

func FetchUsersFromChannel(slackClient *slack.Client, channelId string) (*slack.Channel, []*slack.User, error) {
	slackChannel, err := slackClient.GetConversationInfo(channelId, true)
	if err != nil {
		return nil, nil, err
	}

	channelMembers, _, err := slackClient.GetUsersInConversation(&slack.GetUsersInConversationParameters{ChannelID: channelId})
	if err != nil {
		return nil, nil, err
	}

	slackChannelMembers := make([]*slack.User, len(channelMembers))
	for idx, userId := range channelMembers {
		slackUserInfo, err := slackClient.GetUserInfo(userId)
		if err != nil {
			log.Printf("Failed fetch User(%d) : %s", idx, err.Error())
			return slackChannel, slackChannelMembers, err
		}
		slackChannelMembers[idx] = slackUserInfo
	}
	return slackChannel, slackChannelMembers, nil
}
