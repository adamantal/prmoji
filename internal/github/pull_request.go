package github

import "encoding/json"

// prEventAction is the "action" field of a pull_request webhook.
type prEventAction string

const (
	prClosed   prEventAction = "closed"
	prReopened prEventAction = "reopened"
)

type pullRequestEvent struct {
	Action      prEventAction `json:"action"`
	PullRequest struct {
		Merged  bool   `json:"merged"`
		HTMLURL string `json:"html_url"`
	} `json:"pull_request"`
}

func classifyPullRequest(body []byte) (Classification, bool) {
	var e pullRequestEvent
	if err := json.Unmarshal(body, &e); err != nil {
		return Classification{}, false
	}
	if e.PullRequest.HTMLURL == "" {
		return Classification{}, false
	}
	switch e.Action {
	case prClosed:
		if e.PullRequest.Merged {
			return Classification{Action: ActionMerged, PRURL: e.PullRequest.HTMLURL}, true
		}
		return Classification{Action: ActionClosed, PRURL: e.PullRequest.HTMLURL}, true
	case prReopened:
		return Classification{Action: ActionReopened, PRURL: e.PullRequest.HTMLURL}, true
	default:
		return Classification{}, false
	}
}
