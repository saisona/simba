/**
 * File              : main.go
 * Author            : Alexandre Saison <alexandre.saison@inarix.com>
 * Date              : 08.11.2021
 * Last Modified Date: 09.11.2021
 * Last Modified By  : Alexandre Saison <alexandre.saison@inarix.com>
 */

package main

import (
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
	_, job, err := simba.InitScheduler(slackClient, config)

	if err != nil {
		log.Fatalf("Failed launching server: %s", err.Error())
	}

	jErr := job.Error()
	if jErr != nil {
		log.Fatalf("Job has failed: %s", jErr.Error())
	}

	e.GET("/healthz", func(c echo.Context) error {
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		return c.NoContent(http.StatusNoContent)
	})
	e.POST("/interactive", func(c echo.Context) error {
		// TODO: Handle interactive for Slack application
		return c.String(http.StatusOK, "Toto")
	})

	e.Start(config.APP_PORT)
}
