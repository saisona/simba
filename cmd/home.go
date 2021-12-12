package main

import (
	"fmt"

	"github.com/saisona/simba"
	"github.com/slack-go/slack"
)

//@desc Render Home view admin or update depending on slackChannelId is given or not
//@params user is a DB representation of a Simba user
//@params [slackChannelId] is optionnal given if already known or not used for update
//@returns Blocks to be send to update Simba Home view
func handleAppHomeViewAdmin(user *simba.User, config *simba.Config, slackChannelId string) slack.Blocks {
	basicText := slackTextBlock("Simba Application (Admin)")
	slackHeaderBlock := slack.NewHeaderBlock(basicText)
	slackChannelActionId := fmt.Sprintf("channel_selected_%s", user.SlackUserID)
	slackChannelSelect := slack.NewOptionsSelectBlockElement(slack.OptTypeChannels, slackTextBlock("Selected Channel"), slackChannelActionId)

	if slackChannelId == "" {
		//TeamInfo
		slackChannelSelect.InitialChannel = config.CHANNEL_ID
	} else {
		slackChannelSelect.InitialChannel = slackChannelId
	}

	slackSection := slack.NewSectionBlock(slackTextBlock("Select channel"), nil, slack.NewAccessory(slackChannelSelect))

	blockSet := []slack.Block{slackHeaderBlock, slack.NewDividerBlock(), slackSection}

	return slack.Blocks{
		BlockSet: blockSet,
	}
}

func handleAppHomeViewNotAdmin(user *simba.User) slack.Blocks {
	//Header
	basicText := slackTextBlock("Simba Application (Not Admin)")
	slackHeaderBlock := slack.NewHeaderBlock(basicText)

	return slack.Blocks{
		BlockSet: []slack.Block{slackHeaderBlock, slack.NewDividerBlock()},
	}

}
