package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/saisona/simba"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"gorm.io/gorm"
)

func secretVerifier(c echo.Context, body []byte, slackSigningSecret string) error {
	sv, err := slack.NewSecretsVerifier(c.Request().Header, slackSigningSecret)
	if err != nil {
		c.NoContent(http.StatusBadRequest)
		c.Logger().Errorf("#slack.NewSecretsVerifier : %s", err.Error())
		return err
	}
	if _, err := sv.Write(body); err != nil {
		c.NoContent(http.StatusInternalServerError)
		c.Logger().Errorf("#slack.SendBackBody : %s", err.Error())
		return err
	}
	if err := sv.Ensure(); err != nil {
		c.NoContent(http.StatusUnauthorized)
		return err
	}
	return nil
}

func handleRouteEvents(c echo.Context, slackClient *slack.Client, dbClient *gorm.DB, config *simba.Config, slackSigningSecret string) error {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		c.NoContent(http.StatusBadRequest)
		return err
	} else if err := secretVerifier(c, body, slackSigningSecret); err != nil {
		return err
	}

	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		c.NoContent(http.StatusInternalServerError)
		c.Logger().Errorf("#slack.ParseEventToken: %s", err.Error())
		return err
	}

	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			c.NoContent(http.StatusInternalServerError)
			c.Logger().Errorf("#slack.URLVerification parsing: %s", err.Error())
			return err
		}
		c.String(http.StatusOK, r.Challenge)
	}

	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppHomeOpenedEvent:
			viewResponse, err := slackClient.PublishView(ev.User, handleAppHomeView(slackClient, dbClient, config, ev.User), "")
			if err != nil {
				c.Logger().Errorf("PublishView AppHomeOpenedEvent = %s", err.Error())
				log.Printf("[ERROR] response => %+v", viewResponse)
				log.Printf("[ERROR] responseError => %s", viewResponse.Err().Error())
				return err
			}
		case *slackevents.AppMentionEvent:
			slackClient.PostMessage(ev.Channel, slack.MsgOptionText("Meow :cat:", false))
		}
	}

	return nil
}

func handleRouteInteractive(c echo.Context, slackClient *slack.Client, config *simba.Config, dbClient *gorm.DB, threadTS string, previousBlocks []slack.Block) error {
	callBackStruct := new(slack.InteractionCallback)
	err := json.Unmarshal([]byte(c.Request().FormValue("payload")), &callBackStruct)

	if err != nil {
		return err
	} else if len(callBackStruct.ActionCallback.BlockActions) > 0 {
		blockActions := callBackStruct.ActionCallback.BlockActions
		user := callBackStruct.User
		channelId := callBackStruct.Channel.ID
		userId := user.ID
		profile, err := slackClient.GetUserProfile(&slack.GetUserProfileParameters{UserID: userId})

		var username string
		if err != nil {
			username = "John Snow"
			c.Logger().Error("[ERROR] #getUserProfile => %s", err.Error())
		} else {
			username = profile.DisplayName
		}

		for _, action := range blockActions {
			switch {
			case strings.Contains(action.ActionID, "mood_feeling_select"):
				go handleUpdateMood(config, slackClient, dbClient, threadTS, channelId, userId, username, action)
			case strings.Contains(action.ActionID, "mood_ctxt_cancel"):
				newThreadTS, err := simba.UpdateMessage(slackClient, config, dbClient, threadTS, true)
				if err != nil {
					return err
				}
				config.SLACK_MESSAGE_CHANNEL <- newThreadTS
			case strings.Contains(action.ActionID, "mood_ctxt"):
				c.Logger().Printf("user %s added context : %s", userId, action.Value)
				slackMessage := fmt.Sprintf("<@%s> added %s", userId, action.Value)
				simba.SendSlackTSMessage(slackClient, config, slackMessage, threadTS)
			case strings.Contains(action.ActionID, "mood_user"):
				viewModal := viewAppModalMood(userId, username, action.Value)
				viewResponse, err := slackClient.OpenView(callBackStruct.TriggerID, viewModal)
				if err != nil {
					c.Logger().Errorf("Failed open modal view %s", err.Error())
					c.Logger().Errorf("MetadataError %v", viewResponse.ResponseMetadata.Messages)
				}
			case strings.Contains(action.ActionID, "send_kind_message"):
				go handleSendKindMessage(slackClient, userId, action)
			case strings.Contains(action.ActionID, "channel_selected"):
				c.Logger().Printf("Enter channelSelected => %s", action.SelectedChannel)
				_, users, err := simba.FetchUsersFromChannel(slackClient, action.SelectedChannel)

				if err != nil {
					c.Logger().Error(err)
					return err
				}
				viewResponse, err := slackClient.PublishView(userId, handleAppHomeViewUpdated(slackClient, dbClient, config, userId, action.SelectedChannel, users), "")
				if err != nil {
					c.Logger().Error(err)
					c.Logger().Error(viewResponse.Err())
					return err
				}
			default:
				err := fmt.Errorf("ActionId %s (Value:%s/SelectedChannel:%s) is not registered (default case)", action.ActionID, action.Value, action.SelectedChannel)
				c.Logger().Error(err)
				return err
			}
		}
	} else {
		return fmt.Errorf("nothing has been received when clicking the button")
	}

	return nil
}