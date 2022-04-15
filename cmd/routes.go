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
			c.Error(fmt.Errorf("#slack.URLVerification parsing: %s", err.Error()))
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
				log.Printf("[ERROR] response => %+v", viewResponse.ResponseMetadata.Messages)
				log.Printf("[ERROR] responseError => %s", viewResponse.Err().Error())
				return err
			}
		case *slackevents.AppMentionEvent:
			slackClient.PostMessage(ev.Channel, slack.MsgOptionText("Meow :cat:", false))
		}
	}

	return nil
}

func handleRouteInteractive(c echo.Context, slackClient *slack.Client, config *simba.Config, dbClient *gorm.DB, threadTS string) error {
	callBackStruct := new(slack.InteractionCallback)
	err := json.Unmarshal([]byte(c.Request().FormValue("payload")), &callBackStruct)

	if err != nil {
		c.Logger().Errorf("Error from FormValue.payload in callbackStruct = %s", err.Error())
		return err
	} else if modalValue := callBackStruct.View.State; modalValue != nil && len(modalValue.Values) > 0 {
		if modalValue.Values["MoodContext"]["mood_ctxt"].Value != "" {
			contextString := modalValue.Values["MoodContext"]["mood_ctxt"].Value
			privateMetadata := callBackStruct.View.PrivateMetadata
			moodIdSplit := strings.Split(privateMetadata, "::")
			_, err := simba.UpdateMoodById(dbClient, moodIdSplit[1], nil, &contextString)
			if err != nil {
				return err
			}
			threadTS, err := simba.UpdateMessage(slackClient, config, dbClient, threadTS)
			if err != nil {
				return err
			}
			config.SLACK_MESSAGE_CHANNEL <- threadTS
			return nil
		}
	}

	if len(callBackStruct.ActionCallback.BlockActions) > 0 {
		blockActions := callBackStruct.ActionCallback.BlockActions
		user := callBackStruct.User
		channelId := callBackStruct.Channel.ID
		userId := user.ID
		profile, err := slackClient.GetUserProfile(&slack.GetUserProfileParameters{UserID: userId})

		var username string
		if err != nil {
			username = "John Snow"
			c.Logger().Error("[ERROR] #getUserProfile => %s", err.Error())
			simba.SendErrorMessageToUser(slackClient, userId, err)
		} else if profile.DisplayName != "" {
			username = profile.DisplayName
		} else if profile.RealName != "" {
			username = profile.RealName
		}

		for _, action := range blockActions {
			log.Println("ActionBlock", action.ActionID, action.Value)
			switch {
			case strings.Contains(action.ActionID, "mood_feeling_select"):
				c.Logger().Printf("Clicked on button for mood_feeling_select with value = %s", action.Value)

				if simbaUser, _, err := simba.FechCurrent(dbClient, slackClient, userId); err != nil {
					simba.SendErrorMessageToUser(slackClient, userId, err)
					return err
				} else if dailyMood, err := simba.FetchMoodFromThreadTS(dbClient, threadTS, simbaUser.ID); err != nil {
					simba.SendErrorMessageToUser(slackClient, userId, err)
					return err
				} else if _, err = simba.UpdateMood(dbClient, dailyMood, &action.Value, nil); err != nil {
					simba.SendErrorMessageToUser(slackClient, userId, err)
					return err
				}

				go simba.UpdateMessage(slackClient, config, dbClient, threadTS)

				return nil
			case strings.Contains(action.ActionID, "mood_user"):
				dailyMood, err := simba.HandleAddDailyMood(dbClient, slackClient, channelId, userId, username, action.Value, threadTS)
				if err != nil {
					c.Error(err)
					simba.SendErrorMessageToUser(slackClient, userId, err)
					return err
				}
				viewModal := viewAppModalMood(userId, username, action.Value, dailyMood.ID)
				viewResponse, err := slackClient.OpenView(callBackStruct.TriggerID, viewModal)
				if err != nil {
					c.Logger().Errorf("Failed open modal view %s", err.Error())
					c.Logger().Errorf("MetadataError %v", viewResponse.ResponseMetadata.Messages)
				}

				threadTS, err := simba.UpdateMessage(slackClient, config, dbClient, threadTS)
				if err != nil {
					simba.SendErrorMessageToUser(slackClient, userId, err)
					return err
				}
				config.SLACK_MESSAGE_CHANNEL <- threadTS

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
				err := simba.NewErrNoActionFound(action.ActionID, action.Value)
				simba.SendErrorMessageToUser(slackClient, userId, err)
				return err
			}
		}

	} else {
		return fmt.Errorf("nothing has been received when clicking the button")
	}

	return nil
}
