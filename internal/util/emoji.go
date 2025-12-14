package util

import "github.com/adamantal/prmoji/internal/github"

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
