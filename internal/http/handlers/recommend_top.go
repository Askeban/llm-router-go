package handlers

import (
	"encoding/json"
	"net/http"

	"llm-router-go/internal/api"
	"llm-router-go/internal/policy"
)

// RecommendTop returns up to the top three models for the given request.
func RecommendTop(d RecommendDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		models := policy.TopModels(d.Catalog, &req, 3)
		infos := make([]api.ModelInfo, len(models))
		for i, m := range models {
			infos[i] = api.ModelInfo{
				Name:           m.Name,
				Strengths:      m.Strengths,
				MaxInputTokens: m.MaxInputTokens,
				CostTier:       m.CostTier,
				LatencyTier:    m.LatencyTier,
				Languages:      m.Languages,
			}
		}
		res := api.RecommendTopResponse{Models: infos, Version: d.Version}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}
}
