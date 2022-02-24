package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/saisona/simba"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

type homeViewInfo struct {
	SlackUserId string
	WeeklyMoods []simba.DailyMood
}

func NewHomeViewInfo(dbClient *gorm.DB, slackUserId, simbaUserId string) (*homeViewInfo, error) {
	hvi := &homeViewInfo{
		SlackUserId: slackUserId,
		WeeklyMoods: make([]simba.DailyMood, 7),
	}
	if err := hvi.fetchWeeklyMoods(dbClient, simbaUserId); err != nil {
		return nil, err
	}
	return hvi, nil
}

func (hvi *homeViewInfo) fetchWeeklyMoods(dbClient *gorm.DB, simbaUserId string) error {
	var weeklyMoods []simba.DailyMood
	if tx := dbClient.Debug().Where("user_id=?", simbaUserId).Limit(7).Order("created_at DESC").Find(&weeklyMoods); tx.Error != nil {
		return tx.Error
	}

	hvi.WeeklyMoods = weeklyMoods
	return nil
}

func (hvi homeViewInfo) mapCount() map[string]int {
	var moodCountMap map[string]int = map[string]int{}
	for _, k := range hvi.WeeklyMoods {
		moodCountMap[k.Mood] += 1
	}
	return moodCountMap
}

//@desc Render Home view not admin or update depending on slackChannelId is given or not
//@params user is a DB representation of a Simba user
//@params [slackChannelId] is optionnal given if already known or not used for update
//@returns Blocks to be send to update Simba Home view
func handleAppHomeViewNotAdmin(user *simba.User, config *simba.Config, dbClient *gorm.DB) slack.Blocks {
	//Header
	basicText := slackTextBlock("Simba Application (Not Admin)")
	hvi, err := NewHomeViewInfo(dbClient, user.SlackUserID, fmt.Sprint(user.ID))
	if err != nil {
		panic(err)
	}
	slackHeaderBlock := slack.NewHeaderBlock(basicText)

	slackTitleInfo := slackTextBlock("Week informations about you")
	blockSet := []slack.Block{slackHeaderBlock, slack.NewDividerBlock(), slackTitleInfo}
	for mood, count := range hvi.mapCount() {
		txtSlack := fmt.Sprintf("You've been %d in %s", count, strings.ReplaceAll(mood, "_", " "))
		blockSet = append(blockSet, slackTextBlock(txtSlack))
	}

	return slack.Blocks{
		BlockSet: blockSet,
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
		blocks = handleAppHomeViewAdmin(user, config, dbClient)
	} else {
		blocks = handleAppHomeViewNotAdmin(user, config, dbClient)
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
	var blocks slack.Blocks = handleAppHomeViewAdmin(user, config, dbClient)

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
