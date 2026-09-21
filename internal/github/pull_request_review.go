package github

import (
	"encoding/json"
	"strings"
)

// reviewEventAction is the "action" field of a pull_request_review webhook.
type reviewEventAction string

const (
	reviewSubmitted reviewEventAction = "submitted"
	reviewDismissed reviewEventAction = "dismissed"
)

// reviewState is the "review.state" field of a pull_request_review webhook.
type reviewState string

const (
	reviewStateCommented        reviewState = "commented"
	reviewStateApproved         reviewState = "approved"
	reviewStateChangesRequested reviewState = "changes_requested"
)

type prReviewEvent struct {
	Action reviewEventAction `json:"action"`
	Review struct {
		State string `json:"state"`
		User  struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"review"`
	PullRequest struct {
		HTMLURL string `json:"html_url"`
		User    struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"pull_request"`
}

func classifyPRReview(body []byte) (Classification, bool) {
	var e prReviewEvent
	if err := json.Unmarshal(body, &e); err != nil {
		return Classification{}, false
	}
	if e.Action != reviewSubmitted && e.Action != reviewDismissed {
		return Classification{}, false
	}
	if e.PullRequest.HTMLURL == "" {
		return Classification{}, false
	}

	if e.Action == reviewDismissed {
		return Classification{Action: ActionReviewDismissed, PRURL: e.PullRequest.HTMLURL, Commenter: e.Review.User.Login, Author: e.PullRequest.User.Login}, true
	}

	switch reviewState(strings.ToLower(e.Review.State)) {
	case reviewStateCommented:
		return Classification{Action: ActionCommented, PRURL: e.PullRequest.HTMLURL, Commenter: e.Review.User.Login, Author: e.PullRequest.User.Login}, true
	case reviewStateApproved:
		return Classification{Action: ActionApproved, PRURL: e.PullRequest.HTMLURL, Commenter: e.Review.User.Login, Author: e.PullRequest.User.Login}, true
	case reviewStateChangesRequested:
		return Classification{Action: ActionChangesRequested, PRURL: e.PullRequest.HTMLURL, Commenter: e.Review.User.Login, Author: e.PullRequest.User.Login}, true
	default:
		return Classification{}, false
	}
}
