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
	"strconv"
	"time"

	"github.com/slack-go/slack"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// create database foreign key for user & credit_cards
func handleMigration(db *gorm.DB) error {

	if err := db.AutoMigrate(&User{}); err != nil {
		return err
	}

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

	return nil
}

// Initialize database client (*gorm.DB)
//--------------------------------------
// @args
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

func UpdateMood(dbClient *gorm.DB, sourceMood *DailyMood, feeling *string, context *string) (*DailyMood, error) {
	if sourceMood == nil {
		return nil, fmt.Errorf("sourceMood is nil")
	}
	if feeling != nil {
		tx := dbClient.Model(sourceMood).Update("feeling", *feeling)
		if tx.Error != nil {
			return sourceMood, tx.Error
		}
	}
	if context != nil {
		tx := dbClient.Model(sourceMood).Update("context", *context)
		if tx.Error != nil {
			return sourceMood, tx.Error
		}
	}
	return sourceMood, nil
}

func UpdateMoodById(dbClient *gorm.DB, moodId string, feeling *string, context *string) (*DailyMood, error) {
	var sourceMood DailyMood
	if tx := dbClient.Debug().First(&sourceMood, "id = ? ", moodId); tx.Error != nil {
		return &sourceMood, tx.Error
	}

	moodIdInt, _ := strconv.Atoi(moodId)

	if sourceMood.ID != uint(moodIdInt) {
		return &sourceMood, fmt.Errorf("sourceMood not found for moodId = %s", moodId)
	}

	if feeling != nil {
		tx := dbClient.Model(&sourceMood).Update("feeling", *feeling)
		if tx.Error != nil {
			return &sourceMood, tx.Error
		}
	}
	if context != nil {
		tx := dbClient.Model(&sourceMood).Update("context", *context)
		if tx.Error != nil {
			return &sourceMood, tx.Error
		}
	}
	return &sourceMood, nil
}

func deleteDailyMood(dbClient *gorm.DB, moodId uint) (bool, error) {
	deleteTx := dbClient.Debug().Delete(&DailyMood{}, moodId)
	if deleteTx.Error != nil {
		return false, deleteTx.Error
	}
	return true, nil
}

func handleUpdateDailyMood(dbClient *gorm.DB, user *User, mood, threadTS string) (*DailyMood, error) {
	moodToDelete, err := FetchMoodFromThreadTS(dbClient, threadTS, user.ID)
	if err != nil {
		return nil, err
	}
	if isDeleted, err := deleteDailyMood(dbClient, moodToDelete.ID); err != nil {
		return nil, err
	} else if !isDeleted {
		return nil, fmt.Errorf("Mood %d has not been deleted since does not exists", moodToDelete.ID)
	} else {
		moodToCreate := &DailyMood{UserID: user.ID, Mood: mood, ThreadTS: threadTS}

		user.Moods = append(user.Moods, *moodToCreate)
		tx := dbClient.Debug().Session(&gorm.Session{FullSaveAssociations: true}).Updates(&user)
		if tx.Error != nil {
			return nil, fmt.Errorf("update with dailyMood: %s", tx.Error.Error())
		}
		return moodToCreate, nil
	}
}

func HandleAddDailyMood(dbClient *gorm.DB, slackClient *slack.Client, channelId, userId, userName, mood, threadTS string) (*DailyMood, error) {
	var foundUser User = User{SlackUserID: userId, SlackChannelId: channelId, Username: userName}

	tx := dbClient.FirstOrInit(&foundUser, "slack_user_id = ?", foundUser.SlackUserID)
	if tx.Error != nil {
		return nil, fmt.Errorf("firstOrInit: %s", tx.Error)
	} else if foundUser.ID != 0 {
		if hasAlreadySetMood, err := HasAlreadySetMood(dbClient, slackClient, userId, threadTS); err != nil {
			log.Printf("Error hasAlreadySetMood : %s", err.Error())
			return nil, err
		} else if hasAlreadySetMood {
			//TODO: handle update mood !!
			return handleUpdateDailyMood(dbClient, &foundUser, mood, threadTS)
			//return nil, NewErrMoodAlreadySet(userId)
		}
	} else {
		tx = dbClient.Debug().Save(&foundUser)
		if tx.Error != nil {
			return nil, tx.Error
		}
	}

	moodToCreate := &DailyMood{UserID: foundUser.ID, Mood: mood, ThreadTS: threadTS}

	foundUser.Moods = append(foundUser.Moods, *moodToCreate)
	tx = dbClient.Debug().Session(&gorm.Session{FullSaveAssociations: true}).Updates(&foundUser)
	if tx.Error != nil {
		return nil, fmt.Errorf("update with dailyMood: %s", tx.Error.Error())
	} else if tx = dbClient.First(&moodToCreate, "user_id = ? AND thread_ts = ? ", foundUser.ID, threadTS); tx.Error != nil {
		return nil, fmt.Errorf("fetch real dailyMood failed : %s", tx.Error.Error())
	}

	return moodToCreate, nil
}

func HasAlreadySetMood(dbClient *gorm.DB, slackClient *slack.Client, userID, threadTS string) (bool, error) {
	user, _, err := FechCurrent(dbClient, slackClient, userID)
	if err != nil {
		return false, err
	}

	moodToFind, err := FetchMoodFromThreadTS(dbClient, threadTS, user.ID)
	if err != nil {
		return false, err
	}

	return moodToFind.ID != 0, nil
}

func FetchMoodFromThreadTS(dbClient *gorm.DB, threadTS string, userId uint) (*DailyMood, error) {
	var moodToFind DailyMood
	if tx := dbClient.Find(&moodToFind, "thread_ts = ? AND user_id = ? ", threadTS, userId); tx.Error != nil {
		return nil, tx.Error
	}

	return &moodToFind, nil
}

func FetchAllDailyMoodsByThreadTS(dbClient *gorm.DB, threadTS string) ([]*User, error) {
	var dailyMoodsUser []*User
	dbClient = dbClient.Debug()

	if tx := dbClient.Model(&User{}).Find(&dailyMoodsUser); tx.Error != nil {
		return nil, tx.Error
	}

	for _, user := range dailyMoodsUser {
		var tmpMood []DailyMood
		err := dbClient.Model(user).Association("Moods").Find(&tmpMood, "thread_ts = ? ", threadTS)
		if err != nil {
			return nil, err
		}
		user.Moods = tmpMood
	}

	return dailyMoodsUser, nil
}

func IsUserAdmin(dbClient *gorm.DB, userId string) (bool, error) {
	var user *User
	fetchUserTx := dbClient.Find(&user, "slack_user_id = ?", userId)
	if fetchUserTx.Error != nil {
		return false, fetchUserTx.Error
	}
	return user.IsManager, nil
}

func FechCurrent(dbClient *gorm.DB, slackClient *slack.Client, slackUserId string) (*User, *slack.User, error) {
	var user *User
	fetchUserTx := dbClient.Debug().Find(&user, "slack_user_id = ?", slackUserId)
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
	Username       string      `gorm:"unique"`
	Moods          []DailyMood `gorm:"many2many:has_moods"`
}

type DailyMood struct {
	gorm.Model
	CreatedAt time.Time
	UserID    uint
	Mood      string
	Feeling   string
	ThreadTS  string
	Context   string
}
