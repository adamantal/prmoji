package util

import (
	"log/slog"

	"github.com/adamantal/prmoji/internal/github"
)

// MaxPRsPerMessage is the maximum number of PR URLs tracked per Slack message (slots 0..8).
const MaxPRsPerMessage = 9

// SlotDisplayNumber returns the 1-based PR index shown on numbered custom emojis (slot 0 → 1).
func SlotDisplayNumber(slotIndex int) int {
	return slotIndex + 1
}

// EmojiPools holds nine Slack emoji names per action (index = slot_index in a message).
type EmojiPools struct {
	Commented        [MaxPRsPerMessage]string
	Approved         [MaxPRsPerMessage]string
	ChangesRequested [MaxPRsPerMessage]string
	Merged           [MaxPRsPerMessage]string
	Closed           [MaxPRsPerMessage]string
}

// DefaultEmojiPools returns the built-in emoji pools (slot 0 matches single-PR behavior).
func DefaultEmojiPools() EmojiPools {
	return EmojiPools{
		Commented: [MaxPRsPerMessage]string{
			"speech_balloon",
			"left_speech_bubble",
			"thought_balloon",
			"envelope",
			"incoming_envelope",
			"email",
			"memo",
			"pencil2",
			"writing_hand",
		},
		Approved: [MaxPRsPerMessage]string{
			"white_check_mark",
			"heavy_check_mark",
			"ballot_box_with_check",
			"thumbsup",
			"ok_hand",
			"raised_hands",
			"muscle",
			"star",
			"sparkles",
		},
		ChangesRequested: [MaxPRsPerMessage]string{
			"no_entry",
			"x",
			"negative_squared_cross_mark",
			"warning",
			"octagonal_sign",
			"stop_sign",
			"imp",
			"rage",
			"face_with_symbols_on_mouth",
		},
		Merged: [MaxPRsPerMessage]string{
			"pr-merged",
			"tada",
			"rocket",
			"party_popper",
			"sparkles",
			"star",
			"champagne",
			"beers",
			"confetti_ball",
		},
		Closed: [MaxPRsPerMessage]string{
			"wastebasket",
			"file_cabinet",
			"door",
			"lock",
			"mailbox_with_no_mail",
			"put_litter_in_its_place",
			"hole",
			"black_large_square",
			"eject",
		},
	}
}

// EmojiFor picks the Slack emoji for a classified GitHub event and slot, taking the
// commenter into account so that Copilot comments are distinguished from human ones.
func (p EmojiPools) EmojiFor(c github.Classification, slotIndex int) string {
	if c.Action == github.ActionCommented && github.IsCopilot(c.Commenter) {
		return "robot_face"
	}
	return p.EmojiForAction(c.Action, slotIndex)
}

func (p EmojiPools) poolForAction(a github.Action) [MaxPRsPerMessage]string {
	switch a {
	case github.ActionCommented:
		return p.Commented
	case github.ActionApproved:
		return p.Approved
	case github.ActionChangesRequested:
		return p.ChangesRequested
	case github.ActionMerged:
		return p.Merged
	case github.ActionClosed:
		return p.Closed
	default:
		return p.Commented
	}
}

// EmojiForAction returns the Slack reaction name for the given action and slot_index (0-based PR order).
func (p EmojiPools) EmojiForAction(a github.Action, slotIndex int) string {
	pool := p.poolForAction(a)
	if slotIndex < 0 || slotIndex >= len(pool) {
		slog.Warn("slot index out of range, using slot 0", "slot", slotIndex, "action", a)
		slotIndex = 0
	}
	return pool[slotIndex]
}
