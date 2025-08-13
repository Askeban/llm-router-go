package server

import (
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"

	"llm-router-go/internal/config"
	"llm-router-go/internal/handlers"
	"llm-router-go/internal/metrics"
)

type Server struct{ h *handlers.Handler }

func New(cfg *config.ConfigStore) *Server {
	metrics.MustRegister()
	return &Server{h: handlers.New(cfg)}
}

func (s *Server) Router() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	allowed := os.Getenv("ALLOWED_ORIGINS")
	if allowed == "" {
		allowed = "*"
	}
	r.Use(CORS(allowed))

	r.Group(func(gr chi.Router) {
		gr.Use(httprate.LimitByIP(100, 60*time.Second))

		gr.Get("/v1/health", s.h.Health)
		gr.Get("/v1/models", s.h.Models)

		gr.Group(func(pr chi.Router) {
			pr.Use(APIKeyAuth)
			pr.Post("/v1/select-model", s.h.SelectModel)
			pr.Post("/v1/rank-models", s.h.RankModels)
		})
	})

	r.Handle("/metrics", s.h.Metrics())
	r.NotFound(func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	return r
}
