/**
 * File              : main.go
 * Author            : Alexandre Saison <alexandre.saison@inarix.com>
 * Date              : 08.11.2021
 * Last Modified Date: 14.11.2021
 * Last Modified By  : Alexandre Saison <alexandre.saison@inarix.com>
 */

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/saisona/simba"
	"github.com/slack-go/slack"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	//datadogClient := *datadog.NewAPIClient(&datadog.Configuration{Host: "datadoghq.eu", Debug: true})

	config, err := simba.InitConfig()
	if err != nil {
		log.Fatalf("Failed launching server: %s", err.Error())
	}

	slackClient := slack.New(config.SLACK_API_TOKEN, slack.OptionDebug(true), slack.OptionLog(log.Default()))
	scheduler, job, err := simba.InitScheduler(slackClient, config)
	scheduler.StartAsync()

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
		}
		if len(callBackStruct.ActionCallback.AttachmentActions) > 0 {
			blockAttachmentActions := callBackStruct.ActionCallback.AttachmentActions
			log.Printf("There is %d block actions", len(blockAttachmentActions))
			for idx, action := range blockAttachmentActions {
				log.Printf("AttachementAction[%d] : %+v", idx, action)
				log.Printf("AttachementValue = %s", action.Value)
			}
		} else if len(callBackStruct.ActionCallback.BlockActions) > 0 {
			blockActions := callBackStruct.ActionCallback.BlockActions
			user := callBackStruct.User
			userId := user.ID
			profile, err := slackClient.GetUserProfile(&slack.GetUserProfileParameters{UserID: userId})
			if err != nil {
				log.Printf("Warning some error while fetchingProfile = %s ", err.Error())
			}

			userName := profile.DisplayName
			log.Printf("There is %d block actions", len(blockActions))
			for _, action := range blockActions {
				log.Printf("User (Id: %s) %s clicked on %s", userId, userName, action.Value)
			}

		} else {
			return fmt.Errorf("Nothing has been received when clicking the button")
		}
		return nil
	})

	port := fmt.Sprintf(":%s", config.APP_PORT)
	if err := e.Start(port); err != nil {
		log.Fatalf("Error when launching server : %s", err.Error())
		return
	}
}
