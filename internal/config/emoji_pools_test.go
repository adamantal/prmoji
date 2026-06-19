package config

import (
	"strings"
	"testing"

	"github.com/adamantal/prmoji/internal/github"
	"github.com/adamantal/prmoji/internal/util"
	"github.com/spf13/viper"
)

func TestLoad_emojiPoolsFromEnv(t *testing.T) {
	t.Setenv("SLACK_TOKEN", "xoxb-test")
	t.Setenv("EMOJI_POOL_CHANGES_REQUESTED", strings.Join([]string{
		"a", "b", "c", "d", "e", "f", "g", "h", "i",
	}, ","))

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.EmojiPools.EmojiForAction(github.ActionChangesRequested, 0); got != "a" {
		t.Fatalf("slot 0: got %q want a", got)
	}
	if got := cfg.EmojiPools.EmojiForAction(github.ActionChangesRequested, 8); got != "i" {
		t.Fatalf("slot 8: got %q want i", got)
	}
	// Other pools stay at defaults.
	if got := cfg.EmojiPools.EmojiForAction(github.ActionApproved, 0); got != "white_check_mark" {
		t.Fatalf("approved default: got %q", got)
	}
}

func TestParseEmojiPoolEnv_invalidCount(t *testing.T) {
	fallback := util.DefaultEmojiPools().ChangesRequested
	_, err := parseEmojiPoolEnv("EMOJI_POOL_CHANGES_REQUESTED", "only,one", fallback)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_emojiPoolsInvalidFails(t *testing.T) {
	t.Setenv("SLACK_TOKEN", "xoxb-test")
	t.Setenv("EMOJI_POOL_APPROVED", "one,two")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadEmojiPools_viper(t *testing.T) {
	v := viper.New()
	v.Set("EMOJI_POOL_MERGED", "m1,m2,m3,m4,m5,m6,m7,m8,m9")
	pools, err := loadEmojiPools(v)
	if err != nil {
		t.Fatal(err)
	}
	if pools.Merged[0] != "m1" || pools.Merged[8] != "m9" {
		t.Fatalf("unexpected merged pool: %#v", pools.Merged)
	}
}
