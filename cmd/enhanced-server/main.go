package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	httpHandlers "github.com/Askeban/llm-router-go/internal/http"
	"github.com/Askeban/llm-router-go/internal/services"
)

func main() {
	log.Println("[ENHANCED-SERVER] Starting Enhanced LLM Router Server v2.0")

	// Get model path from environment or use default
	modelPath := os.Getenv("MODEL_PATH")
	if modelPath == "" {
		modelPath = "./configs/model_1.json"
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	// Initialize the enhanced router service
	log.Printf("[ENHANCED-SERVER] Initializing router service with model path: %s", modelPath)
	routerService, err := services.NewEnhancedRouterService(modelPath)
	if err != nil {
		log.Fatalf("[ENHANCED-SERVER] Failed to initialize router service: %v", err)
	}

	// Log initial statistics
	stats := routerService.GetStats()
	log.Printf("[ENHANCED-SERVER] Service initialized successfully:")
	log.Printf("  - Total models: %v", stats["total_models"])
	log.Printf("  - Models by type: %v", stats["models_by_type"])
	log.Printf("  - Data sources: %v", stats["data_sources"])

	// Set up Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	
	// Add middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	// Set up enhanced handlers
	enhancedHandlers := httpHandlers.NewEnhancedHandlers(routerService)
	enhancedHandlers.SetupEnhancedRoutes(r)

	// Set up additional routes
	setupAdditionalRoutes(r, routerService)

	// Start server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Graceful shutdown setup
	go func() {
		log.Printf("[ENHANCED-SERVER] Server starting on port %s", port)
		log.Printf("[ENHANCED-SERVER] Enhanced API endpoints available at:")
		log.Printf("  - POST http://localhost:%s/api/v2/recommend/smart", port)
		log.Printf("  - POST http://localhost:%s/api/v2/recommend/direct", port) 
		log.Printf("  - POST http://localhost:%s/api/v2/classify", port)
		log.Printf("  - GET  http://localhost:%s/api/v2/models", port)
		log.Printf("  - GET  http://localhost:%s/api/v2/stats", port)
		log.Printf("  - GET  http://localhost:%s/api/v2/status", port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[ENHANCED-SERVER] Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[ENHANCED-SERVER] Shutting down server...")

	// Give outstanding requests 5 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[ENHANCED-SERVER] Server forced to shutdown: %v", err)
	}

	log.Println("[ENHANCED-SERVER] Server exited")
}

func corsMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}

func setupAdditionalRoutes(r *gin.Engine, routerService *services.EnhancedRouterService) {
	// Root endpoint
	r.GET("/", func(c *gin.Context) {
		stats := routerService.GetStats()
		c.JSON(http.StatusOK, gin.H{
			"service":     "Enhanced LLM Router",
			"version":     "2.0",
			"description": "AI Model Recommendation System with Smart Classification",
			"features": []string{
				"Smart prompt classification",
				"Multi-modal model support",
				"Analytics AI integration", 
				"Community intelligence",
				"Complexity-aware scoring",
				"Real-time data fusion",
			},
			"stats": stats,
			"endpoints": gin.H{
				"smart_recommendations":  "POST /api/v2/recommend/smart",
				"direct_recommendations": "POST /api/v2/recommend/direct",
				"prompt_classification":  "POST /api/v2/classify",
				"model_discovery":        "GET /api/v2/models",
				"service_stats":          "GET /api/v2/stats",
				"health_check":           "GET /api/v2/health",
			},
		})
	})

	// Legacy compatibility endpoint
	r.POST("/recommend", func(c *gin.Context) {
		var legacyReq struct {
			Category   string `json:"category"`
			Difficulty string `json:"difficulty"`
		}

		if err := c.ShouldBindJSON(&legacyReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request format",
			})
			return
		}

		// Convert legacy request to new format
		prompt := "I need help with " + legacyReq.Category + " task"
		if legacyReq.Difficulty != "" {
			prompt += " with " + legacyReq.Difficulty + " complexity"
		}

		smartReq := services.SmartRecommendationRequest{
			Prompt: prompt,
		}

		response := routerService.GetSmartRecommendations(smartReq)

		// Return in legacy format for backward compatibility
		c.JSON(http.StatusOK, gin.H{
			"top_model":     response.Recommendations.Recommendations[0].Model,
			"ranked_models": response.Recommendations.Recommendations,
			"classification": response.Classification,
		})
	})

	// Quick test endpoints for development
	r.GET("/test/text", func(c *gin.Context) {
		req := services.SmartRecommendationRequest{
			Prompt: "Write a Python function to calculate fibonacci numbers with optimizations",
		}
		response := routerService.GetSmartRecommendations(req)
		c.JSON(http.StatusOK, response)
	})

	r.GET("/test/image", func(c *gin.Context) {
		req := services.SmartRecommendationRequest{
			Prompt: "Generate a photorealistic image of a sunset over mountains",
		}
		response := routerService.GetSmartRecommendations(req)
		c.JSON(http.StatusOK, response)
	})

	r.GET("/test/video", func(c *gin.Context) {
		req := services.SmartRecommendationRequest{
			Prompt: "Create a 30-second marketing video with professional quality",
		}
		response := routerService.GetSmartRecommendations(req)
		c.JSON(http.StatusOK, response)
	})
}