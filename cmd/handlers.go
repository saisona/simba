package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/go-co-op/gocron"
	"github.com/labstack/echo/v4"
	"github.com/saisona/simba"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

func handleMigration(e *echo.Echo) bool {
	hasMigrationStr, exists := os.LookupEnv("APP_MIGRATE")
	var hasMigration bool = exists
	var err error

	if exists && hasMigrationStr != "" {
		if hasMigration, err = strconv.ParseBool(hasMigrationStr); err != nil {
			e.Logger.Printf(
				"APP_MIGRATE(%s) has been set but cannot be converted to bool : %s ",
				hasMigrationStr,
				err.Error(),
			)
			return false
		}
	}
	return hasMigration
}

func initApplication(
	e *echo.Echo,
	threadTS string,
) (string, *simba.Config, *gorm.DB, *slack.Client, *gocron.Scheduler, error) {
	slackSigningSecret, ok := os.LookupEnv("SLACK_SIGNING_SECRET")
	if !ok || slackSigningSecret == "" {
		err := fmt.Errorf("SLACK_SIGNING_SECRET does not exists and must be provided")
		return "", nil, nil, nil, nil, err
	}

	config, err := simba.InitConfig(false)
	if err != nil {
		err = fmt.Errorf("failed initConfig: %s", err.Error())
		return slackSigningSecret, nil, nil, nil, nil, err
	}
	dbClient := simba.InitDbClient(
		config.DB.Host,
		config.DB.Username,
		config.DB.Password,
		config.DB.Name,
		handleMigration(e),
	)

	slackClient := slack.New(
		config.SLACK_API_TOKEN,
		slack.OptionDebug(true),
		slack.OptionLog(log.Default()),
	)

	scheduler, _, err := simba.InitScheduler(dbClient, slackClient, config, threadTS)
	if err != nil {
		err = fmt.Errorf("failed initScheduler: %s", err.Error())
		return slackSigningSecret, nil, nil, nil, nil, err
	}
	return slackSigningSecret, config, dbClient, slackClient, scheduler, nil
}

func watchValueChanged(valueToChange *string, channel chan string, logger echo.Logger) {
	for {
		oldValue := *valueToChange
		val := <-channel
		*valueToChange = val
		logger.Printf("Changed slackTS from %s to %s", oldValue, *valueToChange)
	}
}
