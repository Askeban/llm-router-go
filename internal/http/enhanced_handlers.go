package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/Askeban/llm-router-go/internal/recommendation"
	"github.com/Askeban/llm-router-go/internal/services"
)

// EnhancedHandlers provides HTTP handlers for the enhanced router service
type EnhancedHandlers struct {
	routerService *services.EnhancedRouterService
}

func NewEnhancedHandlers(routerService *services.EnhancedRouterService) *EnhancedHandlers {
	return &EnhancedHandlers{
		routerService: routerService,
	}
}

// SetupEnhancedRoutes sets up all the enhanced router endpoints
func (h *EnhancedHandlers) SetupEnhancedRoutes(r *gin.Engine) {
	// Enhanced recommendation endpoints
	api := r.Group("/api/v2")
	{
		// Smart recommendation - just send a prompt
		api.POST("/recommend/smart", h.getSmartRecommendations)
		
		// Direct recommendation - with explicit parameters
		api.POST("/recommend/direct", h.getDirectRecommendations)
		
		// Classification testing
		api.POST("/classify", h.classifyPrompt)
		
		// Model discovery and information
		api.GET("/models", h.getAllModels)
		api.GET("/models/:id", h.getModelById)
		api.GET("/models/type/:type", h.getModelsByType)
		
		// Service information
		api.GET("/stats", h.getServiceStats)
		api.POST("/refresh", h.refreshData)
		
		// Health and status
		api.GET("/health", h.healthCheck)
		api.GET("/status", h.getStatus)
	}
}

// getSmartRecommendations handles intelligent prompt-based recommendations
func (h *EnhancedHandlers) getSmartRecommendations(c *gin.Context) {
	var req services.SmartRecommendationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if req.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Prompt is required",
		})
		return
	}

	response := h.routerService.GetSmartRecommendations(req)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// getDirectRecommendations handles explicit recommendation requests
func (h *EnhancedHandlers) getDirectRecommendations(c *gin.Context) {
	var req recommendation.RecommendationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format", 
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if req.TaskType == "" {
		req.TaskType = "text" // default
	}
	if req.Category == "" {
		req.Category = "writing" // default
	}
	if req.Complexity == "" {
		req.Complexity = "medium" // default
	}
	if req.Priority == "" {
		req.Priority = "balanced" // default
	}

	response := h.routerService.GetDirectRecommendations(req)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// classifyPrompt handles prompt classification testing
func (h *EnhancedHandlers) classifyPrompt(c *gin.Context) {
	var req struct {
		Prompt string `json:"prompt" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	classification := h.routerService.TestClassification(req.Prompt)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    classification,
	})
}

// getAllModels returns all available models
func (h *EnhancedHandlers) getAllModels(c *gin.Context) {
	// Parse query parameters
	limit := 50 // default
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offset := 0 // default
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	models := h.routerService.GetAllModels()

	// Apply pagination
	total := len(models)
	start := offset
	end := offset + limit

	if start >= total {
		models = nil
	} else {
		if end > total {
			end = total
		}
		models = models[start:end]
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"models": models,
			"pagination": gin.H{
				"total":  total,
				"limit":  limit,
				"offset": offset,
				"count":  len(models),
			},
		},
	})
}

// getModelById returns a specific model by ID
func (h *EnhancedHandlers) getModelById(c *gin.Context) {
	modelId := c.Param("id")
	if modelId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Model ID is required",
		})
		return
	}

	model, found := h.routerService.GetModelByID(modelId)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Model not found",
			"id":    modelId,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    model,
	})
}

// getModelsByType returns models filtered by type
func (h *EnhancedHandlers) getModelsByType(c *gin.Context) {
	modelType := c.Param("type")
	validTypes := []string{"text", "image", "video", "audio", "multimodal"}
	
	isValidType := false
	for _, validType := range validTypes {
		if modelType == validType {
			isValidType = true
			break
		}
	}

	if !isValidType {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       "Invalid model type",
			"provided":    modelType,
			"valid_types": validTypes,
		})
		return
	}

	models := h.routerService.GetModelsByType(modelType)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"model_type": modelType,
			"models":     models,
			"count":      len(models),
		},
	})
}

// getServiceStats returns service statistics and metadata
func (h *EnhancedHandlers) getServiceStats(c *gin.Context) {
	stats := h.routerService.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// refreshData triggers a refresh of data sources
func (h *EnhancedHandlers) refreshData(c *gin.Context) {
	if err := h.routerService.RefreshData(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to refresh data",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Data refresh initiated successfully",
	})
}

// healthCheck provides a simple health check endpoint
func (h *EnhancedHandlers) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "enhanced-llm-router",
		"version": "2.0",
	})
}

// getStatus provides detailed service status information
func (h *EnhancedHandlers) getStatus(c *gin.Context) {
	stats := h.routerService.GetStats()

	status := gin.H{
		"service": "enhanced-llm-router",
		"version": "2.0",
		"status":  "running",
		"features": []string{
			"smart-classification",
			"multi-modal-support",
			"analytics-ai-integration",
			"community-intelligence",
			"complexity-scoring",
			"data-fusion",
		},
		"endpoints": []string{
			"POST /api/v2/recommend/smart",
			"POST /api/v2/recommend/direct",
			"POST /api/v2/classify",
			"GET /api/v2/models",
			"GET /api/v2/models/{id}",
			"GET /api/v2/models/type/{type}",
			"GET /api/v2/stats",
			"POST /api/v2/refresh",
			"GET /api/v2/health",
			"GET /api/v2/status",
		},
		"stats": stats,
	}

	c.JSON(http.StatusOK, status)
}