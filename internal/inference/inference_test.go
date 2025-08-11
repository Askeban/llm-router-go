package inference

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestSendPromptAnthropicSuccess(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/v1/messages" {
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"id":"msg1","type":"message","role":"assistant","model":"claude-test","content":[{"type":"text","text":"hi"}],"stop_reason":"end_turn","stop_sequence":null}`))
    }))
    defer ts.Close()

    t.Setenv("ANTHROPIC_API_KEY", "test")
    t.Setenv("ANTHROPIC_API_URL", ts.URL)

    resp, err := SendPrompt("claude-test", "hello")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp != "hi" {
        t.Fatalf("unexpected response: %s", resp)
    }
}

func TestSendPromptAnthropicError(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        http.Error(w, "bad request", http.StatusBadRequest)
    }))
    defer ts.Close()

    t.Setenv("ANTHROPIC_API_KEY", "test")
    t.Setenv("ANTHROPIC_API_URL", ts.URL)

    if _, err := SendPrompt("claude-test", "hello"); err == nil {
        t.Fatalf("expected error but got nil")
    }
}

