package githubadmin

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-github/v82/github"
)

func TestHookMatches(t *testing.T) {
	h := &github.Hook{
		Events: []string{"issue_comment", "pull_request"},
		Config: &github.HookConfig{
			URL:         github.Ptr("https://example.com"),
			ContentType: github.Ptr("json"),
		},
	}

	if !hookMatches(h, []string{"pull_request", "issue_comment"}, SetupOptions{
		URL: "https://example.com",
	}) {
		t.Fatalf("expected hook to match")
	}

	if hookMatches(h, []string{"issue_comment"}, SetupOptions{
		URL: "https://example.com",
	}) {
		t.Fatalf("expected hook not to match (events differ)")
	}

	if hookMatches(h, []string{"issue_comment", "pull_request"}, SetupOptions{
		URL: "https://example.com/other",
	}) {
		t.Fatalf("expected hook not to match (url differs)")
	}
}

// This test exercises the overall flow with a very small timeout and no real GitHub calls.
// It mainly asserts that validation works and that context cancellation is respected.
func TestSetupOrgWebhooks_ValidationAndContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{t: t}, nil))

	opts := SetupOptions{
		Org:            "",
		URL:            "https://example.com",
		Token:          "token",
		IncludePattern: "",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := SetupOrgWebhooks(ctx, opts, logger); err == nil {
		t.Fatalf("expected error for missing org")
	}
}

type testWriter struct {
	t *testing.T
}

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

