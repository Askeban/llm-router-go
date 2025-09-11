package services

import (
	"context"
	"log"

	"github.com/Askeban/llm-router-go/internal/classification"
	"github.com/Askeban/llm-router-go/internal/models"
	"github.com/Askeban/llm-router-go/internal/recommendation"
)

// EnhancedRouterService provides the complete AI model routing functionality
type EnhancedRouterService struct {
	fusionService       *models.FusionService
	recommendationEngine *recommendation.EnhancedRecommendationEngine
	taskClassifier      *classification.TaskClassifier
}

// SmartRecommendationRequest represents a high-level request with just a prompt
type SmartRecommendationRequest struct {
	Prompt   string `json:"prompt"`
	Context  string `json:"context,omitempty"`
	UserID   string `json:"user_id,omitempty"`
}

// SmartRecommendationResponse includes both classification and recommendations
type SmartRecommendationResponse struct {
	Classification    classification.ClassificationResult      `json:"classification"`
	Recommendations   recommendation.RecommendationResponse    `json:"recommendations"`
	ProcessingTime    float64                                  `json:"total_processing_time_ms"`
}

func NewEnhancedRouterService(modelPath string) (*EnhancedRouterService, error) {
	// Initialize fusion service
	fusionService := models.NewFusionService(modelPath)
	if err := fusionService.Initialize(context.Background()); err != nil {
		return nil, err
	}

	// Initialize recommendation engine
	recommendationEngine := recommendation.NewEnhancedRecommendationEngine(fusionService)

	// Initialize task classifier
	taskClassifier := classification.NewTaskClassifier()

	return &EnhancedRouterService{
		fusionService:       fusionService,
		recommendationEngine: recommendationEngine,
		taskClassifier:      taskClassifier,
	}, nil
}

// GetSmartRecommendations analyzes a prompt and provides intelligent recommendations
func (ers *EnhancedRouterService) GetSmartRecommendations(req SmartRecommendationRequest) SmartRecommendationResponse {
	startTime := getCurrentTimeMs()

	// Step 1: Classify the prompt
	log.Printf("[ROUTER] Classifying prompt: %s", truncateString(req.Prompt, 100))
	classification := ers.taskClassifier.ClassifyPrompt(req.Prompt)

	// Step 2: Convert to recommendation request
	recRequest := ers.taskClassifier.ConvertToRecommendationRequest(classification, req.Context)

	// Step 3: Get recommendations
	log.Printf("[ROUTER] Getting recommendations for task_type=%s, category=%s, complexity=%s", 
		recRequest.TaskType, recRequest.Category, recRequest.Complexity)
	recommendations := ers.recommendationEngine.GetRecommendations(recRequest)

	endTime := getCurrentTimeMs()
	totalTime := endTime - startTime

	log.Printf("[ROUTER] Smart recommendation complete in %.2fms - %d recommendations", 
		totalTime, len(recommendations.Recommendations))

	return SmartRecommendationResponse{
		Classification:  classification,
		Recommendations: recommendations,
		ProcessingTime:  totalTime,
	}
}

// GetDirectRecommendations provides recommendations with explicit parameters
func (ers *EnhancedRouterService) GetDirectRecommendations(req recommendation.RecommendationRequest) recommendation.RecommendationResponse {
	log.Printf("[ROUTER] Getting direct recommendations for task_type=%s, category=%s", 
		req.TaskType, req.Category)
	return ers.recommendationEngine.GetRecommendations(req)
}

// GetAllModels returns all available models with their metadata
func (ers *EnhancedRouterService) GetAllModels() []models.EnhancedModel {
	return ers.fusionService.GetAllModels()
}

// GetModelsByType filters models by type
func (ers *EnhancedRouterService) GetModelsByType(modelType string) []models.EnhancedModel {
	return ers.fusionService.GetModelsByType(modelType)
}

// GetModelByID retrieves a specific model by ID
func (ers *EnhancedRouterService) GetModelByID(id string) (models.EnhancedModel, bool) {
	return ers.fusionService.GetModelByID(id)
}

// GetStats returns service statistics
func (ers *EnhancedRouterService) GetStats() map[string]interface{} {
	stats := ers.fusionService.GetStats()
	
	// Add router-specific stats
	stats["service_type"] = "enhanced_router"
	stats["features"] = []string{
		"smart_classification",
		"multi_modal_support", 
		"analytics_ai_integration",
		"community_intelligence",
		"complexity_scoring",
	}
	
	return stats
}

// RefreshData triggers a refresh of underlying data sources
func (ers *EnhancedRouterService) RefreshData(ctx context.Context) error {
	log.Printf("[ROUTER] Refreshing data sources...")
	return ers.fusionService.RefreshData(ctx)
}

// TestClassification provides a way to test the classification system
func (ers *EnhancedRouterService) TestClassification(prompt string) classification.ClassificationResult {
	return ers.taskClassifier.ClassifyPrompt(prompt)
}

// Helper functions
func getCurrentTimeMs() float64 {
	// Placeholder - should implement actual time measurement
	return 0.0
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}