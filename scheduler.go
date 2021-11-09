/**
 * File              : scheduler.go
 * Author            : Alexandre Saison <alexandre.saison@inarix.com>
 * Date              : 08.11.2021
 * Last Modified Date: 09.11.2021
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
	threadTs, err := SendSlackBlocks(client, config, nil)
	if err != nil {
		log.Printf("Error => %s", err)
		return err
	}
	log.Printf("To use it as a thread, please use %s", threadTs)
	return nil
}

func InitScheduler(client *slack.Client, config *Config) (*gocron.Scheduler, *gocron.Job, error) {
	scheduler := gocron.NewScheduler(time.Local)
	if os.Getenv("APP_ENV") == "production" {
		scheduler.Cron(config.CRON_EXPRESSION)
	} else {
		scheduler.Every(1).Minute()
	}

	job, err := scheduler.Do(funcHandler, client, config)
	if err != nil {
		return scheduler, nil, err
	} else if job.Error() != nil {
		return scheduler, job, err
	}

	return scheduler, job, nil
}
