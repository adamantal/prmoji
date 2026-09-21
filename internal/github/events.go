package github

import (
	"strings"
)

type Action string

const (
	ActionCommented        Action = "commented"
	ActionApproved         Action = "approved"
	ActionChangesRequested Action = "changes_requested"
	ActionMerged           Action = "merged"
	ActionClosed           Action = "closed"
	// ActionReviewDismissed is emitted when a review is dismissed, either by a
	// user or by branch protection dismissing stale reviews. GitHub does not
	// report the dismissed review's original state, so this action clears both
	// the approved and the changes-requested reaction.
	ActionReviewDismissed Action = "review_dismissed"
)

// RemovesReaction reports whether the action should remove reactions instead of adding one.
func (a Action) RemovesReaction() bool {
	return a == ActionReviewDismissed
}

type Classification struct {
	Action    Action
	PRURL     string
	Commenter string
	Author    string
}

// IsCopilot reports whether the given GitHub login belongs to GitHub Copilot,
// e.g. "Copilot", "copilot-pull-request-reviewer[bot]" or "github-copilot[bot]".
func IsCopilot(login string) bool {
	return strings.Contains(strings.ToLower(login), "copilot")
}

func Classify(eventType string, body []byte) (Classification, bool) {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "issue_comment":
		return classifyIssueComment(body)
	case "pull_request_review":
		return classifyPRReview(body)
	case "pull_request":
		return classifyPullRequest(body)
	default:
		return Classification{}, false
	}
}
