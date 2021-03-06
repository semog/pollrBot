package main

import (
	"fmt"
)

type answer struct {
	ID       int
	PollID   int
	UserID   int
	OptionID int
}

type option struct {
	ID     int
	PollID int
	Text   string
	Ctr    int
}

type poll struct {
	ID             int
	MessageID      int
	UserID         int
	Question       string
	Inactive       int
	Private        int
	Type           int
	DisplayPercent int
	Options        []option
	Answers        []answer
}

func (poll *poll) fmtQuery(query string) string {
	return fmt.Sprintf("%c:%d:%s", qryEditPayload, poll.ID, query)
}

func (poll *poll) isInactive() bool {
	return poll.Inactive == inactive
}

func (poll *poll) isMultipleChoice() bool {
	return poll.Type == multipleChoice
}

func (poll *poll) isDisplayVotePercent() bool {
	return poll.DisplayPercent == displayVotePercent
}
