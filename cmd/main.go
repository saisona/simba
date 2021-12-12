/**
 * File              : main.go
 * Author            : Alexandre Saison <alexandre.saison@inarix.com>
 * Date              : 08.11.2021
 * Last Modified Date: 16.11.2021
 * Last Modified By  : Alexandre Saison <alexandre.saison@inarix.com>
 */

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/saisona/simba"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{LogLevel: 2}))

	slackSigningSecret, ok := os.LookupEnv("SLACK_SIGNING_SECRET")
	if !ok || slackSigningSecret == "" {
		e.Logger.Fatalf("SLACK_SIGNING_SECRET does not exists and must be provided")
	}

	hasMigrationStr, exists := os.LookupEnv("APP_MIGRATE")
	var hasMigration bool = exists

	if exists && hasMigrationStr != "" {
		var err error
		if hasMigration, err = strconv.ParseBool(hasMigrationStr); err != nil {
			e.Logger.Warnf("APP_MIGRATE(%s) has been set but cannot be converted to bool : %s ", hasMigrationStr, err.Error())
		}
	}

	config, err := simba.InitConfig()
	dbClient := simba.InitDbClient(config.DB.Host, config.DB.Username, config.DB.Password, config.DB.Name, hasMigration)
	if err != nil {
		e.Logger.Fatalf("Failed initConfig: %s", err.Error())
	}

	slackClient := slack.New(config.SLACK_API_TOKEN, slack.OptionDebug(true), slack.OptionLog(log.Default()))
	scheduler, _, err := simba.InitScheduler(dbClient, slackClient, config)
	if err != nil {
		e.Logger.Fatalf("Failed launching server: %s", err.Error())
	}

	scheduler.StartAsync()

	var threadTS string

	// call anonymous goroutine
	go watchValueChanged(&threadTS, config.SLACK_MESSAGE_CHANNEL, e.Logger)

	e.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	e.POST("/events", func(c echo.Context) error {
		body, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			c.NoContent(http.StatusBadRequest)
			return err
		}
		sv, err := slack.NewSecretsVerifier(c.Request().Header, slackSigningSecret)
		if err != nil {
			c.NoContent(http.StatusBadRequest)
			return err
		}
		if _, err := sv.Write(body); err != nil {
			c.NoContent(http.StatusInternalServerError)
			return err
		}
		if err := sv.Ensure(); err != nil {
			c.NoContent(http.StatusUnauthorized)
			return err

		}

		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			c.NoContent(http.StatusInternalServerError)
			return err
		}

		if eventsAPIEvent.Type == slackevents.URLVerification {
			var r *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(body), &r)
			if err != nil {
				c.NoContent(http.StatusInternalServerError)
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
					log.Printf("Error on launching Home: %s\n", err.Error())
					log.Printf("[ERROR] response => %+v", viewResponse)
					return err
				}
				if viewResponse.Err() != nil {
					log.Printf("[ERROR] err = %s", viewResponse.Err().Error())
				}
				responseMetaMsg := viewResponse.ResponseMetadata.Messages
				if len(responseMetaMsg) > 0 {
					log.Printf("responseMetaMsg=%+v", responseMetaMsg)
				}

			case *slackevents.AppMentionEvent:
				slackClient.PostMessage(ev.Channel, slack.MsgOptionText("Meow :cat:", false))
			}
		}

		return nil
	})

	e.POST("/interactive", func(c echo.Context) error {
		callBackStruct := new(slack.InteractionCallback)
		err := json.Unmarshal([]byte(c.Request().FormValue("payload")), &callBackStruct)
		log.Println("CallbackStruct = ", callBackStruct)

		if err != nil {
			return err
		} else if len(callBackStruct.ActionCallback.AttachmentActions) > 0 {
			blockAttachmentActions := callBackStruct.ActionCallback.AttachmentActions
			log.Printf("There is %d block actions", len(blockAttachmentActions))
			for _, action := range blockAttachmentActions {
				log.Printf("AttachementValue = %s", action.Value)
			}
		} else if len(callBackStruct.ActionCallback.BlockActions) > 0 {
			blockActions := callBackStruct.ActionCallback.BlockActions
			user := callBackStruct.User
			channelId := callBackStruct.Channel.ID
			userId := user.ID
			profile, err := slackClient.GetUserProfile(&slack.GetUserProfileParameters{UserID: userId})

			var username string
			if err != nil {
				log.Printf("Warning some error while fetchingProfile:  %s ", err.Error())
				username = "John Snow"
				c.Logger().Error("[ERROR] #getUserProfile => %s", err.Error())
			} else {
				username = profile.DisplayName
			}

			for _, action := range blockActions {
				switch {
				case strings.Contains(action.ActionID, "mood_ctxt"):
					c.Logger().Debugf("Enter in %s with value = %s", action.ActionID, action.Value)
				case strings.Contains(action.ActionID, "mood_user"):
					go handleUpdateMood(config, slackClient, dbClient, threadTS, channelId, userId, username, action)
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
			return fmt.Errorf("Nothing has been received when clicking the button")
		}

		return nil
	})

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("slackClient", slackClient)
			return next(c)
		}
	})

	defer close(config.SLACK_MESSAGE_CHANNEL)
	port := fmt.Sprintf(":%s", config.APP_PORT)
	if err := e.Start(port); err != nil {
		log.Fatalf("Error when launching server : %s", err.Error())
		return
	}
}
