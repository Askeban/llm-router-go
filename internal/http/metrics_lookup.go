package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Askeban/llm-router-go/internal/metrics"
	"github.com/gin-gonic/gin"
)

func registerMetricsLookup(r *gin.Engine, store *metrics.Store) {
	// GET /metrics/lookup?slug=GPT-4o%20mini
	r.GET("/metrics/lookup", func(c *gin.Context) {
		slug := strings.TrimSpace(c.Query("slug"))
		if slug == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing slug"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// 1) exact/loose
		if rows, _ := store.GetMetricsByModelLoose(ctx, slug); len(rows) > 0 {
			c.JSON(http.StatusOK, rows)
			return
		}
		// 2) fuzzy fallback
		if rows, _ := store.GetMetricsByModelFuzzy(ctx, slug, 500); len(rows) > 0 {
			c.JSON(http.StatusOK, rows)
			return
		}
		c.JSON(http.StatusNotFound, nil)
	})

	// Note: The /metrics/:model route is already registered in handlers.go
	// This function only adds the /metrics/lookup and /metrics/models routes

	// Handy index: /metrics/models?source=helm_gcs
	r.GET("/metrics/models", func(c *gin.Context) {
		source := strings.TrimSpace(c.Query("source"))
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		list, err := store.ListModelsBySource(ctx, source, 2000)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, list)
	})
}
