package main

import (
	"fmt"
	"log"

	"github.com/saisona/simba"
	"github.com/slack-go/slack"
)

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

func viewAppModalMood(userId, username, mood string, dailyMoodId uint) slack.ModalViewRequest {

	blockActionId := "MoodFeeling"
	var feelingButtonList []slack.BlockElement

	switch mood {
	case "good_mood":
		excitedButton := slack.NewButtonBlockElement("excited_mood_feeling_select", "Excited", slackTextBlock("Excited :star-struck:"))
		happyButton := slack.NewButtonBlockElement("happy_mood_feeling_select", "Happy", slackTextBlock("Happy :smile:"))
		chillingButton := slack.NewButtonBlockElement("chilling_mood_feeling_select", "Chilling", slackTextBlock("Chilling :relaxed:"))
		feelingButtonList = []slack.BlockElement{excitedButton, happyButton, chillingButton}
	case "average_mood":
		neutral := slack.NewButtonBlockElement("neutral_mood_feeling_select", "Neutral", slackTextBlock("Neutral :expressionless:"))
		frustratedButton := slack.NewButtonBlockElement("frustrated_mood_feeling_select", "Frustrated", slackTextBlock("Frustrated :face_with_rolling_eyes:"))
		tiredButton := slack.NewButtonBlockElement("tired_mood_feeling_select", "Tired", slackTextBlock("Tired :yawning_face:"))
		feelingButtonList = []slack.BlockElement{neutral, frustratedButton, tiredButton}
	case "bad_mood":
		sadButton := slack.NewButtonBlockElement("sad_mood_feeling_select", "Sad", slackTextBlock("Sad :cry:"))
		triumphButton := slack.NewButtonBlockElement("triumph_mood_feeling_select", "Mad", slackTextBlock("Mad :triumph:"))
		disappointedButton := slack.NewButtonBlockElement("disappointed_mood_feeling_select", "Disappointed", slackTextBlock("Disappointed :disappointed:"))
		feelingButtonList = []slack.BlockElement{sadButton, triumphButton, disappointedButton}
	default:
		log.Printf("[ERROR] entered in default case about mood")
	}

	blockAction := slack.NewActionBlock(blockActionId, feelingButtonList...)
	inputBlock := simba.ContextInputText()

	blockSet := []slack.Block{blockAction, inputBlock}

	slackBlocks := slack.Blocks{BlockSet: blockSet}

	return slack.ModalViewRequest{
		Type:            slack.VTModal,
		Blocks:          slackBlocks,
		Title:           slackTextBlock("What's your mood"),
		Close:           slackTextBlock("Cancel"),
		Submit:          slackTextBlock("Share"),
		CallbackID:      "mood_modal_sharing",
		PrivateMetadata: fmt.Sprintf("daily_mood_id::%d", dailyMoodId),
		ClearOnClose:    true,
	}
}
