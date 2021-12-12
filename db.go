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

	"github.com/slack-go/slack"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func handleMigration(db *gorm.DB) error {
	// create database foreign key for user & credit_cards
	if !db.Migrator().HasConstraint(&User{}, "Moods") {
		if err := db.Migrator().CreateConstraint(&User{}, "Moods"); err != nil {
			log.Printf("Cannot create basic constraint for DailyMoods and Users: %s", err.Error())
			return err
		}
	} else if !db.Migrator().HasConstraint(&User{}, "fk_users_daily_moods") {
		if err := db.Migrator().CreateConstraint(&User{}, "fk_users_daily_moods"); err != nil {
			log.Printf("Cannot create basic constraint Key for DailyMoods and Users: %s", err.Error())
			return err
		}
	}

	if err := db.AutoMigrate(&User{}, &DailyMood{}); err != nil {
		return err
	}
	return nil
}

func InitDbClient(dbHost, dbUser, dbPassword, dbName string, migrate bool) *gorm.DB {
	connectionString := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable TimeZone=Europe/Paris", dbHost, dbUser, dbPassword, dbName)
	gormConfig := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		AllowGlobalUpdate:                        true,
	}

	db, err := gorm.Open(postgres.Open(connectionString), gormConfig)
	if err != nil {
		panic(err)
	}

	if err := handleMigration(db); migrate && err != nil {
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

	dailyMoodToCreate := &DailyMood{UserID: foundUser.ID, Mood: mood, Context: ""}
	moodCreationTx := dbClient.Create(dailyMoodToCreate)

	if moodCreationTx.Error != nil {
		log.Printf("StatementError : %s", moodCreationTx.Statement.Error)
		return moodCreationTx.Error
	}

	return nil
}

func FetchLastPersonInBadMood(dbClient *gorm.DB, channelId string) (*User, *DailyMood, error) {
	var foundUser *User
	var lastBadMood *DailyMood

	//Find related BadMood
	txBadMood := dbClient.Where("mood = ?", "bad_mood").Last(&lastBadMood)
	if txBadMood.Error != nil {
		return nil, nil, txBadMood.Error
	}

	//Find last user with BadMood
	badMoodTx := dbClient.Where(foundUser, lastBadMood.UserID).First(&foundUser)
	if err := badMoodTx.Error; err != nil {
		return nil, nil, badMoodTx.Error
	}

	return foundUser, lastBadMood, nil
}

func IsUserAdmin(dbClient *gorm.DB, userId string) (bool, error) {
	var user *User
	fetchUserTx := dbClient.Find(&user, "slack_user_id = ?", userId)
	if fetchUserTx.Error != nil {
		return false, fetchUserTx.Error
	}
	log.Printf("[DEBUG] User => %+v", user)

	return user.IsManager, nil
}

func FechCurrent(dbClient *gorm.DB, slackClient *slack.Client, slackUserId string) (*User, *slack.User, error) {
	var user *User
	fetchUserTx := dbClient.Find(&user, "slack_user_id = ?", slackUserId)
	if fetchUserTx.Error != nil {
		return nil, nil, fetchUserTx.Error
	}

	slackUser, err := FetchUserById(slackClient, slackUserId)
	if err != nil {
		return nil, nil, err
	}

	return user, slackUser, nil
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
