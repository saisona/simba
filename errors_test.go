package simba_test

import (
	"testing"

	"github.com/saisona/simba"
)

func TestErrMoodAlreadySet(t *testing.T) {
	err := simba.NewErrMoodAlreadySet("1234")
	if err.UserID != "1234" {
		t.FailNow()
	} else if err.Error() != "1234 has already set DailyMood" {
		t.FailNow()
	}
}

func TestErrNoActionFound(t *testing.T) {
	err := simba.NewErrNoActionFound("fake_action", "fake_result")
	if err.ActionID != "fake_action" {
		t.FailNow()
	} else if err.ActionValue != "fake_result" {
		t.FailNow()
	} else if err.Error() != "ActionId(fake_action) = fake_result is not registered" {
		t.FailNow()
	}
}
