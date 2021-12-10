package main

import (
	"fmt"
	"log"
	"time"

	"github.com/saisona/simba"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
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

func handleAppHomeViewAdmin(userId string) slack.Blocks {
	basicText := slackTextBlock("Simba Application (Admin)")
	slackHeaderBlock := slack.NewHeaderBlock(basicText)
	return slack.Blocks{
		BlockSet: []slack.Block{slackHeaderBlock, slack.NewDividerBlock()},
	}
}

func handleAppHomeViewNotAdmin(userId string) slack.Blocks {
	basicText := slackTextBlock("Simba Application (Not Admin)")
	slackHeaderBlock := slack.NewHeaderBlock(basicText)
	return slack.Blocks{
		BlockSet: []slack.Block{slackHeaderBlock},
	}
}

func isUserAdmin(dbClient *gorm.DB, userId string) (bool, error) {
	var user *simba.User
	fetchUserTx := dbClient.Find(&user, "slack_user_id = ?", userId)
	if fetchUserTx.Error != nil {
		return false, fetchUserTx.Error
	}
	log.Printf("[DEBUG] User => %+v", user)

	return user.IsManager, nil
}

func handleAppHomeView(dbClient *gorm.DB, userId string) slack.HomeTabViewRequest {
	callbackId := fmt.Sprintf("app_home_callback_%d", time.Now().UnixMilli())
	externalId := fmt.Sprintf("app_home_external_%d", time.Now().UnixMilli())
	isManager, err := isUserAdmin(dbClient, userId)
	if err != nil {
		log.Printf("Error during OpenView to fetch Admin = %s", err.Error())
	}
	var blocks slack.Blocks
	if isManager {
		blocks = handleAppHomeViewAdmin(userId)
	} else {
		blocks = handleAppHomeViewNotAdmin(userId)
	}

	slackModalViewRequest := slack.HomeTabViewRequest{
		Type:       slack.VTHomeTab,
		CallbackID: callbackId,
		ExternalID: externalId,
		Blocks:     blocks,
	}

	log.Printf("[DEBUG] slackModalViewRequest=%+v\n", slackModalViewRequest)
	return slackModalViewRequest
}
