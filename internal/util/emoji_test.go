package util

import (
	"testing"

	"github.com/adamantal/prmoji/internal/github"
)

func TestDefaultEmojiPools_slotOrder(t *testing.T) {
	p := DefaultEmojiPools()
	if got := p.EmojiForAction(github.ActionApproved, 0); got != "white_check_mark" {
		t.Fatalf("slot 0: got %q want white_check_mark", got)
	}
	if got := p.EmojiForAction(github.ActionApproved, 2); got != "ballot_box_with_check" {
		t.Fatalf("slot 2: got %q want ballot_box_with_check", got)
	}
	if got := p.EmojiForAction(github.ActionChangesRequested, 2); got != "negative_squared_cross_mark" {
		t.Fatalf("slot 2 changes_requested: got %q want negative_squared_cross_mark", got)
	}
}

func TestDefaultEmojiPools_distinctPerSlot(t *testing.T) {
	p := DefaultEmojiPools()
	seen := make(map[string]struct{})
	for i := 0; i < MaxPRsPerMessage; i++ {
		name := p.EmojiForAction(github.ActionApproved, i)
		if _, ok := seen[name]; ok {
			t.Fatalf("duplicate approved emoji at slot %d: %q", i, name)
		}
		seen[name] = struct{}{}
	}
}

func TestEmojiPools_customOverrides(t *testing.T) {
	p := DefaultEmojiPools()
	p.ChangesRequested = [MaxPRsPerMessage]string{
		"prmoji-changes-1",
		"prmoji-changes-2",
		"prmoji-changes-3",
		"prmoji-changes-4",
		"prmoji-changes-5",
		"prmoji-changes-6",
		"prmoji-changes-7",
		"prmoji-changes-8",
		"prmoji-changes-9",
	}
	if got := p.EmojiForAction(github.ActionChangesRequested, 2); got != "prmoji-changes-3" {
		t.Fatalf("got %q want prmoji-changes-3", got)
	}
}

func TestDefaultEmojiPools_outOfRangeUsesSlot0(t *testing.T) {
	p := DefaultEmojiPools()
	if got := p.EmojiForAction(github.ActionApproved, 99); got != "white_check_mark" {
		t.Fatalf("out of range: got %q want white_check_mark", got)
	}
}

func TestSlotDisplayNumber(t *testing.T) {
	if got := SlotDisplayNumber(2); got != 3 {
		t.Fatalf("got %d want 3", got)
	}
}

func TestEmojiFor_CopilotComment(t *testing.T) {
	p := DefaultEmojiPools()
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
			got := p.EmojiFor(github.Classification{Action: github.ActionCommented, Commenter: tc.commenter}, 0)
			if got != tc.want {
				t.Fatalf("expected %q got %q", tc.want, got)
			}
		})
	}
}

func TestEmojiFor_CopilotOnlyAffectsComments(t *testing.T) {
	p := DefaultEmojiPools()
	got := p.EmojiFor(github.Classification{Action: github.ActionApproved, Commenter: "Copilot"}, 0)
	if got != "white_check_mark" {
		t.Fatalf("expected white_check_mark got %q", got)
	}
}
