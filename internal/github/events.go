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
)

type Classification struct {
	Action    Action
	PRURL     string
	Commenter string
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
