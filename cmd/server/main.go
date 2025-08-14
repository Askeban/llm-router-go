package main

import (
	"github.com/Askeban/llm-router-go/internal/config"
	httpapi "github.com/Askeban/llm-router-go/internal/http"
	"github.com/Askeban/llm-router-go/internal/models"
	"github.com/Askeban/llm-router-go/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"os"
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
	defer db.Close()
	if err := models.SeedFromJSON(db, cfg.ModelProfilesPath); err != nil {
		log.Printf("seed warn: %v", err)
	}
	r := gin.Default()
	httpapi.RegisterRoutes(r, cfg, db)
	addr := ":" + cfg.Port
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	log.Printf("llm-router-go v4 on %s", addr)
	log.Fatal(r.Run(addr))
}
