package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"llm-router-go/internal/http/handlers"
	"llm-router-go/internal/policy"
)

var version = "dev" // set via -ldflags at build

func main() {
	apiKey := os.Getenv("ROUTER_API_KEY")
	catalog, err := policy.LoadCatalog("config/models.yaml")
	if err != nil {
		log.Fatalf("load catalog: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer, middleware.Timeout(30*time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK); w.Write([]byte("ok")) })

	deps := handlers.RecommendDeps{Version: version, Catalog: catalog, APIKey: apiKey}
	r.Post("/v1/recommend", handlers.Recommend(deps))

	addr := ":8080"
	log.Printf("router listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
