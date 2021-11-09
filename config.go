/**
 * File              : config.go
 * Author            : Alexandre Saison <alexandre.saison@inarix.com>
 * Date              : 08.11.2021
 * Last Modified Date: 08.11.2021
 * Last Modified By  : Alexandre Saison <alexandre.saison@inarix.com>
 */
package simba

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func InitConfig() (*Config, error) {
	os.Clearenv()
	if _, err := os.Open(".env"); err == nil {
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

	if chanId == "" || slackApiToken == "" {
		log.Fatalf("One of CHANNEL_ID or SLACK_API_TOKEN (%s, %s)", chanId, slackApiToken)
	}

	return &Config{CHANNEL_ID: chanId, SLACK_API_TOKEN: slackApiToken, APP_PORT: applicationPort}, nil
}

type Config struct {
	CHANNEL_ID      string
	SLACK_API_TOKEN string
	APP_PORT        string
}
