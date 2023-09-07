package main

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"time"
)

type (
	twitchFollowList []twitchFollow
	twitchFollow     struct {
		UserID     string    `json:"user_id"`
		UserLogin  string    `json:"user_login"`
		UserName   string    `json:"user_name"`
		FollowedAt time.Time `json:"followed_at"`
	}
)

func (t twitchFollowList) ToCSV() io.Reader {
	out := new(bytes.Buffer)

	sort.Slice(t, func(i, j int) bool { return t[i].UserLogin < t[j].UserLogin })

	fmt.Fprintln(out, "login,display_name,id,followed_at")
	for _, f := range t {
		fmt.Fprintf(
			out, "%q,%q,%s,%s\n",
			f.UserLogin,
			f.UserName,
			f.UserID,
			f.FollowedAt.Format(time.RFC3339),
		)
	}

	return out
}

type (
	twitchSubscriptionList []twitchSubscription
	twitchSubscription     struct {
		BroadcasterID    string `json:"broadcaster_id"`
		BroadcasterLogin string `json:"broadcaster_login"`
		BroadcasterName  string `json:"broadcaster_name"`
		GifterID         string `json:"gifter_id"`
		GifterLogin      string `json:"gifter_login"`
		GifterName       string `json:"gifter_name"`
		IsGift           bool   `json:"is_gift"`
		Tier             string `json:"tier"`
		PlanName         string `json:"plan_name"`
		UserID           string `json:"user_id"`
		UserName         string `json:"user_name"`
		UserLogin        string `json:"user_login"`
	}
)

func (t twitchSubscriptionList) ToCSV() io.Reader {
	out := new(bytes.Buffer)

	sort.Slice(t, func(i, j int) bool { return t[i].UserLogin < t[j].UserLogin })

	fmt.Fprintln(out, "login,display_name,id,tier,is_gift,gifter_login,gifter_name,gifter_id")
	for _, f := range t {
		fmt.Fprintf(
			out, "%q,%q,%s,%s,%v,%q,%q,%s\n",
			f.UserLogin,
			f.UserName,
			f.UserID,
			f.Tier,
			f.IsGift,
			f.GifterLogin,
			f.GifterName,
			f.GifterID,
		)
	}

	return out
}
