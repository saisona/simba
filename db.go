/**
 * File              : db.go
 * Author            : Alexandre Saison <alexandre.saison@inarix.com>
 * Date              : 14.11.2021
 * Last Modified Date: 16.11.2021
 * Last Modified By  : Alexandre Saison <alexandre.saison@inarix.com>
 */
package simba

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDbClient(dbHost, dbUser, dbPassword, dbName string) *gorm.DB {
	connectionString := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable TimeZone=Europe/Paris", dbHost, dbUser, dbPassword, dbName)
	gormConfig := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		AllowGlobalUpdate:                        true,
	}

	db, err := gorm.Open(postgres.Open(connectionString), gormConfig)
	if err != nil {
		panic(err)
	}

	// create database foreign key for user & credit_cards
	if err = db.Migrator().CreateConstraint(&User{}, "Moods"); err != nil {
		log.Printf("Cannot create basic constraint for DailyMoods and Users: %s", err.Error())
	} else if err = db.Migrator().CreateConstraint(&User{}, "fk_users_daily_moods"); err != nil {
		log.Printf("Cannot create basic constraint Key for DailyMoods and Users: %s", err.Error())
	}

	if err = db.AutoMigrate(&User{}, &DailyMood{}); err != nil {
		panic(err)
	}

	return db

}

func HandleAddDailyMood(dbClient *gorm.DB, channelId string, userId string, userName string, mood string, context string) error {
	var foundUser User
	tx := dbClient.Where(User{SlackUserID: userId, Username: userName, SlackChannelId: channelId}).FirstOrCreate(&foundUser, User{SlackUserID: userId, Username: userName, SlackChannelId: channelId})
	if tx.Error != nil {
		return tx.Error
	}

	//Adding mood to the user
	if context == "" {
		log.Printf("For the moment we have to wait for the Context Input, and sometimes it won't be added by the user")
	}

	dailyMoodToCreate := &DailyMood{UserID: foundUser.ID, Mood: mood, Context: ""}
	moodCreationTx := dbClient.Create(dailyMoodToCreate)

	if moodCreationTx.Error != nil {
		log.Printf("StatementError : %s", moodCreationTx.Statement.Error)
		return moodCreationTx.Error
	}

	return nil
}

type User struct {
	gorm.Model
	SlackUserID    string
	SlackChannelId string
	IsManager      bool
	Username       string `gorm:"unique"`
	Moods          []DailyMood
}

type DailyMood struct {
	gorm.Model
	CreatedAt time.Time
	UserID    uint
	Mood      string
	Context   string
}
