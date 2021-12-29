package main

import (
	"github.com/saisona/simba"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

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
