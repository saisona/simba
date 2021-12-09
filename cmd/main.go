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
	"gorm.io/gorm"
)

func watchValueChanged(valueToChange *string, channel chan string, logger echo.Logger) {
	for val, more := <-channel; !more; {
		logger.Debugf("received new value for ThreadTS=", val)
		if valueToChange != nil {
			logger.Debugf("OldValue=%s\n", *valueToChange)
			*valueToChange = val
			logger.Debugf("NewValue=%s\n", *valueToChange)
		} else {
		}
	}
}

func handleError(errChan chan error, c echo.Context) {
	for hasErr, more := <-errChan; more; {
		if hasErr != nil {
			c.Error(hasErr)
		}
		log.Printf("No error")
	}
}

func handleUpdateMood(config *simba.Config, slackClient *slack.Client, dbClient *gorm.DB, threadTS, channelId, userId, username string, action *slack.BlockAction) error {
	if err := simba.HandleAddDailyMood(dbClient, channelId, userId, username, action.Value, ""); err != nil {
		return err
	} else {
		log.Printf("Mood %s has been added for the daily for %s", action.Value, username)
		slackMessage := fmt.Sprintf("<@%s> has responded to the daily message with %s", userId, strings.ReplaceAll(action.Value, "_", " "))
		simba.SendSlackTSMessage(slackClient, config, slackMessage, threadTS)
		slackClient.AddReaction("robot_face", slack.ItemRef{Timestamp: threadTS, Channel: channelId})
		return nil
	}
}

func handleSendKindMessage(slackClient *slack.Client, errChan chan error, userId string, action *slack.BlockAction) error {
	log.Printf("Warning this has to be handled by another thing (blockId:%s, value:%s)", action.ActionID, action.Value)
	defer close(errChan)
	privateChannel, _, _, err := slackClient.OpenConversation(&slack.OpenConversationParameters{Users: []string{action.Value}})
	if err != nil {
		log.Printf("WARNING CANNOT OPEN PRIVATE CONV : %s\nPrivateChannel=%+v", err.Error(), privateChannel)
		return err
	}

	words := simba.GenerateBuzzWords()
	title, urlToDownload, err := simba.FetchRelatedGif(words[simba.GenerateRandomIndexBuzzWord(words)])
	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("/tmp/%s", title)
	comment := fmt.Sprintf("<@%s> thought about you :hugging_face: and wanted to send you some kind image", userId)
	if err := simba.DownloadFile(filePath, urlToDownload, false); err != nil {
		errChan <- err
		return err
	} else if err = simba.SendImage(slackClient, privateChannel.ID, filePath, title, comment); err != nil {
		errChan <- err
		return err
	}
	return nil
}

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
	scheduler, job, err := simba.InitScheduler(dbClient, slackClient, config)
	scheduler.StartAsync()

	var threadTS string

	// call anonymous goroutine
	go watchValueChanged(&threadTS, config.SLACK_MESSAGE_CHANNEL, e.Logger)

	if err != nil {
		e.Logger.Fatalf("Failed launching server: %s", err.Error())
	}

	jErr := job.Error()
	if jErr != nil {
		e.Logger.Fatalf("Job has failed: %s", jErr.Error())
	}

	e.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	e.POST("/events", func(c echo.Context) error {
		body, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			c.Response().Writer.WriteHeader(http.StatusBadRequest)
			return err
		}
		sv, err := slack.NewSecretsVerifier(c.Request().Header, slackSigningSecret)
		if err != nil {
			c.Response().Writer.WriteHeader(http.StatusBadRequest)
			return err
		}
		if _, err := sv.Write(body); err != nil {
			c.Response().Writer.WriteHeader(http.StatusInternalServerError)
			return err
		}
		if err := sv.Ensure(); err != nil {
			c.Response().Writer.WriteHeader(http.StatusUnauthorized)
			return err
		}

		var slackVerificationToken simba.SlackVerificationStruct
		e.Logger.Debug("SlackEventMapping >", slack.EventMapping)
		if err := c.Bind(&slackVerificationToken); err != nil {
			return err
		}
		e.Logger.Printf("slackVerificationToken=%+v", slackVerificationToken)
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
				switch {
				case strings.Contains(action.ActionID, "mood_user"):
					go handleUpdateMood(config, slackClient, dbClient, threadTS, channelId, userId, username, action)
				case strings.Contains(action.ActionID, "send_kind_message"):
					errChan := make(chan error, 1)
					defer close(errChan)
					go handleError(errChan, c)
					go handleSendKindMessage(slackClient, errChan, userId, action)

				default:
					c.Logger().Warn("Action is in default case, which means not handled at the moment")
					c.Logger().Printf("User (Id:%s) %s clicked on (ActionId:%s, ActionValue:%s)", userId, username, action.ActionID, action.Value)
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
