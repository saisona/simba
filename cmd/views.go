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

func handleAppHomeView(slackClient *slack.Client, dbClient *gorm.DB, config *simba.Config, userId string) slack.HomeTabViewRequest {
	callbackId := fmt.Sprintf("app_home_callback_%d", time.Now().UnixMilli())
	externalId := fmt.Sprintf("app_home_external_%d", time.Now().UnixMilli())
	user, slackUser, err := simba.FechCurrent(dbClient, slackClient, userId)
	if err != nil {
		log.Printf("Error during OpenView to fetch Admin = %s", err.Error())
	}
	var blocks slack.Blocks
	if user.IsManager || slackUser.IsAdmin {
		blocks = handleAppHomeViewAdmin(user, config, "")
	} else {
		blocks = handleAppHomeViewNotAdmin(user)
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

func handleAppHomeViewUpdated(slackClient *slack.Client, dbClient *gorm.DB, config *simba.Config, userId string, channelId string, channelUsers []*slack.User) slack.HomeTabViewRequest {
	callbackId := fmt.Sprintf("app_home_callback_%d", time.Now().UnixMilli())
	externalId := fmt.Sprintf("app_home_external_%d", time.Now().UnixMilli())
	user, _, err := simba.FechCurrent(dbClient, slackClient, userId)
	if err != nil {
		log.Printf("Error during OpenView to fetch Admin = %s", err.Error())
		return slack.HomeTabViewRequest{}
	}
	var blocks slack.Blocks
	blocks = handleAppHomeViewAdmin(user, config, channelId)
	userProfile, err := slackClient.GetUserProfile(&slack.GetUserProfileParameters{UserID: userId})
	if err != nil {
		log.Printf("Cannot fetch UserProfile : %s", err.Error())
		return slack.HomeTabViewRequest{}
	}

	for _, user := range channelUsers {
		userListName := slackMkDownBlock(fmt.Sprintf("*%s*", userProfile.DisplayName))
		msgAction := slack.NewAccessory(slack.NewButtonBlockElement(fmt.Sprintf("direct_message_%s", user.ID), user.ID, slackTextBlock("Send IM")))
		userListItem := slack.NewSectionBlock(userListName, nil, msgAction)
		blocks.BlockSet = append(blocks.BlockSet, userListItem)
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
