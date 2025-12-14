package github

import (
	"encoding/json"
	"strings"
)

type prReviewEvent struct {
	Action string `json:"action"`
	Review struct {
		State string `json:"state"`
		User  struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"review"`
	PullRequest struct {
		HTMLURL string `json:"html_url"`
	} `json:"pull_request"`
}

func classifyPRReview(body []byte) (Classification, bool) {
	var e prReviewEvent
	if err := json.Unmarshal(body, &e); err != nil {
		return Classification{}, false
	}
	if e.Action != "submitted" {
		return Classification{}, false
	}
	if e.PullRequest.HTMLURL == "" {
		return Classification{}, false
	}

	switch strings.ToLower(e.Review.State) {
	case "commented":
		return Classification{Action: ActionCommented, PRURL: e.PullRequest.HTMLURL, Commenter: e.Review.User.Login}, true
	case "approved":
		return Classification{Action: ActionApproved, PRURL: e.PullRequest.HTMLURL, Commenter: e.Review.User.Login}, true
	case "changes_requested":
		return Classification{Action: ActionChangesRequested, PRURL: e.PullRequest.HTMLURL, Commenter: e.Review.User.Login}, true
	default:
		return Classification{}, false
	}
}
