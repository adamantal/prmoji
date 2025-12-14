package slack

import (
	"encoding/json"
	"regexp"
)

type EventEnvelope struct {
	Challenge string     `json:"challenge"`
	Event     SlackEvent `json:"event"`
}

type SlackEvent struct {
	Text    string `json:"text"`
	Channel string `json:"channel"`
	EventTS string `json:"event_ts"`
}

var prURLRe = regexp.MustCompile(`https://github\.com/[^/\s]+/[^/\s]+/pull/\d+`)

func ParseEnvelope(body []byte) (EventEnvelope, error) {
	var env EventEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return EventEnvelope{}, err
	}
	return env, nil
}

func ExtractPRURLs(text string) []string {
	if text == "" {
		return nil
	}
	matches := prURLRe.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}
	// Slack messages sometimes repeat the same URL (unfurls/quotes); de-dupe within one message.
	seen := make(map[string]struct{}, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, m)
	}
	return out
}
