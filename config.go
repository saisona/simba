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

	"github.com/joho/godotenv"
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

	cronExpression := os.Getenv("APP_CRON_EXPRESSION")
	if cronExpression == "" {
		cronExpression = "0 0 10 ? * MON-FRI"
		log.Printf("APP_CRON_EXPRESSION has not been set in env ! Using default one : %s", cronExpression)
	}

	if chanId == "" || slackApiToken == "" {
		log.Fatalf("One of CHANNEL_ID or SLACK_API_TOKEN (%s, %s)", chanId, slackApiToken)
	}

	dbConfig, err := initDbConfig()
	if err != nil {
		log.Fatalf("initDbConfig failed : %s", err.Error())
	}

	return &Config{CHANNEL_ID: chanId, SLACK_API_TOKEN: slackApiToken, APP_PORT: applicationPort, CRON_EXPRESSION: cronExpression, DB: dbConfig, SLACK_MESSAGE_CHANNEL: make(chan string)}, nil
}

func initDbConfig() (*DbConfig, error) {
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	name := os.Getenv("DB_NAME")

	if user == "" || password == "" || host == "" || name == "" {
		return nil, fmt.Errorf("One of DB_USER(%s), DB_PASSWORD(%s), DB_HOST(%s), DB_NAME(%s) is not set", user, password, host, name)
	}

	return &DbConfig{Username: user, Password: password, Host: host, Name: name}, nil

}

type Config struct {
	CHANNEL_ID            string
	SLACK_API_TOKEN       string
	APP_PORT              string
	CRON_EXPRESSION       string
	SLACK_MESSAGE_CHANNEL chan string
	DB                    *DbConfig
}

type DbConfig struct {
	Username string
	Password string
	Host     string
	Name     string
}
