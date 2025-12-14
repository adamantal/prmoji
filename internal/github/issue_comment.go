package github

import "encoding/json"

type issueCommentEvent struct {
	Action string `json:"action"`
	Issue  struct {
		PullRequest struct {
			HTMLURL string `json:"html_url"`
		} `json:"pull_request"`
	} `json:"issue"`
	Comment struct {
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"comment"`
}

func classifyIssueComment(body []byte) (Classification, bool) {
	var e issueCommentEvent
	if err := json.Unmarshal(body, &e); err != nil {
		return Classification{}, false
	}
	if e.Action != "created" {
		return Classification{}, false
	}
	if e.Issue.PullRequest.HTMLURL == "" {
		return Classification{}, false
	}
	return Classification{
		Action:    ActionCommented,
		PRURL:     e.Issue.PullRequest.HTMLURL,
		Commenter: e.Comment.User.Login,
	}, true
}
