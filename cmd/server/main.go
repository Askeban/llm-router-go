package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/Askeban/llm-router-go/internal/config"
	h "github.com/Askeban/llm-router-go/internal/http"
	"github.com/Askeban/llm-router-go/internal/ingesters"
	"github.com/Askeban/llm-router-go/internal/models"
	"github.com/Askeban/llm-router-go/internal/storage"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := storage.InitSQLite(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("sqlite: %v", err)
	}

	// Seed models table from JSON (idempotent)
	if err := models.SeedFromJSON(db, cfg.ModelProfilesPath); err != nil {
		log.Fatalf("seed models: %v", err)
	}

	// Optional: Pull Analytics AI at boot if the key is set
	if apiKey := os.Getenv("ANALYTICS_AI_KEY"); apiKey != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := ingesters.SyncAnalyticsAI(ctx, apiKey, db); err != nil {
			log.Printf("warn: analytics sync: %v", err)
		}
		cancel()
	}

	r := gin.Default()
	h.RegisterRoutes(r, cfg, db)

	addr := cfg.Port
	if addr == "" {
		addr = ":8080"
	}
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}
	log.Printf("llm-router-go v4 on %s", addr)
	log.Fatal(r.Run(addr))
}
