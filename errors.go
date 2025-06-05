package simba

import (
	"fmt"
)

type ErrMoodAlreadySet struct {
	UserID string
}

func (err *ErrMoodAlreadySet) Error() string {
	return fmt.Sprintf("%s has already set DailyMood", err.UserID)
}

func NewErrMoodAlreadySet(userID string) *ErrMoodAlreadySet {
	return &ErrMoodAlreadySet{UserID: userID}
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
