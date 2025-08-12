package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"llm-router-go/internal/api"
	"llm-router-go/internal/policy"
)

type RecommendDeps struct {
	Version string
	Catalog policy.Catalog
	APIKey  string
}

func Recommend(d RecommendDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		if d.APIKey != "" && r.Header.Get("X-API-Key") != d.APIKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var req api.RecommendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := req.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		dec := policy.Score(d.Catalog, &req)
		res := api.RecommendResponse{
			RecommendedModel: dec.Top,
			Rationale:        dec.Rationale,
			Confidence:       dec.Confidence,
			Alternatives:     dec.Alternatives,
			CostMsEstimate:   int(time.Since(start).Milliseconds()),
			Flags:            dec.Flags,
			Version:          d.Version,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}
}
