package service

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jaam8/mattermost_bot/internal/models"
	"github.com/jaam8/mattermost_bot/internal/repository"
	"go.uber.org/zap"
	"strconv"
)

type PollService struct {
	r repository.PollRepository
	l *zap.Logger
}

func New(r *repository.PollRepository, l *zap.Logger) *PollService {
	return &PollService{
		r: *r,
		l: l,
	}
}

func (s *PollService) CreatePoll(question, creatorID string, optionsRaw []string) (string, []models.Option, error) {
	s.l.Debug("creating poll", zap.String("question", question), zap.String("creatorID", creatorID), zap.Strings("options", optionsRaw))
	options := make([]models.Option, len(optionsRaw))
	votes := make(map[string]int)
	for i, option := range optionsRaw {
		if len(option) < 1 {
			return "", nil, models.ErrOptionIsEmpty
		}
		options[i] = models.Option{
			ID:   i + 1,
			Text: option,
		}
		votes[strconv.Itoa(options[i].ID)] = 0
	}

	poll := &models.Poll{
		ID:        uuid.New().String()[:8],
		Question:  question,
		Options:   options,
		Votes:     votes,
		CreatorID: creatorID,
		IsActive:  true,
	}

	id, options, err := s.r.CreatePoll(poll)
	if err != nil {
		s.l.Error("failed to create poll", zap.Error(err))
		return "", nil, fmt.Errorf("service: failed to create poll: %w", err)
	}
	return id, options, nil
}

func (s *PollService) Vote(pollID, choiceID, userID string) error {
	err := s.r.Vote(pollID, choiceID, userID)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrPollNotFound):
			return models.ErrPollNotFound
		case errors.Is(err, models.ErrVoteAlreadyExists):
			return models.ErrVoteAlreadyExists
		case errors.Is(err, models.ErrOptionIsNotFound):
			return models.ErrOptionIsNotFound
		default:
			s.l.Error("failed to vote", zap.Error(err))
			return fmt.Errorf("service: failed to vote: %w", err)
		}
	}
	return nil
}

func (s *PollService) GetPollResult(pollID string) (string, []models.Option, map[string]int, error) {
	poll, err := s.r.GetPollResult(pollID)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrPollNotFound):
			return "", nil, nil, models.ErrPollNotFound
		case errors.Is(err, models.ErrFailedToProcessData):
			return "", nil, nil, models.ErrFailedToProcessData
		default:
			s.l.Error("error getting poll result", zap.Error(err))
			return "", nil, nil, fmt.Errorf("service: failed to get poll result: %w", err)
		}
	}

	return poll.Question, poll.Options, poll.Votes, nil
}
