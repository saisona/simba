package simba

import (
	"fmt"
)

type ErrMoodAlreadySet struct {
	UserId string
}

func (err *ErrMoodAlreadySet) Error() string {
	return fmt.Sprintf("%s has already set DailyMood", err.UserId)
}

func NewErrMoodAlreadySet(userId string) *ErrMoodAlreadySet {
	return &ErrMoodAlreadySet{UserId: userId}
}

// ------------------------------//
type ErrNoActionFound struct {
	ActionID    string
	ActionValue string
}

func (err *ErrNoActionFound) Error() string {
	return fmt.Sprintf("ActionId(%s) = %s is not registered", err.ActionID, err.ActionValue)
}

func NewErrNoActionFound(actionId, actionValue string) *ErrNoActionFound {
	return &ErrNoActionFound{ActionID: actionId, ActionValue: actionValue}
}
