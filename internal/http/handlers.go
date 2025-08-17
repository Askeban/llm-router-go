package http

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Askeban/llm-router-go/internal/classifier"
	"github.com/Askeban/llm-router-go/internal/config"
	"github.com/Askeban/llm-router-go/internal/ingesters"
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
	profiles := models.NewProfiles(db, nil)
	clf := classifier.New(cfg.ClassifierURL)
	reg := providers.NewRegistry(cfg)

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

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

	// Pull Analytics AI and fuse into DB
	r.POST("/ingest/analytics", func(c *gin.Context) {
		apiKey := os.Getenv("ANALYTICS_AI_KEY")
		if strings.TrimSpace(apiKey) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "set ANALYTICS_AI_KEY in environment"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		if err := ingesters.SyncAnalyticsAI(ctx, apiKey, db); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
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

		mods, err := profiles.ListModels(context.Background())
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
			min, max := minBy[metric], maxBy[metric]
			if max <= min {
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
