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

	t.Run("extracts labelled links and normalizes http to https", func(t *testing.T) {
		text := "" +
			"- <http://github.com/a/b/pull/1|feat: first change>\n" +
			"- <https://github.com/c/d/pull/22|fix: second change>\n" +
			"- <http://github.com/e/f/pull/333|chore: third change>"

		urls := ExtractPRURLs(text)
		want := []string{
			"https://github.com/a/b/pull/1",
			"https://github.com/c/d/pull/22",
			"https://github.com/e/f/pull/333",
		}
		if len(urls) != len(want) {
			t.Fatalf("expected %d urls got %d: %#v", len(want), len(urls), urls)
		}
		for i := range want {
			if urls[i] != want[i] {
				t.Fatalf("unexpected url[%d]: %s", i, urls[i])
			}
		}
	})

	t.Run("dedupes http and https links to the same PR", func(t *testing.T) {
		text := "<http://github.com/a/b/pull/1|label> and https://github.com/a/b/pull/1"
		urls := ExtractPRURLs(text)
		if len(urls) != 1 {
			t.Fatalf("expected 1 url got %d: %#v", len(urls), urls)
		}
		if urls[0] != "https://github.com/a/b/pull/1" {
			t.Fatalf("unexpected url[0]: %s", urls[0])
		}
	})
}
