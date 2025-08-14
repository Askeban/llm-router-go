package http

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/Askeban/llm-router-go/internal/classifier"
	"github.com/Askeban/llm-router-go/internal/config"
	"github.com/Askeban/llm-router-go/internal/metrics"
	"github.com/Askeban/llm-router-go/internal/models"
	"github.com/Askeban/llm-router-go/internal/providers"
	"github.com/Askeban/llm-router-go/internal/recommendation"
	"github.com/gin-gonic/gin"
)

type RouteRequest struct {
	Prompt      string         `json:"prompt"`
	Mode        string         `json:"mode"`
	Constraints map[string]any `json:"constraints"`
}
type IngestPayload struct {
	Source  string                     `json:"source"`
	Metrics []metrics.NormalizedMetric `json:"metrics"`
}

func RegisterRoutes(r *gin.Engine, cfg *config.Config, db *sql.DB) {
	store := metrics.NewStore(db)
	profiles := models.NewProfiles(db, cfg)
	clf := classifier.New(cfg.ClassifierURL)
	reg := providers.NewRegistry(cfg)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
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
	// Note: /metrics/models route is now handled in registerMetricsLookup
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
		start := time.Now()
		cat, diff, err := clf.Classify(c.Request.Context(), rr.Prompt)
		if err != nil {
			c.JSON(502, gin.H{"error": "classification failed"})
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
		for _, r := range rows {
			k := strings.ToLower(r.Metric)
			if _, ok := minBy[k]; !ok {
				minBy[k] = r.Value
				maxBy[k] = r.Value
			} else {
				if r.Value < minBy[k] {
					minBy[k] = r.Value
				}
				if r.Value > maxBy[k] {
					maxBy[k] = r.Value
				}
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
		metricMap := map[string]struct {
			task   string
			diff   string
			weight float64
		}{"mmlu": {"reasoning", "medium", 1.0}, "bbh": {"reasoning", "hard", 1.2}, "gsm8k": {"math", "medium", 1.0}, "math": {"math", "hard", 1.2}, "human_eval": {"coding", "medium", 1.2}, "humaneval": {"coding", "medium", 1.2}, "livecodebench": {"coding", "hard", 1.2}, "scicode": {"coding", "hard", 1.0}, "ifeval": {"writing", "easy", 0.7}, "arena_elo": {"general", "", 0.6}, "artificial_analysis_coding_index": {"coding", "", 1.0}, "artificial_analysis_math_index": {"math", "", 1.0}, "artificial_analysis_intelligence_index": {"reasoning", "", 1.0}}
		for _, r := range rows {
			m := strings.ToLower(r.Metric)
			if strings.Contains(m, "coding") && strings.Contains(m, "complex") {
				metricMap[m] = struct {
					task   string
					diff   string
					weight float64
				}{"coding", "hard", 1.2}
			}
			if strings.Contains(m, "coding") && strings.Contains(m, "easy") {
				metricMap[m] = struct {
					task   string
					diff   string
					weight float64
				}{"coding", "easy", 0.8}
			}
			if mm, ok := metricMap[m]; ok {
				if mm.task == strings.ToLower(cat) || (mm.task == "general") {
					if _, ok := perModel[r.ModelID]; !ok {
						perModel[r.ModelID] = map[string]float64{}
					}
					perModel[r.ModelID][m] = norm(m, r.Value) * mm.weight
				}
			}
		}
		boost := map[string]float64{}
		for id, mm := range perModel {
			sum := 0.0
			n := 0.0
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
		best, ranking := recommendation.Rank(cat, diff, mods, boost)
		if rr.Mode == "recommend" {
			c.JSON(200, gin.H{"classification": gin.H{"category": cat, "difficulty": diff, "latency_ms": classifyMs}, "recommended_model": best, "ranking": ranking})
			return
		}
		client, err := reg.ClientFor(best.ID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		out, usage, err := client.Generate(c.Request.Context(), rr.Prompt, nil)
		if err != nil {
			c.JSON(502, gin.H{"error": err.Error(), "recommended_model": best})
			return
		}
		c.JSON(200, gin.H{"classification": gin.H{"category": cat, "difficulty": diff, "latency_ms": classifyMs}, "model_used": best, "usage": usage, "output": out})
	})
	mstore := metrics.NewStore(db)
	registerMetricsLookup(r, mstore)
}
