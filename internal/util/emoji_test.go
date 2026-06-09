package util

import (
	"testing"

	"github.com/adamantal/prmoji/internal/github"
)

func TestEmojiFor_CopilotComment(t *testing.T) {
	cases := []struct {
		name      string
		commenter string
		want      string
	}{
		{"human comment", "bob", "speech_balloon"},
		{"copilot display name", "Copilot", "robot_face"},
		{"copilot reviewer bot", "copilot-pull-request-reviewer[bot]", "robot_face"},
		{"github copilot bot", "github-copilot[bot]", "robot_face"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EmojiFor(github.Classification{Action: github.ActionCommented, Commenter: tc.commenter})
			if got != tc.want {
				t.Fatalf("expected %q got %q", tc.want, got)
			}
		})
	}
}

func TestEmojiFor_CopilotOnlyAffectsComments(t *testing.T) {
	// A Copilot approval should still use the approval emoji, not the copilot one.
	got := EmojiFor(github.Classification{Action: github.ActionApproved, Commenter: "Copilot"})
	if got != "white_check_mark" {
		t.Fatalf("expected white_check_mark got %q", got)
	}
}
