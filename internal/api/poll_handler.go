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
	s      *service.PollService
	l      *zap.Logger
	client *model.Client4
}

func New(s *service.PollService, l *zap.Logger, client *model.Client4) *PollHandler {
	return &PollHandler{
		s:      s,
		l:      l,
		client: client,
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
		err = h.SendMsg(HelpMessage, post.ChannelId)
		return
	}
	h.l.Info("new request for the bot",
		zap.String("command", args[0]),
		zap.String("action", args[1]),
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
		respPost := &model.PostEphemeral{UserID: post.UserId}
		if len(args) <= 4 {
			respPost.Post = &model.Post{ChannelId: post.ChannelId,
				Message: HelpMessage}
			_, _, _ = h.client.CreatePostEphemeral(respPost)
			return
		}
		err = h.CreatePoll(createArgs[0], post.UserId, post.ChannelId, createArgs[1:])
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
			}
			_, _, _ = h.client.CreatePostEphemeral(errPost)
			return
		}
	case "vote":
		respPost := &model.PostEphemeral{UserID: post.UserId}
		h.l.Debug("len args", zap.Int("len(args)", len(args)))
		if len(args) != 4 {
			respPost.Post = &model.Post{ChannelId: post.ChannelId,
				Message: HelpMessage}
			_, _, err = h.client.CreatePostEphemeral(respPost)
			if err != nil {
				h.l.Error("error sending message", zap.Error(err))
				return
			}
			return
		}
		err = h.Vote(args[2], args[3], post.UserId)
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
			case errors.Is(err, models.ErrPollIsEnd):
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: fmt.Sprintf("poll with id: %s is ended", args[2])}
			default:
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: "somthing went wrong"}
			}
			_, _, _ = h.client.CreatePostEphemeral(respPost)
			return
		}
		respPost.Post = &model.Post{ChannelId: post.ChannelId,
			Message: "your vote successfully written"}
		_, _, _ = h.client.CreatePostEphemeral(respPost)
	case "result":
		respPost := &model.PostEphemeral{UserID: post.UserId}
		if len(args) != 3 {
			respPost.Post = &model.Post{ChannelId: post.ChannelId,
				Message: HelpMessage}
			_, _, _ = h.client.CreatePostEphemeral(respPost)
			return
		}
		err = h.GetPollResult(args[2], post.ChannelId)
		if err != nil {
			switch {
			case errors.Is(err, models.ErrPollNotFound):
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: fmt.Sprintf("not found poll with id: %s", args[2])}
			default:
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: "somthing went wrong"}
			}
			_, _, _ = h.client.CreatePostEphemeral(respPost)
			return
		}
	case "end":
		respPost := &model.PostEphemeral{UserID: post.UserId}
		if len(args) != 3 {
			respPost.Post = &model.Post{ChannelId: post.ChannelId,
				Message: HelpMessage}
			_, _, _ = h.client.CreatePostEphemeral(respPost)
			return
		}
		err = h.EndPoll(args[2], post.UserId)
		if err != nil {
			switch {
			case errors.Is(err, models.ErrPollNotFound):
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: fmt.Sprintf("not found poll with id: %s", args[2])}
			case errors.Is(err, models.ErrUserNotOwner):
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: err.Error()}
			case errors.Is(err, models.ErrPollAlreadyEnded):
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: err.Error()}
			default:
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: "somthing went wrong"}
			}
			_, _, _ = h.client.CreatePostEphemeral(respPost)
			return
		}
		respPost.Post = &model.Post{ChannelId: post.ChannelId,
			Message: "poll successfully ended"}
		_, _, _ = h.client.CreatePostEphemeral(respPost)
	case "delete":
		respPost := &model.PostEphemeral{UserID: post.UserId}
		if len(args) != 3 {
			respPost.Post = &model.Post{ChannelId: post.ChannelId,
				Message: HelpMessage}
			_, _, _ = h.client.CreatePostEphemeral(respPost)
			return
		}
		err = h.DeletePoll(args[2], post.UserId)
		if err != nil {
			switch {
			case errors.Is(err, models.ErrPollNotFound):
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: fmt.Sprintf("not found poll with id: %s", args[2])}
			case errors.Is(err, models.ErrUserNotOwner):
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: err.Error()}
			default:
				respPost.Post = &model.Post{ChannelId: post.ChannelId,
					Message: "somthing went wrong"}
			}
			_, _, _ = h.client.CreatePostEphemeral(respPost)
			return
		}
		respPost.Post = &model.Post{ChannelId: post.ChannelId,
			Message: "poll successfully deleted"}
		_, _, _ = h.client.CreatePostEphemeral(respPost)
	default:
		err = h.SendMsg(HelpMessage, post.ChannelId)
	}

}

func (h *PollHandler) CreatePoll(question, creatorID, channelID string, optionsRaw []string) error {
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
	if err = h.SendMsg(message, channelID); err != nil {
		h.l.Error("failed sending poll message", zap.Error(err))
		return fmt.Errorf("handler: failed to send message: %w", err)
	}
	h.l.Info("successfully created poll",
		zap.String("poll_id", id),
		zap.String("question", question),
		zap.Any("options", options))
	return nil
}

func (h *PollHandler) GetPollResult(pollID, channelID string) error {
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
		h.l.Error("failed getting poll result",
			zap.String("poll_id", pollID),
			zap.Error(err))
		return fmt.Errorf("handler: failed to get poll result: %w", err)
	}
	message := fmt.Sprintf("**Question**: %s\n", question)
	for _, option := range options {
		message += fmt.Sprintf("  [%d] votes: **%d** (*%s*)\n",
			option.ID, votes[strconv.Itoa(option.ID)], option.Text)
	}
	if err = h.SendMsg(message, channelID); err != nil {
		h.l.Error("error sending message", zap.Error(err))
		return err
	}
	h.l.Info("successfully sent poll result",
		zap.String("poll_id", pollID))
	return nil
}

func (h *PollHandler) Vote(pollID, choiceID, userID string) error {
	h.l.Debug("data for voting",
		zap.String("poll_id", pollID),
		zap.String("choice_id", choiceID))
	err := h.s.Vote(pollID, choiceID, userID)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrPollNotFound):
			h.l.Warn("poll not found", zap.String("poll_id", pollID))
			return err
		case errors.Is(err, models.ErrOptionIsNotFound):
			h.l.Warn("option not found", zap.String("choice_id", choiceID))
			return err
		case errors.Is(err, models.ErrVoteAlreadyExists):
			h.l.Warn("vote already exists",
				zap.String("poll_id", pollID),
				zap.String("choice_id", choiceID))
			return err
		case errors.Is(err, models.ErrPollIsEnd):
			h.l.Warn("poll is ended",
				zap.String("poll_id", pollID),
				zap.String("choice_id", choiceID))
			return err
		default:
			h.l.Error("failed to vote",
				zap.String("poll_id", pollID),
				zap.String("choice_id", choiceID),
				zap.Error(err))
			return fmt.Errorf("handler: failed to vote: %w", err)
		}
	}

	h.l.Info("voted successfully",
		zap.String("poll_id", pollID),
		zap.String("user_id", userID),
		zap.String("choice_id", choiceID))
	return nil
}

func (h *PollHandler) EndPoll(pollID, userID string) error {
	h.l.Debug("data for ending poll",
		zap.String("poll_id", pollID),
		zap.String("user_id", userID))
	err := h.s.EndPoll(pollID, userID)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrPollNotFound):
			h.l.Warn("poll not found", zap.String("poll_id", pollID))
			return err
		case errors.Is(err, models.ErrUserNotOwner):
			h.l.Warn("user is not owner of poll",
				zap.String("poll_id", pollID),
				zap.String("user_id", userID))
			return err
		case errors.Is(err, models.ErrPollAlreadyEnded):
			h.l.Warn("poll already ended",
				zap.String("poll_id", pollID),
				zap.String("user_id", userID))
			return err
		default:
			h.l.Error("failed to end poll",
				zap.String("poll_id", pollID),
				zap.String("user_id", userID),
				zap.Error(err))
			return fmt.Errorf("handler: failed to end poll: %w", err)
		}
	}
	h.l.Info("successfully ended poll",
		zap.String("poll_id", pollID),
		zap.String("user_id", userID))
	return nil
}

func (h *PollHandler) DeletePoll(pollID, userID string) error {
	h.l.Debug("data for deleting poll",
		zap.String("poll_id", pollID),
		zap.String("user_id", userID))
	err := h.s.DeletePoll(pollID, userID)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrPollNotFound):
			h.l.Warn("poll not found", zap.String("poll_id", pollID))
			return err
		case errors.Is(err, models.ErrUserNotOwner):
			h.l.Warn("user is not owner of poll",
				zap.String("poll_id", pollID),
				zap.String("user_id", userID))
			return err
		default:
			h.l.Error("failed to delete poll",
				zap.String("poll_id", pollID),
				zap.String("user_id", userID),
				zap.Error(err))
			return fmt.Errorf("handler: failed to delete poll: %w", err)
		}
	}
	h.l.Info("successfully deleted poll",
		zap.String("poll_id", pollID),
		zap.String("user_id", userID))
	return nil
}

func (h *PollHandler) SendMsg(message, channelID string) error {
	post := &model.Post{
		ChannelId: channelID,
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
