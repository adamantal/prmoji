package slack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	token string
	hc    *http.Client
	log   *slog.Logger
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
		hc: &http.Client{
			Timeout: 10 * time.Second,
		},
		log: slog.Default(),
	}
}

type slackAPIResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func (c *Client) AddReaction(ctx context.Context, channel, timestamp, emojiName string) error {
	c.log.Debug("adding reaction", "channel", channel, "timestamp", timestamp, "emoji", emojiName)

	form := url.Values{}
	form.Set("channel", channel)
	form.Set("timestamp", timestamp)
	form.Set("name", emojiName)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/reactions.add", strings.NewReader(form.Encode()))
	if err != nil {
		c.log.Error("failed to build slack request", "err", err, "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.hc.Do(req)
	if err != nil {
		c.log.Error("slack request failed", "err", err, "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return fmt.Errorf("slack reactions.add: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("failed reading slack response", "err", err, "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return fmt.Errorf("read slack response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.log.Error("slack http error", "status", resp.StatusCode, "body", string(b), "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return fmt.Errorf("slack http %d: %s", resp.StatusCode, string(b))
	}

	var apiResp slackAPIResponse
	if err := json.Unmarshal(b, &apiResp); err != nil {
		c.log.Error("failed decoding slack response", "err", err, "body", string(b), "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return fmt.Errorf("decode slack response: %w", err)
	}
	if apiResp.OK {
		c.log.Debug("reaction added", "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return nil
	}
	if apiResp.Error == "already_reacted" {
		c.log.Debug("reaction already present", "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return nil
	}
	if apiResp.Error == "message_not_found" {
		c.log.Warn("message not found", "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return nil
	}
	if apiResp.Error == "" {
		c.log.Error("slack api error", "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return errors.New("slack api error")
	}
	c.log.Error("slack api error", "error", apiResp.Error, "channel", channel, "timestamp", timestamp, "emoji", emojiName)
	return fmt.Errorf("slack api error: %s", apiResp.Error)
}
