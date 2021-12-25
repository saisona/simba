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

func slackMkDownBlock(text string) *slack.TextBlockObject {
	textObject := slack.NewTextBlockObject(slack.MarkdownType, text, false, false)
	if err := textObject.Validate(); err != nil {
		log.Printf("Validation failed for %s", text)
		panic(err)
	}
	return textObject
}

func slackTextBlock(text string) *slack.TextBlockObject {
	textObject := slack.NewTextBlockObject(slack.PlainTextType, text, true, false)
	if err := textObject.Validate(); err != nil {
		log.Printf("Validation failed for %s", text)
		panic(err)
	}
	return textObject
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
	firstLine := slackMkDownBlock(quoteOfTheDay)
	sectionBlock := slack.NewSectionBlock(firstLine, nil, nil)
	return author, sectionBlock
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

func drawResults(userWithDailyMoods []*User) ([]slack.Block, error) {
	blockMessageArray := []slack.Block{}
	for _, u := range userWithDailyMoods {
		if len(u.Moods) == 0 {
			continue
		}
		firstField := slackTextBlock(u.Username)
		if firstField.Validate() != nil {
			return blockMessageArray, fmt.Errorf("#drawResults::firstField = %s", firstField.Validate().Error())
		}

		userMood := u.Moods[0].Mood
		userFeeling := u.Moods[0].Feeling
		fields := []*slack.TextBlockObject{firstField}
		if userFeeling != "" {
			secondField := slackTextBlock(fmt.Sprintf("%s %s %s", fromMoodToSmiley(userMood), fromFeelingToSmiley(userFeeling), userFeeling))
			if secondField.Validate() != nil {
				return blockMessageArray, fmt.Errorf("#drawResults::second has no feeling= %s", secondField.Validate().Error())
			}

			fields = append(fields, secondField)
		} else {
			secondField := slackTextBlock(fmt.Sprintf("%s %s", fromMoodToSmiley(userMood), strings.ToUpper(strings.ReplaceAll(userMood, "_", " "))))
			if secondField.Validate() != nil {
				return blockMessageArray, fmt.Errorf("#drawResults::second hasFeeling= %s", secondField.Validate().Error())
			}

			fields = append(fields, secondField)
		}
		section := slack.NewSectionBlock(nil, fields, nil)
		blockMessageArray = append(blockMessageArray, section)

		if u.Moods[0].Context != "" {
			moodContext := slackTextBlock(u.Moods[0].Context)
			if moodContext.Validate() != nil {
				return blockMessageArray, fmt.Errorf("#drawResults::moodContext = %s", moodContext.Validate().Error())
			}
			context := slack.NewContextBlock(fmt.Sprintf("context_%d", u.ID), moodContext)
			blockMessageArray = append(blockMessageArray, context)
		}

	}
	return blockMessageArray, nil
}

func fromJsonToBlocks(dbClient *gorm.DB, channelId, threadTS string, firstPrint bool) slack.Message {
	var blockMessage slack.Message = slack.NewBlockMessage()
	authorName, slackFirstSection := firstSectionBlock()
	contextBlock := AddingContextAuthor(authorName)
	actions := actionSectionBlock()
	blockMessage.Blocks.BlockSet = append(blockMessage.Blocks.BlockSet, slackFirstSection, contextBlock, actions)
	if !firstPrint {
		blockMessage.Blocks.BlockSet = append(blockMessage.Blocks.BlockSet, slack.NewDividerBlock())
		userWithDailyMoods, err := FetchAllDailyMoodsByThreadTS(dbClient, threadTS)
		if err != nil {
			panic(err)
		}

		blockMessageArray, err := drawResults(userWithDailyMoods)
		if err != nil {
			log.Printf("[ERROR] drawResults : %s", err.Error())
			panic(err)
		}

		blockMessage.Blocks.BlockSet = append(blockMessage.Blocks.BlockSet, blockMessageArray...)
	}
	return blockMessage
}

func fromMoodToSmiley(mood string) string {
	switch mood {
	case "good_mood":
		return ":heart:"
	case "average_mood":
		return ":yellow_heart:"
	case "bad_mood":
		return ":black_heart:"
	default:
		return ":meow:"
	}
}

func fromFeelingToSmiley(feeling string) string {
	switch feeling {
	case "Excited":
		return ":star-struck:"
	case "Happy":
		return ":smile:"
	case "Chilling":
		return ":relaxed:"
	case "Neutral":
		return ":expressionless:"
	case "Frustrated":
		return ":face_with_rolling_eyes:"
	case "Tired":
		return ":yawning_face:"
	case "Sad":
		return ":cry:"
	case "Mad":
		return ":triumph:"
	case "Disappointed":
		return ":disappointed:"
	default:
		return ":meow:"
	}
}

func ContextInputText() *slack.InputBlock {
	blockId := "MoodContext"
	actionId := "mood_ctxt"

	//If does not work use true as emoji
	inputBlockElem := slack.NewPlainTextInputBlockElement(slackTextBlock("Context"), actionId)
	inputBlock := slack.NewInputBlock(blockId, slackTextBlock("Context"), inputBlockElem)

	//Modifiers
	inputBlock.DispatchAction = false
	inputBlock.Optional = true
	inputBlock.Hint = slack.NewTextBlockObject(slack.PlainTextType, "To add a bit of context", true, false)

	return inputBlock
}

func SendSlackBlocks(client *slack.Client, config *Config, dbClient *gorm.DB, threadTS string, firstPrint bool) (string, error) {
	blockMessage := fromJsonToBlocks(dbClient, config.CHANNEL_ID, threadTS, firstPrint)
	_, threadTS, err := client.PostMessage(config.CHANNEL_ID, slack.MsgOptionBlocks(blockMessage.Blocks.BlockSet...))
	if err != nil {
		return "", err
	}
	return threadTS, nil
}

func UpdateMessage(client *slack.Client, config *Config, dbClient *gorm.DB, threadTS string, firstPrint bool) (string, error) {
	slackMessage := fromJsonToBlocks(dbClient, config.CHANNEL_ID, threadTS, firstPrint)
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
