package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jaam8/mattermost_bot/internal/models"
	"github.com/jaam8/mattermost_bot/internal/service"
	"github.com/mattermost/mattermost-server/v6/model"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

const (
	COMMAND     = "/poll"
	HelpMessage = "i know only this command:\n- `/poll create \"question\" \"option1\" \"option2\" \"optionN\"`\n- `/poll vote poll_id choice_id`\n- `/poll result poll_id`\n- `/poll end poll_id`\n- `/poll delete poll_id`\n- `/poll help`"
)

type PollHandler struct {
	s         *service.PollService
	l         *zap.Logger
	client    *model.Client4
	channelID string
}

func New(s *service.PollService, l *zap.Logger, client *model.Client4, channelID string) *PollHandler {
	return &PollHandler{
		s:         s,
		l:         l,
		client:    client,
		channelID: channelID,
	}
}

func HandleMessage(h *PollHandler, event *model.WebSocketEvent, botID string) {
	post := &model.Post{}
	err := json.Unmarshal([]byte(event.GetData()["post"].(string)), &post)
	if err != nil {
		h.l.Error("error unmarshalling post", zap.Error(err))
		return
	}
	if post.UserId == botID {
		return
	}
	if post == nil {
		h.l.Error("post is nil")
		return
	}

	args := strings.Fields(post.Message)
	if args[0] != COMMAND {
		//h.l.Error("this HelpMessage has not command")
		return
	}
	if len(args) < 2 {
		err = h.SendMsg(HelpMessage)
		return
	}
	h.l.Info("new request for the bot",
		zap.String("command", args[0]),
		zap.String("subcommand", args[1]),
		zap.String("user_id", post.UserId),
		zap.String("channel_id", post.ChannelId),
		zap.String("message", post.Message))
	switch args[1] {
	case "create":
		createArgs := []string{}
		for i, val := range strings.Split(post.Message, "\"") {
			if i%2 != 0 {
				createArgs = append(createArgs, strings.Trim(val, "\""))
			}
		}
		if len(createArgs) < 2 {
			createArgs = append(createArgs, "hack")
		}
		err = h.CreatePoll(createArgs[0], post.UserId, createArgs[1:])
		if err != nil {
			errPost := &model.PostEphemeral{UserID: post.UserId}
			switch {
			case errors.Is(err, models.ErrNotEnoughOptions):
				errPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: err.Error()}
			case errors.Is(err, models.ErrOptionIsEmpty):
				errPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: err.Error()}
			case errors.Is(err, models.ErrQuestionIsEmpty):
				errPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: err.Error()}
			default:
				errPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: "somthing went wrong"}
				h.l.Error("failed to create poll", zap.Error(err))
			}
			_, _, _ = h.client.CreatePostEphemeral(errPost)
		}
	case "vote":
		err = h.Vote(args[2], args[3], post.UserId)
		respPost := &model.PostEphemeral{UserID: post.UserId}
		if err != nil {
			switch {
			case errors.Is(err, models.ErrPollNotFound):
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: fmt.Sprintf("not found poll with id: %s", args[2])}
			case errors.Is(err, models.ErrOptionIsNotFound):
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: fmt.Sprintf("not found option with id: %s", args[3])}
			case errors.Is(err, models.ErrVoteAlreadyExists):
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: err.Error()}
			default:
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: "somthing went wrong"}
				h.l.Error("failed to vote", zap.Error(err))
			}
			_, _, _ = h.client.CreatePostEphemeral(respPost)
			return
		}
		respPost.Post = &model.Post{ChannelId: post.ChannelId,
			Message: "your vote successfully written"}
		_, _, _ = h.client.CreatePostEphemeral(respPost)
	case "result":
		err = h.GetPollResult(args[2])
		if err != nil {
			errPost := &model.PostEphemeral{UserID: post.UserId}
			switch {
			case errors.Is(err, models.ErrPollNotFound):
				errPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: fmt.Sprintf("not found poll with id: %s", args[2])}
			default:
				errPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: "somthing went wrong"}
				h.l.Error("failed to get poll", zap.Error(err))
			}
			_, _, _ = h.client.CreatePostEphemeral(errPost)
		}
	case "end":
		err = h.EndPoll(args[2])
	case "delete":
		err = h.DeletePoll(args[2])
	default:
		err = h.SendMsg(HelpMessage)
	}

}

func (h *PollHandler) CreatePoll(question, creatorID string, optionsRaw []string) error {
	if len(question) < 1 {
		return models.ErrQuestionIsEmpty
	}
	if len(optionsRaw) < 2 {
		return models.ErrNotEnoughOptions
	}
	h.l.Debug("data for creating new poll",
		zap.String("question", question),
		zap.String("creator_id", creatorID),
		zap.Strings("options", optionsRaw))
	id, options, err := h.s.CreatePoll(question, creatorID, optionsRaw)
	if err != nil {
		if errors.Is(err, models.ErrOptionIsEmpty) {
			h.l.Warn("option is empty")
			return err
		}
		h.l.Error("failed creating poll", zap.Error(err))
		return fmt.Errorf("handler: failed to create poll: %w", err)
	}
	message := fmt.Sprintf("**Poll ID**: %s\n**Question**: %s\n**Options**:\n", id, question)
	for _, option := range options {
		message += fmt.Sprintf("  [%d] *%s*\n", option.ID, option.Text)
	}
	if err = h.SendMsg(message); err != nil {
		h.l.Error("failed sending poll message", zap.Error(err))
		return fmt.Errorf("handler: failed to send message: %w", err)
	}
	h.l.Info("successfully created poll",
		zap.String("poll_id", id),
		zap.String("question", question),
		zap.Any("options", options))
	return nil
}

func (h *PollHandler) GetPollResult(pollID string) error {
	question, options, votes, err := h.s.GetPollResult(pollID)
	h.l.Debug("data for getting poll result",
		zap.String("poll_id", pollID),
		zap.String("question", question),
		zap.Any("options", options),
		zap.Any("votes", votes))
	if err != nil {
		if errors.Is(err, models.ErrPollNotFound) {
			h.l.Warn("poll not found", zap.String("poll_id", pollID))
			return err
		}
		h.l.Error("failed getting poll result", zap.String("poll_id", pollID), zap.Error(err))
		return fmt.Errorf("handler: failed to get poll result: %w", err)
	}
	message := fmt.Sprintf("**Question**: %s\n", question)
	for _, option := range options {
		message += fmt.Sprintf("  [%d] votes: **%d** (*%s*)\n", option.ID, votes[strconv.Itoa(option.ID)], option.Text)
	}
	if err = h.SendMsg(message); err != nil {
		h.l.Error("error sending message", zap.Error(err))
		return err
	}
	h.l.Info("successfully sent poll result",
		zap.String("poll_id", pollID))
	return nil
}

func (h *PollHandler) Vote(pollID, choiceID, userID string) error {
	// todo check if the user has already voted
	// todo add vote to vote space
	h.l.Debug("data for voting",
		zap.String("poll_id", pollID),
		zap.String("choice_id", choiceID))
	err := h.s.Vote(pollID, choiceID, userID)
	if err != nil {
		if errors.Is(err, models.ErrPollNotFound) {
			h.l.Warn("poll not found", zap.String("poll_id", pollID))
			return err
		}
		if errors.Is(err, models.ErrOptionIsNotFound) {
			h.l.Warn("option not found", zap.String("choice_id", choiceID))
			return err
		}
		if errors.Is(err, models.ErrVoteAlreadyExists) {
			h.l.Warn("vote already exists",
				zap.String("poll_id", pollID),
				zap.String("choice_id", choiceID))
			return err
		}
		h.l.Error("failed to vote",
			zap.String("poll_id", pollID),
			zap.String("choice_id", choiceID),
			zap.Error(err))
		return fmt.Errorf("handler: failed to vote: %w", err)
	}

	h.l.Info("voted successfully",
		zap.String("poll_id", pollID),
		zap.String("user_id", userID),
		zap.String("choice_id", choiceID))
	return nil
}

func (h *PollHandler) EndPoll(pollID string) error {
	return nil
}

func (h *PollHandler) DeletePoll(pollID string) error {
	return nil
}

func (h *PollHandler) SendMsg(message string) error {
	post := &model.Post{
		ChannelId: h.channelID,
		Message:   message,
	}
	req, resp, err := h.client.CreatePost(post)
	h.l.Debug("send new message",
		zap.String("channel_id", req.ChannelId),
		zap.String("message", req.Message),
		zap.Int("status_code", resp.StatusCode))
	if err != nil {
		return err
	}
	return nil
}
