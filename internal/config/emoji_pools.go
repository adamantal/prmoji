package config

import (
	"fmt"
	"strings"

	"github.com/adamantal/prmoji/internal/util"
	"github.com/spf13/viper"
)

func loadEmojiPools(v *viper.Viper) (util.EmojiPools, error) {
	defaults := util.DefaultEmojiPools()
	pools := defaults

	var err error
	if pools.Commented, err = parseEmojiPoolEnv("EMOJI_POOL_COMMENTED", v.GetString("EMOJI_POOL_COMMENTED"), defaults.Commented); err != nil {
		return util.EmojiPools{}, err
	}
	if pools.Approved, err = parseEmojiPoolEnv("EMOJI_POOL_APPROVED", v.GetString("EMOJI_POOL_APPROVED"), defaults.Approved); err != nil {
		return util.EmojiPools{}, err
	}
	if pools.ChangesRequested, err = parseEmojiPoolEnv("EMOJI_POOL_CHANGES_REQUESTED", v.GetString("EMOJI_POOL_CHANGES_REQUESTED"), defaults.ChangesRequested); err != nil {
		return util.EmojiPools{}, err
	}
	if pools.Merged, err = parseEmojiPoolEnv("EMOJI_POOL_MERGED", v.GetString("EMOJI_POOL_MERGED"), defaults.Merged); err != nil {
		return util.EmojiPools{}, err
	}
	if pools.Closed, err = parseEmojiPoolEnv("EMOJI_POOL_CLOSED", v.GetString("EMOJI_POOL_CLOSED"), defaults.Closed); err != nil {
		return util.EmojiPools{}, err
	}
	return pools, nil
}

// parseEmojiPoolEnv parses a comma-separated list of exactly MaxPRsPerMessage Slack emoji names.
// An empty value returns fallback unchanged.
func parseEmojiPoolEnv(name, value string, fallback [util.MaxPRsPerMessage]string) ([util.MaxPRsPerMessage]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}

	parts := strings.Split(value, ",")
	out := [util.MaxPRsPerMessage]string{}
	if len(parts) != util.MaxPRsPerMessage {
		return out, fmt.Errorf("%s: expected %d comma-separated emoji names, got %d", name, util.MaxPRsPerMessage, len(parts))
	}
	for i, part := range parts {
		emoji := strings.TrimSpace(part)
		if emoji == "" {
			return out, fmt.Errorf("%s: emoji at position %d is empty", name, i+1)
		}
		out[i] = emoji
	}
	return out, nil
}
