package github

import (
	_ "embed"
	"testing"
)

//go:embed testdata/pull_request_review_approved.json
var fixturePullRequestReviewApproved []byte

//go:embed testdata/pull_request_merged.json
var fixturePullRequestMerged []byte

//go:embed testdata/issue_comment_created.json
var fixtureIssueCommentCreated []byte

func TestClassify_PullRequestReviewApproved(t *testing.T) {
	c, ok := Classify("pull_request_review", fixturePullRequestReviewApproved)
	if !ok {
		t.Fatalf("expected ok")
	}
	if c.Action != ActionApproved {
		t.Fatalf("expected %q got %q", ActionApproved, c.Action)
	}
	if c.PRURL != "https://github.com/o/r/pull/123" {
		t.Fatalf("unexpected url: %s", c.PRURL)
	}
	if c.Author != "carol" {
		t.Fatalf("expected author carol got %q", c.Author)
	}
}

func TestClassify_PullRequestMerged(t *testing.T) {
	c, ok := Classify("pull_request", fixturePullRequestMerged)
	if !ok {
		t.Fatalf("expected ok")
	}
	if c.Action != ActionMerged {
		t.Fatalf("expected %q got %q", ActionMerged, c.Action)
	}
}

func TestClassify_IssueCommentCreated(t *testing.T) {
	c, ok := Classify("issue_comment", fixtureIssueCommentCreated)
	if !ok {
		t.Fatalf("expected ok")
	}
	if c.Action != ActionCommented {
		t.Fatalf("expected %q got %q", ActionCommented, c.Action)
	}
	if c.Commenter != "bob" {
		t.Fatalf("expected commenter bob got %q", c.Commenter)
	}
	if c.Author != "alice" {
		t.Fatalf("expected author alice got %q", c.Author)
	}
}

func TestClassify_IssueCommentByAuthor(t *testing.T) {
	body := []byte(`{
		"action": "created",
		"issue": {"pull_request": {"html_url": "https://github.com/o/r/pull/1"}, "user": {"login": "alice"}},
		"comment": {"user": {"login": "alice"}}
	}`)
	c, ok := Classify("issue_comment", body)
	if !ok {
		t.Fatalf("expected ok")
	}
	if c.Action != ActionCommented {
		t.Fatalf("expected %q got %q", ActionCommented, c.Action)
	}
	if c.Commenter != "alice" || c.Author != "alice" {
		t.Fatalf("expected commenter and author alice, got commenter=%q author=%q", c.Commenter, c.Author)
	}
}
