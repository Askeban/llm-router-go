package server

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    "llm-router-go/internal/config"
)

// NewRouter constructs a chi Router with all API routes registered.  It
// requires a ConfigStore containing the loaded models.
func NewRouter(store *config.ConfigStore) http.Handler {
    r := chi.NewRouter()
    // Health check
    r.Get("/v1/health", HealthHandler)
    // Model registry
    r.Get("/v1/models", ModelsHandler(store))
    // Selection endpoint
    r.Post("/v1/select-model", SelectHandler(store))
    // Prometheus metrics
    r.Handle("/metrics", promhttp.Handler())
    return r
}