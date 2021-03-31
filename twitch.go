package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Luzifer/go_helpers/v2/backoff"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const twitchRequestTimeout = 5 * time.Second

var twitch = newTwitchClient()

type twitchClient struct{}

func newTwitchClient() *twitchClient {
	return &twitchClient{}
}

func (t twitchClient) GetFollowers() (twitchFollowList, error) {
	uid, _, err := t.getAuthorizedUser()
	if err != nil {
		return nil, errors.Wrap(err, "get logged-in user")
	}

	ctx, cancel := context.WithTimeout(context.Background(), twitchRequestTimeout)
	defer cancel()

	var (
		out     []twitchFollow
		payload struct {
			Data       []twitchFollow `json:"data"`
			Pagination struct {
				Cursor string `json:"cursor"`
			} `json:"pagination"`
		}
	)

	for {
		params := make(url.Values)
		params.Set("to_id", uid)
		params.Set("after", payload.Pagination.Cursor)
		payload.Pagination.Cursor = ""

		if err = backoff.NewBackoff().WithMaxIterations(3).Retry(func() error {
			return errors.Wrap(
				t.request(ctx, http.MethodGet, fmt.Sprintf("https://api.twitch.tv/helix/users/follows?%s", params.Encode()), nil, &payload),
				"requesting follows",
			)
		}); err != nil {
			return nil, err
		}

		out = append(out, payload.Data...)

		if payload.Pagination.Cursor == "" {
			break
		}
	}

	return out, nil
}

func (t twitchClient) GetSubscriptions() (twitchSubscriptionList, error) {
	uid, _, err := t.getAuthorizedUser()
	if err != nil {
		return nil, errors.Wrap(err, "get logged-in user")
	}

	ctx, cancel := context.WithTimeout(context.Background(), twitchRequestTimeout)
	defer cancel()

	var (
		out     []twitchSubscription
		payload struct {
			Data       []twitchSubscription `json:"data"`
			Pagination struct {
				Cursor string `json:"cursor"`
			} `json:"pagination"`
		}
	)

	for {
		params := make(url.Values)
		params.Set("broadcaster_id", uid)
		params.Set("after", payload.Pagination.Cursor)
		payload.Pagination.Cursor = ""

		if err = backoff.NewBackoff().WithMaxIterations(3).Retry(func() error {
			return errors.Wrap(
				t.request(ctx, http.MethodGet, fmt.Sprintf("https://api.twitch.tv/helix/subscriptions?%s", params.Encode()), nil, &payload),
				"requesting subscriptions",
			)
		}); err != nil {
			return nil, err
		}

		out = append(out, payload.Data...)

		if payload.Pagination.Cursor == "" {
			break
		}
	}

	return out, nil
}

func (t twitchClient) getAuthorizedUser() (id, username string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), twitchRequestTimeout)
	defer cancel()

	var payload struct {
		Data []struct {
			ID    string `json:"id"`
			Login string `json:"login"`
		} `json:"data"`
	}

	if err := t.request(ctx, http.MethodGet, "https://api.twitch.tv/helix/users", nil, &payload); err != nil {
		return "", "", errors.Wrap(err, "request channel info")
	}

	if l := len(payload.Data); l != 1 {
		return "", "", errors.Errorf("unexpected number of users returned: %d", l)
	}

	return payload.Data[0].ID, payload.Data[0].Login, nil
}

func (twitchClient) request(ctx context.Context, method, url string, body io.Reader, out interface{}) error {
	log.WithFields(log.Fields{
		"method": method,
		"url":    url,
	}).Trace("Execute Twitch API request")

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return errors.Wrap(err, "assemble request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Id", cfg.TwitchClientID)
	req.Header.Set("Authorization", "Bearer "+cfg.TwitchToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "execute request")
	}
	defer resp.Body.Close()

	return errors.Wrap(
		json.NewDecoder(resp.Body).Decode(out),
		"parse user info",
	)
}
