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
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/saisona/simba"
	"github.com/slack-go/slack"
)

func watchValueChanged(valueToChange *string, channel chan string) {
	for {
		// j = receipt from jobs channel
		// more = bool if channel has been closed
		val, more := <-channel
		if more {
			fmt.Println("received new value for ThreadTS=", val)
			if valueToChange != nil {
				fmt.Printf("OldValue=%s", *valueToChange)
				*valueToChange = val
				fmt.Printf("NewValue=%s", *valueToChange)
			} else {
			}
		} else {
			fmt.Println("received all jobs")
			return
		}
	}
}

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{LogLevel: 2}))
	//datadogClient := *datadog.NewAPIClient(&datadog.Configuration{Host: "datadoghq.eu", Debug: true})
	hasMigrationStr := os.Getenv("APP_MIGRATE")
	var hasMigration bool

	if hasMigrationStr != "" {
		var err error
		if hasMigration, err = strconv.ParseBool(hasMigrationStr); err != nil {
			log.Printf("Warning ! APP_MIGRATE(%s) has been set but cannot be converted to bool : %s ", hasMigrationStr, err.Error())
		}
	} else {
		hasMigration = false
	}

	config, err := simba.InitConfig()
	dbClient := simba.InitDbClient(config.DB.Host, config.DB.Username, config.DB.Password, config.DB.Name, hasMigration)
	if err != nil {
		log.Fatalf("Failed initConfig: %s", err.Error())
	}

	slackClient := slack.New(config.SLACK_API_TOKEN, slack.OptionDebug(true), slack.OptionLog(log.Default()))
	scheduler, job, err := simba.InitScheduler(dbClient, slackClient, config)
	scheduler.StartAsync()

	var threadTS string

	// call anonymous goroutine
	go watchValueChanged(&threadTS, config.SLACK_MESSAGE_CHANNEL)

	if err != nil {
		log.Fatalf("Failed launching server: %s", err.Error())
	}

	jErr := job.Error()
	if jErr != nil {
		log.Fatalf("Job has failed: %s", jErr.Error())
	}

	e.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	e.POST("/events", func(c echo.Context) error {
		var slackVerificationToken simba.SlackVerificationStruct
		if err := c.Bind(&slackVerificationToken); err != nil {
			return err
		}
		log.Printf("slackVerificationToken=%+v", slackVerificationToken)
		return c.String(http.StatusOK, slackVerificationToken.Challenge)
	})

	e.POST("/interactive", func(c echo.Context) error {
		callBackStruct := new(slack.InteractionCallback)
		err := json.Unmarshal([]byte(c.Request().FormValue("payload")), &callBackStruct)

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
			} else {
				username = profile.DisplayName
			}

			for _, action := range blockActions {
				log.Printf("User (Id:%s) %s clicked on (ActionId:%s, ActionValue:%s)", userId, username, action.ActionID, action.Value)
				switch {
				case strings.Contains(action.ActionID, "mood_user"):
					if err := simba.HandleAddDailyMood(dbClient, channelId, userId, username, action.Value, ""); err != nil {
						return err
					} else {
						log.Printf("Mood %s has been added for the daily for %s", action.Value, username)
						simba.SendSlackTSMessage(slackClient, config, fmt.Sprintf("<@%s> has responded to the daily message with %s", userId, action.Value), threadTS)
						slackClient.AddReaction("robot_face", slack.ItemRef{Timestamp: threadTS, Channel: channelId})
						return nil
					}
				case strings.Contains(action.ActionID, "send_kind_message"):
					log.Printf("Warning this has to be handled by another thing (blockId:%s, value:%s)", action.ActionID, action.Value)
					privateChannel, _, _, err := slackClient.OpenConversation(&slack.OpenConversationParameters{Users: []string{action.Value}})
					if err != nil {
						log.Printf("WARNING CANNOT OPEN PRIVATE CONV : %s\nPrivateChannel=%+v", err.Error(), privateChannel)
						return err
					}
					_, err = simba.SendSlackMessageToUser(slackClient, privateChannel.ID, "Kind Message TESTS (PS: DSL LOUIS SI CA TOMBE SUR TOI :P )")
					if err != nil {
						return err
					}
					return nil
				default:
					log.Printf("WARNING ENTERED IN DEFAULT !!!")
					return fmt.Errorf("Entered in default case")
				}
			}
		} else {
			return fmt.Errorf("Nothing has been received when clicking the button")
		}

		return nil
	})

	defer close(config.SLACK_MESSAGE_CHANNEL)
	port := fmt.Sprintf(":%s", config.APP_PORT)
	if err := e.Start(port); err != nil {
		log.Fatalf("Error when launching server : %s", err.Error())
		return
	}
}
