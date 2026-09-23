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

const defaultBaseURL = "https://slack.com/api"

type Client struct {
	token   string
	baseURL string
	hc      *http.Client
	log     *slog.Logger
}

func NewClient(token string) *Client {
	return &Client{
		token:   token,
		baseURL: defaultBaseURL,
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
	return c.react(ctx, "reactions.add", channel, timestamp, emojiName, "already_reacted", "reaction added")
}

func (c *Client) RemoveReaction(ctx context.Context, channel, timestamp, emojiName string) error {
	c.log.Debug("removing reaction", "channel", channel, "timestamp", timestamp, "emoji", emojiName)
	return c.react(ctx, "reactions.remove", channel, timestamp, emojiName, "no_reaction", "reaction removed")
}

// react posts to a Slack reactions endpoint. benignErr is the API error code
// that means the message is already in the wanted state.
func (c *Client) react(ctx context.Context, method, channel, timestamp, emojiName, benignErr, okMsg string) error {
	form := url.Values{}
	form.Set("channel", channel)
	form.Set("timestamp", timestamp)
	form.Set("name", emojiName)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/"+method, strings.NewReader(form.Encode()))
	if err != nil {
		c.log.Error("failed to build slack request", "err", err, "method", method, "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.hc.Do(req)
	if err != nil {
		c.log.Error("slack request failed", "err", err, "method", method, "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return fmt.Errorf("slack %s: %w", method, err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("failed reading slack response", "err", err, "method", method, "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return fmt.Errorf("read slack response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.log.Error("slack http error", "status", resp.StatusCode, "body", string(b), "method", method, "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return fmt.Errorf("slack http %d: %s", resp.StatusCode, string(b))
	}

	var apiResp slackAPIResponse
	if err := json.Unmarshal(b, &apiResp); err != nil {
		c.log.Error("failed decoding slack response", "err", err, "body", string(b), "method", method, "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return fmt.Errorf("decode slack response: %w", err)
	}
	if apiResp.OK {
		c.log.Debug(okMsg, "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return nil
	}
	if apiResp.Error == benignErr {
		c.log.Debug("reaction already in wanted state", "method", method, "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return nil
	}
	if apiResp.Error == "message_not_found" {
		c.log.Warn("message not found", "method", method, "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return nil
	}
	if apiResp.Error == "" {
		c.log.Error("slack api error", "method", method, "channel", channel, "timestamp", timestamp, "emoji", emojiName)
		return errors.New("slack api error")
	}
	c.log.Error("slack api error", "error", apiResp.Error, "method", method, "channel", channel, "timestamp", timestamp, "emoji", emojiName)
	return fmt.Errorf("slack api error: %s", apiResp.Error)
}
