package slack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type capturedCall struct {
	path    string
	channel string
	ts      string
	name    string
}

// newTestClient returns a client pointed at a stub Slack API that answers with
// apiBody, plus a pointer to the last captured call.
func newTestClient(t *testing.T, apiBody string) (*Client, *[]capturedCall) {
	t.Helper()
	var calls []capturedCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		calls = append(calls, capturedCall{
			path:    r.URL.Path,
			channel: r.PostForm.Get("channel"),
			ts:      r.PostForm.Get("timestamp"),
			name:    r.PostForm.Get("name"),
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(apiBody))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("xoxb-test")
	c.baseURL = srv.URL
	return c, &calls
}

func TestRemoveReaction_CallsReactionsRemove(t *testing.T) {
	c, calls := newTestClient(t, `{"ok":true}`)

	if err := c.RemoveReaction(context.Background(), "C1", "1700000000.000100", "white_check_mark"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("expected 1 call got %d", len(*calls))
	}
	got := (*calls)[0]
	if got.path != "/reactions.remove" {
		t.Fatalf("expected /reactions.remove got %s", got.path)
	}
	if got.channel != "C1" || got.ts != "1700000000.000100" || got.name != "white_check_mark" {
		t.Fatalf("unexpected form: %+v", got)
	}
}

func TestRemoveReaction_NoReactionIsNotAnError(t *testing.T) {
	c, _ := newTestClient(t, `{"ok":false,"error":"no_reaction"}`)

	if err := c.RemoveReaction(context.Background(), "C1", "1.1", "white_check_mark"); err != nil {
		t.Fatalf("expected no_reaction to be benign, got %v", err)
	}
}

func TestRemoveReaction_MessageNotFoundIsNotAnError(t *testing.T) {
	c, _ := newTestClient(t, `{"ok":false,"error":"message_not_found"}`)

	if err := c.RemoveReaction(context.Background(), "C1", "1.1", "white_check_mark"); err != nil {
		t.Fatalf("expected message_not_found to be benign, got %v", err)
	}
}

func TestRemoveReaction_OtherAPIErrorFails(t *testing.T) {
	c, _ := newTestClient(t, `{"ok":false,"error":"invalid_auth"}`)

	if err := c.RemoveReaction(context.Background(), "C1", "1.1", "white_check_mark"); err == nil {
		t.Fatalf("expected invalid_auth to fail")
	}
}

func TestAddReaction_CallsReactionsAdd(t *testing.T) {
	c, calls := newTestClient(t, `{"ok":true}`)

	if err := c.AddReaction(context.Background(), "C1", "1.1", "speech_balloon"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if (*calls)[0].path != "/reactions.add" {
		t.Fatalf("expected /reactions.add got %s", (*calls)[0].path)
	}
}

func TestAddReaction_AlreadyReactedIsNotAnError(t *testing.T) {
	c, _ := newTestClient(t, `{"ok":false,"error":"already_reacted"}`)

	if err := c.AddReaction(context.Background(), "C1", "1.1", "speech_balloon"); err != nil {
		t.Fatalf("expected already_reacted to be benign, got %v", err)
	}
}

func TestAddReaction_NoReactionStillFails(t *testing.T) {
	c, _ := newTestClient(t, `{"ok":false,"error":"no_reaction"}`)

	if err := c.AddReaction(context.Background(), "C1", "1.1", "speech_balloon"); err == nil {
		t.Fatalf("expected no_reaction to fail on add")
	}
}
