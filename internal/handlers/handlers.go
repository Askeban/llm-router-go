package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"llm-router-go/internal/config"
	"llm-router-go/internal/metrics"
	"llm-router-go/internal/selection"
)

type Handler struct{ cfg *config.ConfigStore }

func New(cfg *config.ConfigStore) *Handler { return &Handler{cfg: cfg} }

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) Models(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.cfg.GetModels())
}

func (h *Handler) Metrics() http.Handler { return promhttp.Handler() }

func (h *Handler) SelectModel(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() { metrics.SelectionLatency.Observe(float64(time.Since(start).Milliseconds())) }()

	var req selection.SelectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		metrics.RequestsTotal.WithLabelValues("select", "400").Inc()
		http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := selection.Selector(h.cfg, req)
	if err != nil {
		metrics.RequestsTotal.WithLabelValues("select", "422").Inc()
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	metrics.RequestsTotal.WithLabelValues("select", "200").Inc()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) RankModels(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() { metrics.SelectionLatency.Observe(float64(time.Since(start).Milliseconds())) }()

	var req selection.SelectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		metrics.RequestsTotal.WithLabelValues("rank", "400").Inc()
		http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := selection.Ranker(h.cfg, req)
	if err != nil {
		metrics.RequestsTotal.WithLabelValues("rank", "422").Inc()
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	metrics.RequestsTotal.WithLabelValues("rank", "200").Inc()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
