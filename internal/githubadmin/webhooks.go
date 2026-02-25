package githubadmin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/google/go-github/v82/github"
	"golang.org/x/oauth2"
)

// SetupOptions holds configuration for configuring webhooks across an org.
type SetupOptions struct {
	Org             string
	URL             string
	Token           string
	IncludePattern  string
	ExcludePattern  string
	IncludeForks    bool
	IncludeArchived bool
	DryRun          bool
	Secret          string
	Verbose         bool

	// APIBaseURL optionally overrides the GitHub API base URL (e.g. for GitHub Enterprise).
	// When empty, the default public GitHub API is used.
	APIBaseURL string
}

// SetupOrgWebhooks lists repositories in the given org and ensures each selected repo
// has a webhook pointing at the target URL with the expected configuration.
func SetupOrgWebhooks(ctx context.Context, opts SetupOptions, logger *slog.Logger) error {
	if opts.Org == "" {
		return fmt.Errorf("org is required")
	}
	if opts.URL == "" {
		return fmt.Errorf("url is required")
	}
	if opts.Token == "" {
		return fmt.Errorf("token is required")
	}

	if logger == nil {
		logger = slog.Default()
	}

	var includeRe, excludeRe *regexp.Regexp
	var err error
	if opts.IncludePattern != "" {
		includeRe, err = regexp.Compile(opts.IncludePattern)
		if err != nil {
			return fmt.Errorf("invalid include-pattern %q: %w", opts.IncludePattern, err)
		}
	}
	if opts.ExcludePattern != "" {
		excludeRe, err = regexp.Compile(opts.ExcludePattern)
		if err != nil {
			return fmt.Errorf("invalid exclude-pattern %q: %w", opts.ExcludePattern, err)
		}
	}

	client, err := newGitHubClient(ctx, opts.Token, opts.APIBaseURL)
	if err != nil {
		return err
	}

	repos, err := listOrgRepos(ctx, client, opts.Org)
	if err != nil {
		return err
	}

	var (
		selectedRepos int
		created       int
		updated       int
		skipped       int
		filtered      int
		failed        int
	)

	// Pre-compute how many repositories will actually be processed so we can emit
	// simple progress information.
	totalSelected := 0
	for _, r := range repos {
		selected, _ := shouldSelectRepo(r, opts, includeRe, excludeRe)
		if selected {
			totalSelected++
		}
	}

	processedSelected := 0

	cancelled := false

	for _, r := range repos {
		select {
		case <-ctx.Done():
			// Context was cancelled or deadline exceeded; break and emit a summary
			// based on work completed so far.
			cancelled = true
			break
		default:
		}

		name := r.GetName()
		owner := r.GetOwner().GetLogin()

		selected, reason := shouldSelectRepo(r, opts, includeRe, excludeRe)
		if !selected {
			filtered++
			if opts.Verbose {
				logger.Info("skipping repo", "repo", name, "reason", reason)
			}
			continue
		}

		selectedRepos++

		action, err := ensureRepoWebhook(ctx, client, owner, name, opts)
		if err != nil {
			// If the context was cancelled while processing this repository, stop
			// immediately without treating this as a per-repo failure.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				cancelled = true
				break
			}
			failed++
			logger.Error("failed to ensure webhook", "repo", name, "err", err)
			continue
		}

		switch action {
		case "created":
			created++
		case "updated":
			updated++
		case "skipped":
			skipped++
		}

		if opts.Verbose {
			logger.Info("processed repo", "repo", name, "action", action)
		}

		processedSelected++
		if totalSelected > 0 {
			percent := float64(processedSelected) / float64(totalSelected) * 100
			logger.Info("progress",
				"org", opts.Org,
				"repos_done", processedSelected,
				"repos_total", totalSelected,
				"percent", fmt.Sprintf("%.1f", percent),
			)
		}

		// Basic throttling to avoid hitting rate limits too aggressively.
		time.Sleep(200 * time.Millisecond)
	}

	logger.Info("github webhook setup summary",
		"org", opts.Org,
		"repos_total", len(repos),
		"repos_selected", selectedRepos,
		"repos_filtered", filtered,
		"created", created,
		"updated", updated,
		"skipped", skipped,
		"failed", failed,
		"cancelled", cancelled,
	)

	if cancelled {
		return ctx.Err()
	}
	if failed > 0 {
		return fmt.Errorf("webhook setup failed for %d repositories", failed)
	}

	return nil
}

func newGitHubClient(ctx context.Context, token, baseURL string) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	if baseURL == "" {
		return github.NewClient(tc), nil
	}

	client, err := github.NewEnterpriseClient(baseURL, "", tc)
	if err != nil {
		return nil, fmt.Errorf("create github client: %w", err)
	}
	return client, nil
}

func listOrgRepos(ctx context.Context, client *github.Client, org string) ([]*github.Repository, error) {
	var all []*github.Repository
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, org, opt)
		if err != nil {
			return nil, fmt.Errorf("list org repos: %w", err)
		}
		all = append(all, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return all, nil
}

// ensureRepoWebhook makes sure the given repo has the desired webhook.
// It returns one of: "created", "updated", "skipped".
func ensureRepoWebhook(ctx context.Context, client *github.Client, owner, repo string, opts SetupOptions) (string, error) {
	hooks, err := listRepoHooks(ctx, client, owner, repo)
	if err != nil {
		return "", err
	}

	desiredEvents := []string{"issue_comment", "pull_request", "pull_request_review", "pull_request_review_comment"}
	desiredConfig := &github.HookConfig{
		URL:         github.Ptr(opts.URL),
		ContentType: github.Ptr("json"),
	}
	if opts.Secret != "" {
		desiredConfig.Secret = github.Ptr(opts.Secret)
	}

	var existing *github.Hook
	for _, h := range hooks {
		if h == nil || h.Config == nil {
			continue
		}
		if h.Config.GetURL() == opts.URL {
			existing = h
			break
		}
	}

	if existing == nil {
		if opts.DryRun {
			return "created", nil
		}

		hook := &github.Hook{
			Name:   github.String("web"),
			Events: desiredEvents,
			Active: github.Bool(true),
			Config: desiredConfig,
		}
		_, _, err := client.Repositories.CreateHook(ctx, owner, repo, hook)
		if err != nil {
			return "", fmt.Errorf("create hook for %s/%s: %w", owner, repo, err)
		}
		return "created", nil
	}

	if hookMatches(existing, desiredEvents, opts) {
		return "skipped", nil
	}

	if opts.DryRun {
		return "updated", nil
	}

	update := &github.Hook{
		Events: desiredEvents,
		Active: github.Ptr(true),
		Config: desiredConfig,
	}
	_, _, err = client.Repositories.EditHook(ctx, owner, repo, existing.GetID(), update)
	if err != nil {
		return "", fmt.Errorf("update hook for %s/%s: %w", owner, repo, err)
	}
	return "updated", nil
}

func listRepoHooks(ctx context.Context, client *github.Client, owner, repo string) ([]*github.Hook, error) {
	var all []*github.Hook
	opt := &github.ListOptions{PerPage: 100}

	for {
		hooks, resp, err := client.Repositories.ListHooks(ctx, owner, repo, opt)
		if err != nil {
			return nil, fmt.Errorf("list hooks for %s/%s: %w", owner, repo, err)
		}
		all = append(all, hooks...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return all, nil
}

func hookMatches(h *github.Hook, desiredEvents []string, opts SetupOptions) bool {
	if h == nil || h.Config == nil {
		return false
	}

	if !stringSlicesEqualIgnoreOrder(h.Events, desiredEvents) {
		return false
	}

	if h.Config.GetURL() != opts.URL {
		return false
	}

	// GitHub's API uses "json" as the content_type value for JSON payloads.
	if ct := h.Config.GetContentType(); ct != "" && ct != "json" {
		return false
	}

	if opts.Secret != "" && h.Config.GetSecret() != opts.Secret {
		return false
	}

	return true
}

func stringSlicesEqualIgnoreOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	m := make(map[string]int, len(a))
	for _, s := range a {
		m[s]++
	}
	for _, s := range b {
		m[s]--
		if m[s] < 0 {
			return false
		}
	}
	return true
}

// shouldSelectRepo applies filtering rules to decide whether a repository
// should be processed. It returns a boolean and a short reason string when
// the repo is filtered out.
func shouldSelectRepo(r *github.Repository, opts SetupOptions, includeRe, excludeRe *regexp.Regexp) (bool, string) {
	name := r.GetName()

	if !opts.IncludeArchived && r.GetArchived() {
		return false, "archived"
	}
	if !opts.IncludeForks && r.GetFork() {
		return false, "fork"
	}
	if includeRe != nil && !includeRe.MatchString(name) {
		return false, "does not match include-pattern"
	}
	if excludeRe != nil && excludeRe.MatchString(name) {
		return false, "matches exclude-pattern"
	}

	return true, ""
}


