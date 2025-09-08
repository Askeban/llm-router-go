package main

import (
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/Askeban/llm-router-go/internal/config"
	h "github.com/Askeban/llm-router-go/internal/http"
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

	// Note: Analytics AI integration now handled by hybrid model service
	// Real-time data is fetched on-demand via the customer API

	r := gin.Default()
	
	// Register original routes for backward compatibility
	h.RegisterRoutes(r, cfg, db)
	
	// TODO: Register enhanced routes with the new recommendation engine
	// h.RegisterEnhancedRoutes(r, cfg, db)

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
