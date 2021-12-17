/**
 * File              : main.go
 * Author            : Alexandre Saison <alexandre.saison@inarix.com>
 * Date              : 08.11.2021
 * Last Modified Date: 16.11.2021
 * Last Modified By  : Alexandre Saison <alexandre.saison@inarix.com>
 */

package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/slack-go/slack"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{LogLevel: 2}))

	var threadTS string

	slackSigningSecret, config, dbClient, slackClient, scheduler, err := initApplication(e, threadTS)
	if err != nil {
		e.Logger.Fatal("initApplication failed : %s", err.Error())
		return
	}

	var previousBlocks []slack.Block

	go watchValueChanged(&threadTS, config.SLACK_MESSAGE_CHANNEL, e.Logger)

	scheduler.StartAsync()

	e.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	e.POST("/events", func(c echo.Context) error {
		return handleRouteEvents(c, slackClient, dbClient, config, slackSigningSecret)
	})

	e.POST("/interactive", func(c echo.Context) error {
		return handleRouteInteractive(c, slackClient, config, dbClient, threadTS, previousBlocks)
	})

	defer close(config.SLACK_MESSAGE_CHANNEL)
	port := fmt.Sprintf(":%s", config.APP_PORT)
	if err := e.Start(port); err != nil {
		e.Logger.Fatalf("Error when launching server : %s", err.Error())
		return
	}
}
