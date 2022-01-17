package main

import (
	"fmt"
	"log"
	"time"

	"github.com/saisona/simba"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

//@desc Render Home view not admin or update depending on slackChannelId is given or not
//@params user is a DB representation of a Simba user
//@params [slackChannelId] is optionnal given if already known or not used for update
//@returns Blocks to be send to update Simba Home view
func handleAppHomeViewNotAdmin(user *simba.User) slack.Blocks {
	//Header
	basicText := slackTextBlock("Simba Application (Not Admin)")
	slackHeaderBlock := slack.NewHeaderBlock(basicText)

	return slack.Blocks{
		BlockSet: []slack.Block{slackHeaderBlock, slack.NewDividerBlock()},
	}

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
		blocks = handleAppHomeViewAdmin(user, config, dbClient, "")
	} else {
		blocks = handleAppHomeViewNotAdmin(user)
	}

	slackModalViewRequest := slack.HomeTabViewRequest{
		Type:       slack.VTHomeTab,
		CallbackID: callbackId,
		ExternalID: externalId,
		Blocks:     blocks,
	}

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
	var blocks slack.Blocks = handleAppHomeViewAdmin(user, config, dbClient, channelId)

	for _, user := range channelUsers {
		userProfile, err := slackClient.GetUserProfile(&slack.GetUserProfileParameters{UserID: userId})
		if err != nil {
			log.Printf("Cannot fetch UserProfile : %s", err.Error())
			return slack.HomeTabViewRequest{
				Type:       slack.VTHomeTab,
				CallbackID: callbackId,
				ExternalID: externalId,
				Blocks:     slack.Blocks{BlockSet: []slack.Block{}},
			}
		}
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
