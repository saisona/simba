/**
 * File              : config.go
 * Author            : Alexandre Saison <alexandre.saison@inarix.com>
 * Date              : 08.11.2021
 * Last Modified Date: 14.11.2021
 * Last Modified By  : Alexandre Saison <alexandre.saison@inarix.com>
 */
package simba

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
)

func InitConfig() (*Config, error) {
	if _, err := os.Open(".env"); err == nil {
		os.Clearenv()
		err := godotenv.Load()
		if err != nil {
			return nil, err
		}
	} else {
		log.Printf(".env does not exists")
	}

	chanId := os.Getenv("CHANNEL_ID")
	slackApiToken := os.Getenv("SLACK_API_TOKEN")
	applicationPort := os.Getenv("APP_PORT")

	missingEnv := []string{}

	switch {
	case chanId == "":
		missingEnv = append(missingEnv, "CHANNEL_ID")
	case slackApiToken == "":
		missingEnv = append(missingEnv, "SLACK_API_TOKEN")
	case applicationPort == "":
		missingEnv = append(missingEnv, "APP_PORT")
	}

	if len(missingEnv) > 0 {
		return nil, fmt.Errorf("%s is not set", strings.Join(missingEnv, " "))
	}

	cronExpression, cronExists := os.LookupEnv("APP_CRON_EXPRESSION")
	if !cronExists || cronExpression == "" {
		cronExpression = "0 0 10 ? * MON-FRI"
		log.Printf("APP_CRON_EXPRESSION has not been set in env ! Using default one : %s", cronExpression)
	}

	dbConfig, err := initDbConfig()
	if err != nil {
		return nil, fmt.Errorf("initDbConfig failed : %s", err.Error())
	}

	slackMessageChannel := make(chan string)
	slackBlockChan := make(chan []slack.Block)
	return &Config{CHANNEL_ID: chanId, SLACK_API_TOKEN: slackApiToken, APP_PORT: applicationPort, CRON_EXPRESSION: cronExpression, DB: dbConfig, SLACK_MESSAGE_CHANNEL: slackMessageChannel, SLACK_PREVIOUS_BLOCK: slackBlockChan}, nil
}

func initDbConfig() (*DbConfig, error) {
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	name := os.Getenv("DB_NAME")

	missingEnv := []string{}
	switch {
	case user == "":
		missingEnv = append(missingEnv, "DB_USER")
	case password == "":
		missingEnv = append(missingEnv, "DB_PASSWORD")
	case host == "":
		missingEnv = append(missingEnv, "DB_HOST")
	case name == "":
		missingEnv = append(missingEnv, "DB_NAME")
	}

	if len(missingEnv) > 0 {
		err := fmt.Errorf("%s is not set", strings.Join(missingEnv, " "))
		return nil, err
	}

	return &DbConfig{Username: user, Password: password, Host: host, Name: name}, nil

}

type Config struct {
	CHANNEL_ID            string
	SLACK_API_TOKEN       string
	APP_PORT              string
	CRON_EXPRESSION       string
	SLACK_MESSAGE_CHANNEL chan string
	SLACK_PREVIOUS_BLOCK  chan []slack.Block
	DB                    *DbConfig
}

type DbConfig struct {
	Username string
	Password string
	Host     string
	Name     string
}
