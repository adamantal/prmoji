package util

import "github.com/adamantal/prmoji/internal/github"

// EmojiFor picks the Slack emoji for a classified GitHub event, taking the
// commenter into account so that Copilot comments are distinguished from human
// ones.
func EmojiFor(c github.Classification) string {
	if c.Action == github.ActionCommented && github.IsCopilot(c.Commenter) {
		return "robot_face"
	}
	return EmojiForAction(c.Action)
}

func EmojiForAction(a github.Action) string {
	switch a {
	case github.ActionCommented:
		return "speech_balloon"
	case github.ActionApproved:
		return "white_check_mark"
	case github.ActionChangesRequested:
		return "no_entry"
	case github.ActionMerged:
		return "pr-merged"
	case github.ActionClosed:
		return "wastebasket"
	default:
		return "speech_balloon"
	}
}
