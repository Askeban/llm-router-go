package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"llm-router-go/internal/api"
	"llm-router-go/internal/http/handlers"
	"llm-router-go/internal/policy"
)

func TestRecommendTop_OK(t *testing.T) {
	cat := policy.Catalog{Models: []policy.Model{{Name: "gpt-4o-mini"}, {Name: "gpt-4o"}, {Name: "claude-3.5-sonnet"}, {Name: "gemini-1.5-pro"}}}
	h := handlers.RecommendTop(handlers.RecommendDeps{Version: "test", Catalog: cat})
	srv := httptest.NewServer(h)
	defer srv.Close()

	body := []byte(`{"prompt":"explain","context":{"language":"go","selection_bytes":1024}}`)
	req, _ := http.NewRequest("POST", srv.URL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var resp api.RecommendTopResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Models) != 3 {
		t.Fatalf("expected 3 models, got %d", len(resp.Models))
	}
	if resp.Models[0].Name != "gpt-4o-mini" {
		t.Fatalf("unexpected first model: %s", resp.Models[0].Name)
	}
}
