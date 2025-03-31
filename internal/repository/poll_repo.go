package repository

import (
	"encoding/json"
	"fmt"
	"github.com/jaam8/mattermost_bot/internal/models"
	"github.com/tarantool/go-tarantool"
	"go.uber.org/zap"
)

type PollRepository struct {
	db *tarantool.Connection
	l  *zap.Logger
}

func New(db *tarantool.Connection, l *zap.Logger) *PollRepository {
	return &PollRepository{
		db: db,
		l:  l,
	}

}

func (r *PollRepository) CreatePoll(poll *models.Poll) (string, []models.Option, error) {
	r.l.Debug("creating poll", zap.Any("poll", poll))
	votesJSON, err := json.Marshal(poll.Votes)
	r.l.Debug("votes to json", zap.Any("votes", votesJSON))
	if err != nil {
		r.l.Debug("error marshalling votes", zap.Error(err))
		return "", nil, fmt.Errorf("repository: json marshal error: %w", err)
	}

	pollReq := []interface{}{
		poll.ID,
		poll.Question,
		poll.Options,
		string(votesJSON),
		poll.CreatorID,
		poll.IsActive,
	}

	resp, err := r.db.Insert("polls", pollReq)
	r.l.Debug("tarantool response",
		zap.Uint32("status_code", resp.Code),
		zap.Any("resp", resp.Data),
		zap.String("error", resp.Error))

	if err != nil {
		r.l.Debug("error inserting poll", zap.Error(err))
		return "", nil, fmt.Errorf("repository: database insert error: %w, tarantool error: %v", err, resp.Error)
	}
	return poll.ID, poll.Options, nil
}

func (r *PollRepository) Vote(pollID, choiceID, userID string) error {
	pollTuple, err := r.GetPoll(pollID)
	if err != nil {
		return err
	}
	votesField, ok := pollTuple[3].(string)
	if !ok {
		r.l.Debug("unexpected type for votes field", zap.Any("votes_field", pollTuple[3]))
		return models.ErrFailedToProcessData
	}
	isActive, ok := pollTuple[5].(bool)
	if !isActive {
		r.l.Debug("poll is not active", zap.String("poll_id", pollID))
		return models.ErrPollIsEnd
	}
	var votes map[string]int
	err = json.Unmarshal([]byte(votesField), &votes)
	if err != nil {
		r.l.Debug("failed to unmarshal votes", zap.Error(err))
		return fmt.Errorf("repository: failed to unmarshal votes: %w", err)
	}
	if _, ok := votes[choiceID]; !ok {
		r.l.Debug("option not found", zap.String("choice_id", choiceID))
		return models.ErrOptionIsNotFound
	}
	if err = r.InsertVote(pollID, userID, choiceID); err != nil {
		r.l.Debug("failed to insert vote", zap.Error(err))
		return err
	}
	votes[choiceID]++
	r.l.Debug("updated votes", zap.Any("votes", votes))

	updatedVotesJSON, err := json.Marshal(votes)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}
	updateVotes := []interface{}{[]interface{}{"=", 3, string(updatedVotesJSON)}}

	resp, err := r.db.Update("polls", "primary", []interface{}{pollID}, updateVotes)
	r.l.Debug("tarantool response",
		zap.Uint32("status_code", resp.Code),
		zap.Any("resp", resp.Data))
	if err != nil {
		r.l.Debug("failed to update votes", zap.Error(err))
		return fmt.Errorf("repository: database update error: %w", err)
	}
	return nil
}

// InsertVote inserts a vote into the votes space if user has not already voted
func (r *PollRepository) InsertVote(pollID, userID, choiceID string) error {
	existenceVote, err := r.db.Select("votes", "user_poll", 0, 1, tarantool.IterEq, []interface{}{pollID, userID})
	if err != nil {
		r.l.Debug("failed to select vote", zap.Error(err))
		return fmt.Errorf("repository: database select error: %w", err)
	}
	r.l.Debug("tarantool response",
		zap.Uint32("status_code", existenceVote.Code),
		zap.Any("resp", existenceVote.Data),
		zap.Any("error", existenceVote.Error))
	if len(existenceVote.Data) > 0 {
		r.l.Debug("vote already exist",
			zap.String("poll_id", pollID),
			zap.String("user_id", userID))
		return models.ErrVoteAlreadyExists
	}
	vote := &models.Vote{
		PollID:   pollID,
		UserID:   userID,
		ChoiceID: choiceID,
	}
	resp, err := r.db.Insert("votes", []interface{}{vote.PollID, vote.UserID, vote.ChoiceID})
	if err != nil {
		r.l.Debug("failed to insert vote", zap.Error(err))
		return fmt.Errorf("repository: database insert error: %w", err)
	}
	r.l.Debug("tarantool response",
		zap.Uint32("status_code", resp.Code),
		zap.Any("resp", resp.Data),
		zap.Any("error", resp.Error))
	return nil
}

func (r *PollRepository) GetPollResult(pollID string) (*models.Poll, error) {
	pollTuple, err := r.GetPoll(pollID)
	poll := &models.Poll{}
	if err != nil {
		return poll, err
	}
	r.l.Debug("tarantool response", zap.Any("result", pollTuple))
	poll.ID = pollTuple[0].(string)
	poll.Question = pollTuple[1].(string)
	optionsRaw, ok := pollTuple[2].([]interface{})
	if !ok {
		return nil, fmt.Errorf("repository: unexpected type for pollTuple options: %w",
			models.ErrFailedToProcessData)
	}
	var options []models.Option
	for _, opt := range optionsRaw {
		optConv := convertKeys(opt)
		optBytes, err := json.Marshal(optConv)
		if err != nil {
			r.l.Debug("failed to marshal option", zap.Any("option", optConv))
			return nil, fmt.Errorf("repository: failed to marshal option: %w", err)
		}
		var option models.Option
		if err = json.Unmarshal(optBytes, &option); err != nil {
			r.l.Debug("failed to unmarshal option", zap.Any("option", optBytes))
			return nil, fmt.Errorf("repository: failed to unmarshal option: %w", err)
		}
		options = append(options, option)
	}
	r.l.Debug("options", zap.Any("options", options))
	poll.Options = options
	poll.Votes = make(map[string]int)
	err = json.Unmarshal([]byte(pollTuple[3].(string)), &poll.Votes)
	if err != nil {
		r.l.Debug("failed to unmarshal votes", zap.Error(err))
		return nil, fmt.Errorf("repository: failed to unmarshal votes: %w", err)
	}
	poll.CreatorID = pollTuple[4].(string)
	poll.IsActive = pollTuple[5].(bool)
	r.l.Debug("poll data from tarantool", zap.Any("poll", poll))
	return poll, nil
}

func convertKeys(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := make(map[string]interface{})
		for k, v := range x {
			m2[fmt.Sprintf("%v", k)] = convertKeys(v)
		}
		return m2
	case []interface{}:
		for idx, item := range x {
			x[idx] = convertKeys(item)
		}
		return x
	default:
		return i
	}
}

func (r *PollRepository) DeletePoll(pollID, userID string) error {
	pollTuple, err := r.GetPoll(pollID)
	if err != nil {
		return err
	}
	if pollTuple[4].(string) != userID {
		r.l.Debug("user is not the owner of the poll", zap.String("user_id", userID))
		return models.ErrUserNotOwner
	}
	resp, err := r.db.Delete("polls", "primary", []interface{}{pollID})
	if err != nil {
		r.l.Debug("failed to delete poll", zap.Error(err))
		return fmt.Errorf("repository: database delete error: %w", err)
	}
	r.l.Debug("tarantool response",
		zap.Uint32("status_code", resp.Code),
		zap.Any("resp", resp.Data),
		zap.String("error", resp.Error))
	return nil
}

func (r *PollRepository) EndPoll(pollID, userID string) error {
	pollTuple, err := r.GetPoll(pollID)
	if err != nil {
		return err
	}
	if isActive, _ := pollTuple[5].(bool); !isActive {
		r.l.Debug("poll is not active", zap.String("poll_id", pollID))
		return models.ErrPollAlreadyEnded
	}
	if pollTuple[4].(string) != userID {
		r.l.Debug("user is not the owner of the poll", zap.String("user_id", userID))
		return models.ErrUserNotOwner
	}
	resp, err := r.db.Update("polls", "primary",
		[]interface{}{pollID},
		[]interface{}{[]interface{}{"=", 5, false}})
	if err != nil {
		r.l.Debug("failed to update poll", zap.Error(err))
		return fmt.Errorf("repository: database update error: %w", err)
	}
	r.l.Debug("tarantool response",
		zap.Uint32("status_code", resp.Code),
		zap.Any("resp", resp.Data),
		zap.String("error", resp.Error))
	return nil
}

func (r *PollRepository) GetPoll(pollID string) ([]interface{}, error) {
	existencePoll, err := r.db.Select("polls", "primary", 0, 1, tarantool.IterEq, []interface{}{pollID})
	if err != nil {
		r.l.Debug("failed to select poll", zap.Error(err))
		return []interface{}{}, fmt.Errorf("repository: database select error: %w", err)
	}
	r.l.Debug("tarantool response",
		zap.Uint32("status_code", existencePoll.Code),
		zap.Any("resp", existencePoll.Data),
		zap.String("error", existencePoll.Error))

	if len(existencePoll.Data) == 0 {
		r.l.Debug("poll not found", zap.String("poll_id", pollID))
		return []interface{}{}, models.ErrPollNotFound
	}
	pollTuple, ok := existencePoll.Data[0].([]interface{})
	if !ok {
		r.l.Debug("unexpected data type", zap.Any("data", existencePoll.Data))
		return []interface{}{}, models.ErrFailedToProcessData
	}
	return pollTuple, nil
}
