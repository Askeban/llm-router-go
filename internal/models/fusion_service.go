package models

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Askeban/llm-router-go/internal/analytics"
)

// FusionService combines model_1.json data with Analytics AI real-time data
type FusionService struct {
	enhancedService  *EnhancedModelService
	analyticsService *analytics.Service
	
	// Caching and synchronization
	fusedModels map[string]EnhancedModel
	mutex       sync.RWMutex
	lastFusion  time.Time
	
	// Metrics
	analyticsSuccessCount int64
	fusionErrorCount      int64
}

func NewFusionService(modelPath string) *FusionService {
	return &FusionService{
		enhancedService:  NewEnhancedModelService(modelPath),
		analyticsService: analytics.NewService(),
		fusedModels:     make(map[string]EnhancedModel),
	}
}

func (fs *FusionService) Initialize(ctx context.Context) error {
	// Load enhanced models from model_1.json
	if err := fs.enhancedService.LoadModels(); err != nil {
		return err
	}

	// Perform initial fusion
	return fs.PerformFusion(ctx)
}

func (fs *FusionService) PerformFusion(ctx context.Context) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	log.Printf("[FUSION] Starting data fusion between model_1.json and Analytics AI")

	// Get base models from model_1.json
	baseModels := fs.enhancedService.GetAllModels()
	fs.fusedModels = make(map[string]EnhancedModel, len(baseModels))

	// Copy all models from model_1.json as base
	for _, model := range baseModels {
		fs.fusedModels[model.ID] = model
	}

	// Fetch Analytics AI data for text models
	analyticsData, err := fs.analyticsService.FetchModels()
	if err != nil {
		log.Printf("[FUSION] Warning: Failed to fetch Analytics AI data: %v", err)
		fs.fusionErrorCount++
		// Continue with model_1.json data only
	} else {
		log.Printf("[FUSION] Fetched %d models from Analytics AI", len(analyticsData))
		fs.analyticsSuccessCount++

		// Fuse Analytics AI data with existing models
		fs.fuseAnalyticsData(analyticsData)

		// Add missing text models from Analytics AI
		fs.addMissingAnalyticsModels(analyticsData)
	}

	fs.lastFusion = time.Now()
	log.Printf("[FUSION] Fusion complete. Total models: %d", len(fs.fusedModels))

	return nil
}

func (fs *FusionService) fuseAnalyticsData(analyticsModels []analytics.ModelData) {
	fusedCount := 0
	
	for _, analyticsModel := range analyticsModels {
		// Try to match with existing model_1.json models
		matchedModel, found := fs.findMatchingModel(analyticsModel)
		if found {
			// Enhance the existing model with Analytics AI data
			enhanced := fs.enhanceWithAnalyticsData(matchedModel, analyticsModel)
			fs.fusedModels[enhanced.ID] = enhanced
			fusedCount++
		}
	}

	log.Printf("[FUSION] Enhanced %d existing models with Analytics AI data", fusedCount)
}

func (fs *FusionService) findMatchingModel(analyticsModel analytics.ModelData) (EnhancedModel, bool) {
	// Try direct ID match first
	if existing, exists := fs.fusedModels[analyticsModel.ID]; exists {
		return existing, true
	}

	// Try name-based matching
	for _, existing := range fs.fusedModels {
		if fs.isModelMatch(existing, analyticsModel) {
			return existing, true
		}
	}

	return EnhancedModel{}, false
}

func (fs *FusionService) isModelMatch(enhanced EnhancedModel, analytics analytics.ModelData) bool {
	// Normalize names for comparison
	enhancedName := strings.ToLower(strings.ReplaceAll(enhanced.DisplayName, " ", "-"))
	analyticsName := strings.ToLower(strings.ReplaceAll(analytics.Name, " ", "-"))
	
	// Check if names are similar
	if strings.Contains(enhancedName, analyticsName) || strings.Contains(analyticsName, enhancedName) {
		return true
	}

	// Check provider match
	if enhanced.Provider == analytics.Creator.Slug {
		// Check if model names contain similar keywords
		enhancedKeywords := fs.extractKeywords(enhanced.DisplayName)
		analyticsKeywords := fs.extractKeywords(analytics.Name)
		
		commonKeywords := fs.countCommonKeywords(enhancedKeywords, analyticsKeywords)
		return commonKeywords >= 2 // Require at least 2 common keywords
	}

	return false
}

func (fs *FusionService) extractKeywords(name string) []string {
	// Extract meaningful keywords from model names
	name = strings.ToLower(name)
	keywords := []string{}
	
	// Common model name patterns
	if strings.Contains(name, "gpt") {
		keywords = append(keywords, "gpt")
	}
	if strings.Contains(name, "claude") {
		keywords = append(keywords, "claude")
	}
	if strings.Contains(name, "gemini") {
		keywords = append(keywords, "gemini")
	}
	if strings.Contains(name, "llama") {
		keywords = append(keywords, "llama")
	}
	if strings.Contains(name, "turbo") {
		keywords = append(keywords, "turbo")
	}
	if strings.Contains(name, "pro") {
		keywords = append(keywords, "pro")
	}
	if strings.Contains(name, "ultra") {
		keywords = append(keywords, "ultra")
	}
	if strings.Contains(name, "opus") {
		keywords = append(keywords, "opus")
	}
	if strings.Contains(name, "sonnet") {
		keywords = append(keywords, "sonnet")
	}
	if strings.Contains(name, "haiku") {
		keywords = append(keywords, "haiku")
	}

	return keywords
}

func (fs *FusionService) countCommonKeywords(keywords1, keywords2 []string) int {
	common := 0
	for _, kw1 := range keywords1 {
		for _, kw2 := range keywords2 {
			if kw1 == kw2 {
				common++
				break
			}
		}
	}
	return common
}

func (fs *FusionService) enhanceWithAnalyticsData(existing EnhancedModel, analytics analytics.ModelData) EnhancedModel {
	enhanced := existing

	// Update composite indices with Analytics AI data
	if enhanced.Benchmarks.CompositeIndices == (CompositeIndices{}) {
		enhanced.Benchmarks.CompositeIndices = CompositeIndices{}
	}

	enhanced.Benchmarks.CompositeIndices.AnalyticsAIIntelligence = analytics.Evaluations.ArtificialAnalysisIntelligenceIndex
	enhanced.Benchmarks.CompositeIndices.AnalyticsAICoding = analytics.Evaluations.ArtificialAnalysisCodingIndex
	enhanced.Benchmarks.CompositeIndices.AnalyticsAIMath = analytics.Evaluations.ArtificialAnalysisMathIndex

	// Update performance metrics
	if analytics.MedianOutputTokensPerSecond > 0 {
		enhanced.Performance.Latency.ThroughputTokensSec = &analytics.MedianOutputTokensPerSecond
	}
	if analytics.MedianTimeToFirstTokenSeconds > 0 {
		ttftMs := int(analytics.MedianTimeToFirstTokenSeconds * 1000)
		enhanced.Performance.Latency.TimeToFirstTokenMs = &ttftMs
	}

	// Update pricing if available
	if analytics.Pricing.Price1MInputTokens > 0 {
		costPer1k := analytics.Pricing.Price1MInputTokens / 1000.0
		enhanced.Pricing.Text.CostInPer1K = &costPer1k
	}
	if analytics.Pricing.Price1MOutputTokens > 0 {
		costPer1k := analytics.Pricing.Price1MOutputTokens / 1000.0
		enhanced.Pricing.Text.CostOutPer1K = &costPer1k
	}

	// Update task capabilities with Analytics AI indices
	if enhanced.TaskCapabilities.TextTasks == nil {
		enhanced.TaskCapabilities.TextTasks = make(map[string]TaskCapability)
	}

	// Map Analytics AI indices to our task capabilities
	if analytics.Evaluations.ArtificialAnalysisCodingIndex != nil {
		enhanced.TaskCapabilities.TextTasks["coding"] = TaskCapability{
			Score:      *analytics.Evaluations.ArtificialAnalysisCodingIndex,
			Confidence: 0.95, // High confidence for Analytics AI data
			ComplexityRange: []string{"simple", "medium", "hard"},
		}
	}

	if analytics.Evaluations.ArtificialAnalysisMathIndex != nil {
		enhanced.TaskCapabilities.TextTasks["math"] = TaskCapability{
			Score:      *analytics.Evaluations.ArtificialAnalysisMathIndex,
			Confidence: 0.95,
			ComplexityRange: []string{"simple", "medium", "hard"},
		}
	}

	if analytics.Evaluations.ArtificialAnalysisIntelligenceIndex != nil {
		enhanced.TaskCapabilities.TextTasks["reasoning"] = TaskCapability{
			Score:      *analytics.Evaluations.ArtificialAnalysisIntelligenceIndex,
			Confidence: 0.95,
			ComplexityRange: []string{"simple", "medium", "hard", "expert"},
		}
	}

	// Update data provenance
	enhanced.DataProvenance.DataQuality = 0.95 // Higher quality with Analytics AI fusion
	
	// Add analytics-ai source tag if not present
	analyticsSourceFound := false
	for _, tag := range enhanced.Tags {
		if tag == "analytics-ai-verified" {
			analyticsSourceFound = true
			break
		}
	}
	if !analyticsSourceFound {
		enhanced.Tags = append(enhanced.Tags, "analytics-ai-verified")
	}

	enhanced.LastUpdated = time.Now().Format("2006-01-02")

	return enhanced
}

func (fs *FusionService) addMissingAnalyticsModels(analyticsModels []analytics.ModelData) {
	addedCount := 0

	for _, analyticsModel := range analyticsModels {
		// Check if this model already exists
		_, exists := fs.findMatchingModel(analyticsModel)
		if !exists {
			// Create new model from Analytics AI data
			newModel := fs.createModelFromAnalytics(analyticsModel)
			fs.fusedModels[newModel.ID] = newModel
			addedCount++
		}
	}

	log.Printf("[FUSION] Added %d new models from Analytics AI", addedCount)
}

func (fs *FusionService) createModelFromAnalytics(analytics analytics.ModelData) EnhancedModel {
	model := EnhancedModel{
		ID:          analytics.ID,
		Provider:    analytics.Creator.Slug,
		DisplayName: analytics.Name,
		ModelType:   "text", // Analytics AI is text-only
		ReleaseDate: analytics.ReleaseDate,
		OpenSource:  fs.inferOpenSourceFromCreator(analytics.Creator.Slug),
		Tags:        []string{"text-generation", "analytics-ai-sourced"},
		ConfidenceScore: 0.90, // High confidence for Analytics AI
		LastUpdated: time.Now().Format("2006-01-02"),
	}

	// Set technical specs (limited info from Analytics AI)
	model.TechnicalSpecs = TechnicalSpecs{
		Parameters: "Unknown", // Analytics AI doesn't provide this
	}

	// Set benchmarks from Analytics AI
	model.Benchmarks = Benchmarks{
		CompositeIndices: CompositeIndices{
			AnalyticsAIIntelligence: analytics.Evaluations.ArtificialAnalysisIntelligenceIndex,
			AnalyticsAICoding:       analytics.Evaluations.ArtificialAnalysisCodingIndex,
			AnalyticsAIMath:         analytics.Evaluations.ArtificialAnalysisMathIndex,
		},
	}

	// Set task capabilities
	model.TaskCapabilities = TaskCapabilities{
		TextTasks: make(map[string]TaskCapability),
	}

	if analytics.Evaluations.ArtificialAnalysisCodingIndex != nil {
		model.TaskCapabilities.TextTasks["coding"] = TaskCapability{
			Score:      *analytics.Evaluations.ArtificialAnalysisCodingIndex,
			Confidence: 0.95,
			ComplexityRange: []string{"simple", "medium", "hard"},
		}
	}

	if analytics.Evaluations.ArtificialAnalysisMathIndex != nil {
		model.TaskCapabilities.TextTasks["math"] = TaskCapability{
			Score:      *analytics.Evaluations.ArtificialAnalysisMathIndex,
			Confidence: 0.95,
			ComplexityRange: []string{"simple", "medium", "hard"},
		}
	}

	if analytics.Evaluations.ArtificialAnalysisIntelligenceIndex != nil {
		model.TaskCapabilities.TextTasks["reasoning"] = TaskCapability{
			Score:      *analytics.Evaluations.ArtificialAnalysisIntelligenceIndex,
			Confidence: 0.95,
			ComplexityRange: []string{"simple", "medium", "hard", "expert"},
		}
	}

	// Add default capabilities
	model.TaskCapabilities.TextTasks["writing"] = TaskCapability{
		Score:      0.80, // Default for new models
		Confidence: 0.75,
		ComplexityRange: []string{"simple", "medium"},
	}
	model.TaskCapabilities.TextTasks["analysis"] = TaskCapability{
		Score:      0.75,
		Confidence: 0.75,
		ComplexityRange: []string{"simple", "medium"},
	}

	// Set performance
	model.Performance = Performance{
		Latency: LatencyMetrics{
			ThroughputTokensSec: &analytics.MedianOutputTokensPerSecond,
		},
	}
	if analytics.MedianTimeToFirstTokenSeconds > 0 {
		ttftMs := int(analytics.MedianTimeToFirstTokenSeconds * 1000)
		model.Performance.Latency.TimeToFirstTokenMs = &ttftMs
	}

	// Set pricing
	model.Pricing = PricingStructure{
		FreeTier: false,
	}
	if analytics.Pricing.Price1MInputTokens > 0 {
		costPer1k := analytics.Pricing.Price1MInputTokens / 1000.0
		model.Pricing.Text.CostInPer1K = &costPer1k
	}
	if analytics.Pricing.Price1MOutputTokens > 0 {
		costPer1k := analytics.Pricing.Price1MOutputTokens / 1000.0
		model.Pricing.Text.CostOutPer1K = &costPer1k
	}

	// Set data provenance
	model.DataProvenance = DataProvenance{
		DataQuality: 0.90,
	}

	return model
}

func (fs *FusionService) inferOpenSourceFromCreator(creator string) bool {
	openSourceCreators := map[string]bool{
		"meta":         true,
		"mistral":      true,
		"alibaba":      true,
		"deepseek":     true,
		"01-ai":        true,
		"stability-ai": true,
	}
	return openSourceCreators[creator]
}

func (fs *FusionService) GetAllModels() []EnhancedModel {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	models := make([]EnhancedModel, 0, len(fs.fusedModels))
	for _, model := range fs.fusedModels {
		models = append(models, model)
	}
	return models
}

func (fs *FusionService) GetModelByID(id string) (EnhancedModel, bool) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	model, exists := fs.fusedModels[id]
	return model, exists
}

func (fs *FusionService) GetModelsByType(modelType string) []EnhancedModel {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	var filtered []EnhancedModel
	for _, model := range fs.fusedModels {
		if model.ModelType == modelType {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

func (fs *FusionService) GetModelsByCapability(capability string, minScore float64) []EnhancedModel {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	var filtered []EnhancedModel
	for _, model := range fs.fusedModels {
		if taskCap, exists := model.TaskCapabilities.TextTasks[capability]; exists {
			if taskCap.Score >= minScore {
				filtered = append(filtered, model)
			}
		}
	}
	return filtered
}

func (fs *FusionService) GetStats() map[string]interface{} {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	// Count models by type
	typeCount := make(map[string]int)
	providerCount := make(map[string]int)
	for _, model := range fs.fusedModels {
		typeCount[model.ModelType]++
		providerCount[model.Provider]++
	}

	return map[string]interface{}{
		"total_models":            len(fs.fusedModels),
		"models_by_type":          typeCount,
		"models_by_provider":      providerCount,
		"last_fusion":             fs.lastFusion,
		"analytics_success_count": fs.analyticsSuccessCount,
		"fusion_error_count":      fs.fusionErrorCount,
	}
}

func (fs *FusionService) RefreshData(ctx context.Context) error {
	log.Printf("[FUSION] Refreshing fusion data...")
	return fs.PerformFusion(ctx)
}