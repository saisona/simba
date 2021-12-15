/**
 * File              : scheduler.go
 * Author            : Alexandre Saison <alexandre.saison@inarix.com>
 * Date              : 08.11.2021
 * Last Modified Date: 14.11.2021
 * Last Modified By  : Alexandre Saison <alexandre.saison@inarix.com>
 */
package simba

import (
	"log"
	"os"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

func funcHandler(dbClient *gorm.DB, client *slack.Client, config *Config, previousBlocks []slack.Block, slackBlockChan chan []slack.Block) error {
	threadTs, err := SendSlackBlocks(client, config, dbClient, previousBlocks, true, slackBlockChan)
	if err != nil {
		log.Printf("#SendSlackBlocks error => %s", err)
		return err
	}
	//Sending threadTS
	config.SLACK_MESSAGE_CHANNEL <- threadTs
	return nil
}

func InitScheduler(dbClient *gorm.DB, client *slack.Client, config *Config, previousBlocks []slack.Block, slackBlockChan chan []slack.Block) (*gocron.Scheduler, *gocron.Job, error) {
	scheduler := gocron.NewScheduler(time.Local)
	if os.Getenv("APP_ENV") == "production" {
		scheduler.CronWithSeconds(config.CRON_EXPRESSION)
	} else if os.Getenv("APP_ENV") == "test" {
		scheduler.Every(10).Minute()
	} else {
		scheduler.Every(10).Minute()
	}

	job, err := scheduler.Do(funcHandler, dbClient, client, config, previousBlocks, slackBlockChan)
	if err != nil {
		return scheduler, nil, err
	} else if job.Error() != nil {
		return scheduler, job, err
	}

	go watcherNewBadMoodUser(dbClient, client, config)

	return scheduler, job, nil
}

func watcherNewBadMoodUser(dbClient *gorm.DB, client *slack.Client, config *Config) {
	for {
		newBadMoodUser := <-config.LAST_BAD_MOOD_USER
		log.Println("newBadMoodUser=", newBadMoodUser)
	}
}
