package main

import (
	"log"
	"os"

	"github.com/saisona/simba"
)

func main() {
	os.Setenv("CHANNEL_ID", "toto")
	dbClient := simba.InitDbClient("devops.cvi0oqscx2wq.eu-west-1.rds.amazonaws.com", "postgres", "cmC2Fab6V5PnP37zaTJp", "simba")
	foundUser, mood, err := simba.FetchLastPersonInBadMood(dbClient, "123212")
	if err != nil {
		panic(err)
	}
	log.Printf("User in bad mood => %s and mood = %s because %s", foundUser.Username, mood.Mood, mood.Context)
	os.Clearenv()
}
