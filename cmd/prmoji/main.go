package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	neturl "net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/adamantal/prmoji/internal/cleanup"
	"github.com/adamantal/prmoji/internal/config"
	"github.com/adamantal/prmoji/internal/githubadmin"
	httpHandlers "github.com/adamantal/prmoji/internal/http"
	"github.com/adamantal/prmoji/internal/log"
	"github.com/adamantal/prmoji/internal/slack"
	"github.com/adamantal/prmoji/internal/store"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "prmoji",
		Short: "Add Slack emoji reactions based on GitHub PR activity",
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Start the prmoji HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer()
		},
	}

	webhookCmd := &cobra.Command{
		Use:   "github-webhook-setup",
		Short: "Configure GitHub webhooks for repositories in an organization",
		RunE:  runGitHubWebhookSetup,
	}

	webhookCmd.Flags().String("org", "", "GitHub organization name (required)")
	webhookCmd.Flags().String("url", "", "Webhook payload URL, e.g. https://YOUR_HOST/event/github (required)")
	webhookCmd.Flags().String("token", "", "GitHub personal access token (overrides GITHUB_TOKEN and GITHUB_PAT)")
	webhookCmd.Flags().String("secret", "", "Webhook secret (optional)")
	webhookCmd.Flags().String("include-pattern", "", "Regexp; only repo names matching this are included")
	webhookCmd.Flags().String("exclude-pattern", "", "Regexp; repo names matching this are excluded")
	webhookCmd.Flags().Bool("include-forks", false, "Include forked repositories")
	webhookCmd.Flags().Bool("include-archived", false, "Include archived repositories")
	webhookCmd.Flags().Bool("dry-run", false, "Do not make any changes, only print what would be done")
	webhookCmd.Flags().Bool("verbose", false, "Print per-repository actions")

	_ = webhookCmd.MarkFlagRequired("org")
	_ = webhookCmd.MarkFlagRequired("url")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(webhookCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServer() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := log.New(cfg.LogLevel)
	slog.SetDefault(logger)

	st, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		logger.Error("failed to init store", "err", err)
		return err
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
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	return nil
}

func runGitHubWebhookSetup(cmd *cobra.Command, args []string) error {
	org, _ := cmd.Flags().GetString("org")
	rawURL, _ := cmd.Flags().GetString("url")
	token, _ := cmd.Flags().GetString("token")
	secret, _ := cmd.Flags().GetString("secret")
	includePattern, _ := cmd.Flags().GetString("include-pattern")
	excludePattern, _ := cmd.Flags().GetString("exclude-pattern")
	includeForks, _ := cmd.Flags().GetBool("include-forks")
	includeArchived, _ := cmd.Flags().GetBool("include-archived")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")

	url := strings.TrimSpace(rawURL)
	if url == "" {
		return fmt.Errorf("--url is required")
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		// Default to https if the user omitted the scheme.
		url = "https://" + url
	}

	parsed, err := neturl.Parse(url)
	if err != nil {
		return fmt.Errorf("invalid --url %q: %w", url, err)
	}
	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/event/github"
	}
	url = parsed.String()

	tok := token
	if tok == "" {
		tok = os.Getenv("GITHUB_TOKEN")
	}
	if tok == "" {
		tok = os.Getenv("GITHUB_PAT")
	}
	if tok == "" {
		return fmt.Errorf("GitHub token is required (set --token or GITHUB_TOKEN/GITHUB_PAT)")
	}

	logger := log.New("info")
	slog.SetDefault(logger)

	baseCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx, cancel := context.WithTimeout(baseCtx, 10*time.Minute)
	defer cancel()

	opts := githubadmin.SetupOptions{
		Org:             org,
		URL:             url,
		Token:           tok,
		IncludePattern:  includePattern,
		ExcludePattern:  excludePattern,
		IncludeForks:    includeForks,
		IncludeArchived: includeArchived,
		DryRun:          dryRun,
		Secret:          secret,
		Verbose:         verbose,
	}

	if err := githubadmin.SetupOrgWebhooks(ctx, opts, logger); err != nil {
		// If the run was cancelled (e.g. Ctrl+C), we've already printed a summary in
		// SetupOrgWebhooks; suppress Cobra's usage output and exit cleanly.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil
		}
		return err
	}
	return nil
}
