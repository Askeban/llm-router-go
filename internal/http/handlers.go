package http

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Askeban/llm-router-go/internal/auth"
	"github.com/Askeban/llm-router-go/internal/classifier"
	"github.com/Askeban/llm-router-go/internal/config"
	"github.com/Askeban/llm-router-go/internal/metrics"
	"github.com/Askeban/llm-router-go/internal/models"
	"github.com/Askeban/llm-router-go/internal/providers"
	"github.com/Askeban/llm-router-go/internal/recommendation"
)

type RouteRequest struct {
	Prompt      string         `json:"prompt"`
	Mode        string         `json:"mode"` // "recommend" | "generate"
	Constraints map[string]any `json:"constraints"`
}

type IngestPayload struct {
	Source  string                     `json:"source"`
	Metrics []metrics.NormalizedMetric `json:"metrics"`
}

func RegisterRoutes(r *gin.Engine, cfg *config.Config, db *sql.DB) {
	store := metrics.NewStore(db)
	
	// Initialize hybrid model service for real-time Analytics AI data
	hybridModelService := models.NewHybridModelService(db, cfg.ModelProfilesPath)
	
	clf := classifier.New(cfg.ClassifierURL)
	reg := providers.NewRegistry(cfg)
	
	// Initialize auth services
	authService := auth.NewService(db)
	jwtManager := auth.NewJWTManager()
	authHandlers := auth.NewHandlers(authService, jwtManager)
	
	// Initialize rate limiter (Redis connection)
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rateLimiter := auth.NewRateLimiter(redisAddr, "", 0)
	
	// Register authentication routes with hybrid model service
	registerAuthRoutes(r, authHandlers, rateLimiter, authService, hybridModelService, clf, store)
	
	// Register metrics lookup endpoint
	registerMetricsLookup(r, store)

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	
	// Hybrid model service endpoints
	r.GET("/models/metrics", func(c *gin.Context) {
		metrics := hybridModelService.GetMetrics()
		c.JSON(200, gin.H{
			"status": "ok",
			"metrics": metrics,
		})
	})
	
	r.POST("/models/refresh", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		
		if err := hybridModelService.RefreshCache(ctx); err != nil {
			c.JSON(500, gin.H{
				"error": gin.H{
					"code": "cache_refresh_failed",
					"message": err.Error(),
				},
			})
			return
		}
		
		c.JSON(200, gin.H{
			"status": "ok", 
			"message": "Model cache refreshed successfully",
			"metrics": hybridModelService.GetMetrics(),
		})
	})

	// Manual ingest (OpenLLM/LMArena dumps)
	r.POST("/ingest", func(c *gin.Context) {
		var p IngestPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if err := store.UpsertMetrics(c.Request.Context(), p.Source, p.Metrics); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true, "count": len(p.Metrics)})
	})


	r.GET("/metrics/models", func(c *gin.Context) {
		rows, err := store.ListModels(c.Request.Context())
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, rows)
	})

	r.GET("/metrics/:model", func(c *gin.Context) {
		model := strings.TrimSpace(c.Param("model"))
		if model == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing model"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if rows, _ := store.GetMetricsByModelLoose(ctx, model); len(rows) > 0 {
			c.JSON(http.StatusOK, rows)
			return
		}
		c.JSON(http.StatusNotFound, nil)
	})

	r.POST("/route", func(c *gin.Context) {
		var rr RouteRequest
		if err := c.ShouldBindJSON(&rr); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if strings.TrimSpace(rr.Prompt) == "" {
			c.JSON(400, gin.H{"error": "prompt required"})
			return
		}

		start := time.Now()
		category, difficulty, err := clf.Classify(c.Request.Context(), rr.Prompt)
		if err != nil {
			c.JSON(500, gin.H{"error": "classification failed"})
			return
		}
		classifyMs := time.Since(start).Milliseconds()

		mods, err := hybridModelService.GetModels(context.Background())
		if err != nil {
			c.JSON(500, gin.H{"error": "models not loaded"})
			return
		}

		rows, _ := store.GetAll(c.Request.Context())
		perModel := map[string]map[string]float64{}
		minBy, maxBy := map[string]float64{}, map[string]float64{}

		for _, r2 := range rows {
			k := strings.ToLower(r2.Metric)
			if _, ok := minBy[k]; !ok {
				minBy[k] = r2.Value
				maxBy[k] = r2.Value
			}
			if r2.Value < minBy[k] {
				minBy[k] = r2.Value
			}
			if r2.Value > maxBy[k] {
				maxBy[k] = r2.Value
			}
		}
		norm := func(metric string, v float64) float64 {
			min, ok1 := minBy[metric]
			max, ok2 := maxBy[metric]
			if !ok1 || !ok2 || max <= min {
				return 0
			}
			nv := (v - min) / (max - min)
			if nv < 0 {
				nv = 0
			}
			if nv > 1 {
				nv = 1
			}
			return nv
		}

		type mw struct {
			task, diff string
			weight     float64
		}
		metricMap := map[string]mw{
			"livecodebench":                          {"code", "", 1.0},
			"mmlu_pro":                               {"reasoning", "", 0.8},
			"gpqa":                                   {"reasoning", "", 0.6},
			"artificial_analysis_intelligence_index": {"reasoning", "", 1.0},
			"artificial_analysis_coding_index":       {"code", "", 1.0},
			"artificial_analysis_math_index":         {"math", "", 1.0},
		}

		for _, r2 := range rows {
			m := strings.ToLower(r2.Metric)
			if mm, ok := metricMap[m]; ok {
				if _, ok := perModel[r2.ModelID]; !ok {
					perModel[r2.ModelID] = map[string]float64{}
				}
				perModel[r2.ModelID][m] = norm(m, r2.Value) * mm.weight
			}
		}
		boost := map[string]float64{}
		for id, mm := range perModel {
			sum, n := 0.0, 0.0
			for _, v := range mm {
				sum += v
				n += 1
			}
			if n > 0 {
				boost[id] = sum / n
			} else {
				boost[id] = 0
			}
		}

		best, ranking := recommendation.Rank(category, difficulty, mods, boost)
		
		if len(ranking) == 0 {
			c.JSON(500, gin.H{"error": "no models available for recommendation"})
			return
		}

		if rr.Mode == "recommend" {
			c.JSON(200, gin.H{
				"classification":    gin.H{"category": category, "difficulty": difficulty, "ms": classifyMs},
				"recommended_model": best,
				"ranking":           ranking,
			})
			return
		}

		client, err := reg.ClientFor(best.ID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		out, usage, err := client.Generate(c.Request.Context(), rr.Prompt, rr.Constraints)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error(), "recommended_model": best})
			return
		}
		c.JSON(200, gin.H{
			"classification": gin.H{"category": category, "difficulty": difficulty, "ms": classifyMs},
			"model_used":     best, "usage": usage, "output": out,
		})
	})
	// hot-reload classifier rules from the configured path (or ?path=...)
	r.POST("/admin/reload-classifier", func(c *gin.Context) {
		path := c.Query("path")
		if strings.TrimSpace(path) == "" {
			path = os.Getenv("CLASSIFIER_RULES_PATH")
			if strings.TrimSpace(path) == "" {
				path = "./configs/classifier_rules.json"
			}
		}
		if err := clf.ReloadFrom(path); err != nil {
			c.JSON(500, gin.H{"ok": false, "error": err.Error(), "path": path})
			return
		}
		c.JSON(200, gin.H{"ok": true, "path": path})
	})

	// dry-run a classification and see raw scores per category
	type explainReq struct {
		Prompt string `json:"prompt"`
	}
	r.POST("/admin/classifier/explain", func(c *gin.Context) {
		var x explainReq
		if err := c.ShouldBindJSON(&x); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		scores := clf.Explain(x.Prompt)
		cat, diff, _ := clf.Classify(c.Request.Context(), x.Prompt)
		c.JSON(200, gin.H{"scores": scores, "category": cat, "difficulty": diff})
	})

}

// registerAuthRoutes sets up authentication and customer API routes
func registerAuthRoutes(r *gin.Engine, h *auth.Handlers, rateLimiter *auth.RateLimiter, authService *auth.Service, hybridService *models.HybridModelService, clf *classifier.Client, store *metrics.Store) {
	// Authentication routes (no auth required)
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", h.Register)
		authGroup.POST("/login", h.Login)
		authGroup.POST("/refresh", h.RefreshToken)
	}

	// Dashboard routes (JWT auth required)
	dashboardGroup := r.Group("/dashboard")
	dashboardGroup.Use(h.AuthMiddleware())
	{
		dashboardGroup.GET("/profile", h.GetProfile)
		dashboardGroup.GET("/api-keys", h.ListAPIKeys)
		dashboardGroup.POST("/api-keys", h.CreateAPIKey)
		dashboardGroup.GET("/usage", h.GetUsage)
	}

	// Customer API routes (API key auth required + rate limiting)
	v1Group := r.Group("/v1")
	v1Group.Use(h.APIKeyMiddleware())
	v1Group.Use(rateLimiter.RateLimitMiddleware(authService))
	{
		v1Group.GET("/models", handleListModels(hybridService))
		v1Group.POST("/recommend", handleRecommend(hybridService, clf, store))
	}
}

// handleListModels returns all available models
func handleListModels(hybridService *models.HybridModelService) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second) // Longer timeout for Analytics AI
		defer cancel()
		
		models, err := hybridService.GetModels(ctx)
		if err != nil {
			c.JSON(500, gin.H{
				"error": gin.H{
					"code":    "models_fetch_failed", 
					"message": "Failed to fetch models",
				},
			})
			return
		}
		
		c.JSON(200, gin.H{
			"models": models,
			"total":  len(models),
		})
	}
}

// handleRecommend provides intelligent model recommendations based on prompts
func handleRecommend(hybridService *models.HybridModelService, clf *classifier.Client, store *metrics.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Prompt     string         `json:"prompt" binding:"required"`
			MaxCost    float64        `json:"max_cost,omitempty"`    // Max cost per 1K tokens
			MaxLatency int            `json:"max_latency,omitempty"` // Max latency in ms
			Preferences map[string]float64 `json:"preferences,omitempty"` // performance, cost, latency weights
		}
		
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"error": gin.H{
					"code":    "invalid_request",
					"message": err.Error(),
				},
			})
			return
		}
		
		start := time.Now()
		
		// Step 1: Classify the prompt
		category, difficulty, err := clf.Classify(c.Request.Context(), req.Prompt)
		if err != nil {
			c.JSON(500, gin.H{
				"error": gin.H{
					"code":    "classification_failed",
					"message": "Failed to classify prompt",
				},
			})
			return
		}
		classifyMs := time.Since(start).Milliseconds()
		
		// Step 2: Get all models
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		
		allModels, err := hybridService.GetModels(ctx)
		if err != nil {
			c.JSON(500, gin.H{
				"error": gin.H{
					"code":    "models_fetch_failed",
					"message": "Failed to fetch models", 
				},
			})
			return
		}
		
		// Step 3: Apply filters if specified
		filteredModels := allModels
		if req.MaxCost > 0 || req.MaxLatency > 0 {
			var filtered []models.ModelProfile
			for _, m := range allModels {
				if req.MaxCost > 0 && m.CostInPer1K > req.MaxCost {
					continue
				}
				if req.MaxLatency > 0 && m.AvgLatencyMs > req.MaxLatency {
					continue
				}
				filtered = append(filtered, m)
			}
			filteredModels = filtered
		}
		
		if len(filteredModels) == 0 {
			c.JSON(200, gin.H{
				"error": gin.H{
					"code":    "no_models_available",
					"message": "No models match the specified constraints",
				},
				"classification": gin.H{
					"category":   category,
					"difficulty": difficulty,
				},
			})
			return
		}
		
		// Step 4: Get performance boosts from metrics
		rows, _ := store.GetAll(c.Request.Context())
		perfBoost := map[string]float64{}
		if len(rows) > 0 {
			perModel := map[string]map[string]float64{}
			minBy, maxBy := map[string]float64{}, map[string]float64{}
			
			// Collect metric ranges
			for _, r2 := range rows {
				k := strings.ToLower(r2.Metric)
				if _, ok := minBy[k]; !ok {
					minBy[k] = r2.Value
					maxBy[k] = r2.Value
				}
				if r2.Value < minBy[k] {
					minBy[k] = r2.Value
				}
				if r2.Value > maxBy[k] {
					maxBy[k] = r2.Value
				}
			}
			
			// Normalize metrics
			norm := func(metric string, v float64) float64 {
				min, ok1 := minBy[metric]
				max, ok2 := maxBy[metric]
				if !ok1 || !ok2 || max <= min {
					return 0
				}
				nv := (v - min) / (max - min)
				if nv < 0 {
					nv = 0
				}
				if nv > 1 {
					nv = 1
				}
				return nv
			}
			
			// Map metrics to categories
			metricMap := map[string]struct {
				category string
				weight   float64
			}{
				"livecodebench":                          {"coding", 1.0},
				"mmlu_pro":                               {"reasoning", 0.8},
				"gpqa":                                   {"reasoning", 0.6},
				"artificial_analysis_intelligence_index": {"reasoning", 1.0},
				"artificial_analysis_coding_index":       {"coding", 1.0},
				"artificial_analysis_math_index":         {"math", 1.0},
			}
			
			// Build per-model performance scores
			for _, r2 := range rows {
				m := strings.ToLower(r2.Metric)
				if mm, ok := metricMap[m]; ok && strings.ToLower(mm.category) == strings.ToLower(category) {
					if _, ok := perModel[r2.ModelID]; !ok {
						perModel[r2.ModelID] = map[string]float64{}
					}
					perModel[r2.ModelID][m] = norm(m, r2.Value) * mm.weight
				}
			}
			
			// Calculate boost scores
			for id, mm := range perModel {
				sum, n := 0.0, 0.0
				for _, v := range mm {
					sum += v
					n++
				}
				if n > 0 {
					perfBoost[id] = sum / n
				}
			}
		}
		
		// Step 5: Rank models using the existing recommendation engine
		bestModel, rankedModels := recommendation.Rank(category, difficulty, filteredModels, perfBoost)
		
		if bestModel.ID == "" {
			c.JSON(500, gin.H{
				"error": gin.H{
					"code":    "recommendation_failed",
					"message": "Failed to generate recommendation",
				},
			})
			return
		}
		
		totalMs := time.Since(start).Milliseconds()
		
		// Step 6: Return recommendation
		c.JSON(200, gin.H{
			"recommendation": gin.H{
				"model": bestModel,
				"score": rankedModels[0].Score,
				"reasoning": rankedModels[0].Why,
			},
			"classification": gin.H{
				"category":   category,
				"difficulty": difficulty,
			},
			"alternatives": func() []gin.H {
				var alt []gin.H
				for i, r := range rankedModels {
					if i >= 3 { // Top 3 alternatives
						break
					}
					alt = append(alt, gin.H{
						"model": r.ModelProfile,
						"score": r.Score,
						"reasoning": r.Why,
					})
				}
				return alt
			}(),
			"timing": gin.H{
				"total_ms":       totalMs,
				"classify_ms":    classifyMs,
				"recommendation_ms": totalMs - classifyMs,
			},
			"request_id": fmt.Sprintf("req_%d", time.Now().Unix()),
		})
	}
}
