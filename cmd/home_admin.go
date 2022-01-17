package main

import (
	"github.com/saisona/simba"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

//@desc Render Home view admin or update depending on slackChannelId is given or not
//@params user is a DB representation of a Simba user
//@params [slackChannelId] is optionnal given if already known or not used for update
//@returns Blocks to be send to update Simba Home view
func handleAppHomeViewAdmin(user *simba.User, config *simba.Config, dbClient *gorm.DB, slackChannelId string) slack.Blocks {
	basicText := slackTextBlock("Simba Application (Admin)")
	slackHeaderBlock := slack.NewHeaderBlock(basicText)

	blockSet := []slack.Block{slackHeaderBlock, slack.NewDividerBlock()}

	return slack.Blocks{
		BlockSet: blockSet,
	}
}

func generateSelectOptions(dbClient *gorm.DB) ([]*slack.DialogSelectOption, error) {
	var users []*simba.User
	tx := dbClient.Model(&simba.User{}).Find(&users)
	if tx.Error != nil {
		return nil, tx.Error
	}

	dialogOptions := make([]*slack.DialogSelectOption, len(users))
	for idx, u := range users {
		tmpUserSlackId := u.SlackUserID
		dialogOptions[idx] = &slack.DialogSelectOption{Label: tmpUserSlackId, Value: tmpUserSlackId}
	}

	return dialogOptions, nil
}
