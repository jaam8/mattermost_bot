package models

import "errors"

var (
	ErrPollIsEnd           = errors.New("poll is end")
	ErrPollNotFound        = errors.New("poll is not found")
	ErrFailedToProcessData = errors.New("failed to process data")
	ErrOptionIsEmpty       = errors.New("option is empty")
	ErrNotEnoughOptions    = errors.New("the number of options should be at least 2")
	ErrQuestionIsEmpty     = errors.New("question is empty")
	ErrOptionIsNotFound    = errors.New("option is not found")
	ErrVoteAlreadyExists   = errors.New("your vote already written")
)

type Poll struct {
	ID       string   `json:"id"`
	Question string   `json:"question" `
	Options  []Option `json:"options"`
	// Votes: unmarshal to map[string]int (key: Option.ID (convert to string), value: count of votes)
	Votes     map[string]int `json:"votes"`
	CreatorID string         `json:"creator_id"`
	IsActive  bool           `json:"is_active"`
}

type Vote struct {
	PollID   string `json:"poll_id"`
	UserID   string `json:"user_id"`
	ChoiceID string `json:"choice_id"`
}

type Option struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}
