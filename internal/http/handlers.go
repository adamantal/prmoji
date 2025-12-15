package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/adamantal/prmoji/internal/cleanup"
	"github.com/adamantal/prmoji/internal/config"
	"github.com/adamantal/prmoji/internal/github"
	"github.com/adamantal/prmoji/internal/slack"
	"github.com/adamantal/prmoji/internal/store"
	"github.com/adamantal/prmoji/internal/util"
)

type Handlers struct {
	Cfg   config.Config
	Store *store.SQLiteStore
	Slack *slack.Client
	Log   *slog.Logger
}

func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.handleOK)
	mux.HandleFunc("GET /healthz", h.handleOK)
	mux.HandleFunc("POST /event/slack", h.handleSlackEvent)
	mux.HandleFunc("POST /event/github", h.handleGitHubEvent)
	mux.HandleFunc("POST /cleanup/", h.handleCleanup)
}

func (h *Handlers) handleOK(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (h *Handlers) handleSlackEvent(w http.ResponseWriter, r *http.Request) {
	body, err := readBody(r, 1<<20)
	if err != nil {
		h.Log.Warn("read slack body failed", "err", err)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
		return
	}

	env, err := slack.ParseEnvelope(body)
	if err != nil {
		h.Log.Warn("parse slack payload failed", "err", err)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
		return
	}

	if strings.TrimSpace(env.Challenge) != "" {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(env.Challenge))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))

	go h.processSlackEvent(body)
}

func (h *Handlers) processSlackEvent(body []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	env, err := slack.ParseEnvelope(body)
	if err != nil {
		h.Log.Warn("parse slack payload failed", "err", err)
		return
	}
	if env.Event.Text == "" || env.Event.Channel == "" || env.Event.EventTS == "" {
		h.Log.Debug("discarding empty slack message", "event", env.Event)
		return
	}

	urls := slack.ExtractPRURLs(env.Event.Text)
	if len(urls) == 0 {
		h.Log.Debug("discarding slack message without PR URLs", "channel", env.Event.Channel, "text", env.Event.Text)
		return
	}

	h.Log.Debug("ingesting slack message with PR URLs", "channel", env.Event.Channel, "count", len(urls))
	for _, u := range urls {
		if err := h.Store.InsertPRMessage(ctx, u, env.Event.Channel, env.Event.EventTS); err != nil {
			h.Log.Error("insert pr message failed", "err", err, "pr_url", u)
		}
	}

	h.Log.Info("slack message ingested", "count", len(urls), "channel", env.Event.Channel)
}

func (h *Handlers) handleGitHubEvent(w http.ResponseWriter, r *http.Request) {
	body, err := readBody(r, 2<<20)
	if err != nil {
		h.Log.Warn("read github body failed", "err", err)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))

	go h.processGitHubEvent(eventType, body)
}

func (h *Handlers) processGitHubEvent(eventType string, body []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	class, ok := github.Classify(eventType, body)
	if !ok {
		return
	}
	if class.PRURL == "" {
		return
	}

	if class.Action == github.ActionCommented {
		who := strings.ToLower(strings.TrimSpace(class.Commenter))
		for _, ignored := range h.Cfg.IgnoredCommenters {
			if who != "" && who == ignored {
				h.Log.Info("suppressed comment reaction", "pr_url", class.PRURL, "commenter", who)
				return
			}
		}
	}

	emoji := util.EmojiForAction(class.Action)
	msgs, err := h.Store.ListMessagesByPRURL(ctx, class.PRURL)
	if err != nil {
		h.Log.Error("list messages failed", "err", err, "pr_url", class.PRURL)
		return
	}
	if len(msgs) == 0 {
		return
	}

	for _, m := range msgs {
		if err := h.Slack.AddReaction(ctx, m.MessageChannel, m.MessageTimestamp, emoji); err != nil {
			h.Log.Error("add reaction failed", "err", err, "pr_url", class.PRURL, "channel", m.MessageChannel, "ts", m.MessageTimestamp, "emoji", emoji)
		}
	}

	if class.Action == github.ActionMerged || class.Action == github.ActionClosed {
		if err := h.Store.DeleteByPRURL(ctx, class.PRURL); err != nil {
			h.Log.Error("delete mappings failed", "err", err, "pr_url", class.PRURL)
		}
	}

	h.Log.Info("processed github event", "event", eventType, "action", string(class.Action), "pr_url", class.PRURL, "messages", len(msgs))
}

func (h *Handlers) handleCleanup(w http.ResponseWriter, r *http.Request) {
	_ = r
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := cleanup.Run(ctx, h.Store, h.Cfg.RetentionDays, time.Now())
	if err != nil {
		h.Log.Error("cleanup failed", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func readBody(r *http.Request, limit int64) ([]byte, error) {
	defer r.Body.Close()
	lr := io.LimitReader(r.Body, limit)
	return io.ReadAll(lr)
}
