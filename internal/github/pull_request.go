package github

import "encoding/json"

type pullRequestEvent struct {
	Action      string `json:"action"`
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
	if e.Action != "closed" {
		return Classification{}, false
	}
	if e.PullRequest.HTMLURL == "" {
		return Classification{}, false
	}
	if e.PullRequest.Merged {
		return Classification{Action: ActionMerged, PRURL: e.PullRequest.HTMLURL}, true
	}
	return Classification{Action: ActionClosed, PRURL: e.PullRequest.HTMLURL}, true
}
