package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adamantal/prmoji/internal/cleanup"
	"github.com/adamantal/prmoji/internal/config"
	httpHandlers "github.com/adamantal/prmoji/internal/http"
	"github.com/adamantal/prmoji/internal/log"
	"github.com/adamantal/prmoji/internal/slack"
	"github.com/adamantal/prmoji/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	logger := log.New(cfg.LogLevel)
	slog.SetDefault(logger)

	st, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		logger.Error("failed to init store", "err", err)
		os.Exit(1)
	}
	defer func() {
		_ = st.Close()
	}()

	slackClient := slack.NewClient(cfg.SlackToken)

	mux := http.NewServeMux()
	h := &httpHandlers.Handlers{Cfg: cfg, Store: st, Slack: slackClient, Log: logger}
	h.Register(mux)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				_, err := cleanup.Run(runCtx, st, cfg.RetentionDays, time.Now())
				cancel()
				if err != nil {
					logger.Error("background cleanup failed", "err", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		logger.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
