/**
 * File              : db.go
 * Author            : Alexandre Saison <alexandre.saison@inarix.com>
 * Date              : 14.11.2021
 * Last Modified Date: 14.11.2021
 * Last Modified By  : Alexandre Saison <alexandre.saison@inarix.com>
 */
package simba

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDbClient(dbHost, dbUser, dbPassword, dbName string) *gorm.DB {
	connectionString := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable TimeZone=Europe/Paris", dbHost, dbUser, dbPassword, dbName)
	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		panic(err)
	}
	return db

}
