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
)

func funcHandler(client *slack.Client, config *Config) error {
	_, err := SendSlackBlocks(client, config, nil)
	if err != nil {
		log.Printf("Error => %s", err)
		return err
	}
	return nil
}

func InitScheduler(client *slack.Client, config *Config) (*gocron.Scheduler, *gocron.Job, error) {
	scheduler := gocron.NewScheduler(time.Local)
	if os.Getenv("APP_ENV") == "production" {
		scheduler.CronWithSeconds(config.CRON_EXPRESSION)
	} else if os.Getenv("APP_ENV") == "test" {
		scheduler.Every(10).Minute()
	} else {
		scheduler.Every(10).Minute()
	}

	job, err := scheduler.Do(funcHandler, client, config)
	if err != nil {
		return scheduler, nil, err
	} else if job.Error() != nil {
		return scheduler, job, err
	}

	return scheduler, job, nil
}
