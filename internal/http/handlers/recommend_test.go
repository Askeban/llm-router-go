package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"llm-router-go/internal/http/handlers"
	"llm-router-go/internal/policy"
)

func TestRecommend_OK(t *testing.T) {
	cat := policy.Catalog{Models: []policy.Model{{Name: "gpt-4o"}, {Name: "gpt-4o-mini"}}}
	h := handlers.Recommend(handlers.RecommendDeps{Version: "test", Catalog: cat})
	srv := httptest.NewServer(h)
	defer srv.Close()

	body := []byte(`{"prompt":"explain this","context":{"language":"go","selection_bytes":1024}}`)
	req, _ := http.NewRequest("POST", srv.URL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("status=%d", res.StatusCode)
	}
}

func TestRecommend_Auth(t *testing.T) {
	cat := policy.Catalog{Models: []policy.Model{{Name: "gpt-4o"}}}
	h := handlers.Recommend(handlers.RecommendDeps{Version: "test", Catalog: cat, APIKey: "dev"})
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL, bytes.NewReader([]byte(`{"prompt":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	if res, _ := http.DefaultClient.Do(req); res.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}
