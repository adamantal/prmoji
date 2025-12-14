package slack

import "testing"

func TestExtractPRURLs(t *testing.T) {
	t.Run("extracts multiple PR urls", func(t *testing.T) {
		text := "please review https://github.com/a/b/pull/1 and https://github.com/c/d/pull/22"
		urls := ExtractPRURLs(text)
		if len(urls) != 2 {
			t.Fatalf("expected 2 urls got %d", len(urls))
		}
		if urls[0] != "https://github.com/a/b/pull/1" {
			t.Fatalf("unexpected url[0]: %s", urls[0])
		}
		if urls[1] != "https://github.com/c/d/pull/22" {
			t.Fatalf("unexpected url[1]: %s", urls[1])
		}
	})

	t.Run("dedupes duplicates and ignores non-PR urls", func(t *testing.T) {
		text := "" +
			"dup https://github.com/a/b/pull/1 " +
			"dup-again https://github.com/a/b/pull/1 " +
			"bad-path https://github.com/a/b/pulls/1 " +
			"issue https://github.com/a/b/issues/1 " +
			"not-github https://gitlab.com/a/b/pull/1 " +
			"enterprise https://github.example.com/a/b/pull/1 " +
			"querystring https://github.com/a/b/pull/1?foo=bar "

		urls := ExtractPRURLs(text)
		// Note: querystrings aren't matched by the PRD regex, but the regex used by
		// the extractor will still match the base PR URL prefix.
		if len(urls) != 1 {
			t.Fatalf("expected 1 url got %d: %#v", len(urls), urls)
		}
		if urls[0] != "https://github.com/a/b/pull/1" {
			t.Fatalf("unexpected url[0]: %s", urls[0])
		}
	})
}
