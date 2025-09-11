package recommendation

import (
	"math"
	"sort"
	"strings"

	"github.com/Askeban/llm-router-go/internal/models"
)

// RecommendationRequest represents a user's model recommendation request
type RecommendationRequest struct {
	TaskType     string                 `json:"task_type"`     // "text", "image", "video", "audio", "multimodal"
	Category     string                 `json:"category"`      // "coding", "math", "creative", etc.
	Complexity   string                 `json:"complexity"`    // "simple", "medium", "hard", "expert"
	Priority     string                 `json:"priority"`      // "quality", "speed", "cost", "balanced"
	Requirements map[string]interface{} `json:"requirements"`  // Special requirements
	Context      string                 `json:"context,omitempty"` // Optional context for better matching
}

// ScoredRecommendation represents a model with its recommendation score
type ScoredRecommendation struct {
	Model           models.EnhancedModel   `json:"model"`
	OverallScore    float64                `json:"overall_score"`
	ComponentScores map[string]float64     `json:"component_scores"`
	Reasoning       string                 `json:"reasoning"`
	Confidence      float64                `json:"confidence"`
	CostEstimate    float64                `json:"cost_estimate"`
	Warnings        []string               `json:"warnings,omitempty"`
}

// RecommendationResponse contains the full recommendation result
type RecommendationResponse struct {
	Request        RecommendationRequest  `json:"request"`
	Recommendations []ScoredRecommendation `json:"recommendations"`
	TotalModels    int                    `json:"total_models"`
	FilteredModels int                    `json:"filtered_models"`
	ProcessingTime float64                `json:"processing_time_ms"`
	Metadata       RecommendationMetadata `json:"metadata"`
}

type RecommendationMetadata struct {
	AlgorithmVersion string                 `json:"algorithm_version"`
	DataSources      []string               `json:"data_sources"`
	Weights          map[string]float64     `json:"weights"`
	AppliedFilters   []string               `json:"applied_filters"`
}

// EnhancedRecommendationEngine provides intelligent model recommendations
type EnhancedRecommendationEngine struct {
	fusionService *models.FusionService
}

func NewEnhancedRecommendationEngine(fusionService *models.FusionService) *EnhancedRecommendationEngine {
	return &EnhancedRecommendationEngine{
		fusionService: fusionService,
	}
}

func (ere *EnhancedRecommendationEngine) GetRecommendations(req RecommendationRequest) RecommendationResponse {
	startTime := getCurrentTimeMs()

	// Get all available models
	allModels := ere.fusionService.GetAllModels()

	// Filter models by task type and basic requirements
	filteredModels := ere.filterModels(allModels, req)

	// Score each filtered model
	scoredModels := make([]ScoredRecommendation, 0, len(filteredModels))
	for _, model := range filteredModels {
		scored := ere.scoreModel(model, req)
		if scored.OverallScore > 0.1 { // Only include models with reasonable scores
			scoredModels = append(scoredModels, scored)
		}
	}

	// Sort by overall score (descending)
	sort.Slice(scoredModels, func(i, j int) bool {
		return scoredModels[i].OverallScore > scoredModels[j].OverallScore
	})

	// Limit to top 10 recommendations
	maxResults := 10
	if len(scoredModels) > maxResults {
		scoredModels = scoredModels[:maxResults]
	}

	endTime := getCurrentTimeMs()
	processingTime := endTime - startTime

	return RecommendationResponse{
		Request:         req,
		Recommendations: scoredModels,
		TotalModels:     len(allModels),
		FilteredModels:  len(filteredModels),
		ProcessingTime:  processingTime,
		Metadata: RecommendationMetadata{
			AlgorithmVersion: "2.0",
			DataSources:      []string{"model_1.json", "analytics-ai"},
			Weights:          ere.getWeights(req.Priority),
			AppliedFilters:   ere.getAppliedFilters(req),
		},
	}
}

func (ere *EnhancedRecommendationEngine) filterModels(allModels []models.EnhancedModel, req RecommendationRequest) []models.EnhancedModel {
	var filtered []models.EnhancedModel

	for _, model := range allModels {
		// Filter by model type
		if !ere.isModelTypeMatch(model, req.TaskType) {
			continue
		}

		// Filter by capability availability
		if !ere.hasRequiredCapability(model, req.Category, req.TaskType) {
			continue
		}

		// Filter by complexity requirements
		if !ere.meetsComplexityRequirement(model, req.Category, req.Complexity, req.TaskType) {
			continue
		}

		// Apply special requirements filters
		if !ere.meetsSpecialRequirements(model, req.Requirements) {
			continue
		}

		filtered = append(filtered, model)
	}

	return filtered
}

func (ere *EnhancedRecommendationEngine) isModelTypeMatch(model models.EnhancedModel, taskType string) bool {
	if taskType == "multimodal" {
		// Multimodal tasks can use any model type, prefer multimodal models
		return true
	}
	return model.ModelType == taskType
}

func (ere *EnhancedRecommendationEngine) hasRequiredCapability(model models.EnhancedModel, category, taskType string) bool {
	if taskType == "text" {
		_, hasCapability := model.TaskCapabilities.TextTasks[category]
		return hasCapability
	} else if taskType == "image" {
		_, hasCapability := model.TaskCapabilities.GenerativeTasks["image_generation"]
		return hasCapability
	} else if taskType == "video" {
		_, hasCapability := model.TaskCapabilities.GenerativeTasks["video_generation"]
		return hasCapability
	} else if taskType == "audio" {
		_, hasCapability := model.TaskCapabilities.GenerativeTasks["audio_generation"]
		return hasCapability
	}

	return true // Default to allowing model
}

func (ere *EnhancedRecommendationEngine) meetsComplexityRequirement(model models.EnhancedModel, category, complexity, taskType string) bool {
	if taskType == "text" {
		if taskCap, exists := model.TaskCapabilities.TextTasks[category]; exists {
			return ere.supportsComplexity(taskCap.ComplexityRange, complexity)
		}
	} else {
		// For generative tasks, check if max complexity meets requirement
		taskKey := taskType + "_generation"
		if genCap, exists := model.TaskCapabilities.GenerativeTasks[taskKey]; exists {
			return ere.complexityLevelMet(genCap.MaxComplexity, complexity)
		}
	}

	return true // Default to allowing model
}

func (ere *EnhancedRecommendationEngine) supportsComplexity(supportedRanges []string, requiredComplexity string) bool {
	complexityOrder := map[string]int{
		"simple": 1,
		"medium": 2,
		"hard":   3,
		"expert": 4,
	}

	requiredLevel := complexityOrder[requiredComplexity]
	if requiredLevel == 0 {
		return true // Unknown complexity, allow all
	}

	for _, supported := range supportedRanges {
		supportedLevel := complexityOrder[supported]
		if supportedLevel >= requiredLevel {
			return true
		}
	}

	return false
}

func (ere *EnhancedRecommendationEngine) complexityLevelMet(maxComplexity, requiredComplexity string) bool {
	complexityOrder := map[string]int{
		"simple": 1,
		"medium": 2,
		"hard":   3,
		"expert": 4,
	}

	maxLevel := complexityOrder[maxComplexity]
	requiredLevel := complexityOrder[requiredComplexity]

	return maxLevel >= requiredLevel
}

func (ere *EnhancedRecommendationEngine) meetsSpecialRequirements(model models.EnhancedModel, requirements map[string]interface{}) bool {
	// Check cost requirements
	if maxCost, exists := requirements["max_cost"]; exists {
		if cost, ok := maxCost.(float64); ok {
			if model.Pricing.Text.CostOutPer1K != nil && *model.Pricing.Text.CostOutPer1K > cost {
				return false
			}
		}
	}

	// Check speed requirements
	if minSpeed, exists := requirements["min_speed"]; exists {
		if speed, ok := minSpeed.(float64); ok {
			if model.Performance.Latency.ThroughputTokensSec != nil && *model.Performance.Latency.ThroughputTokensSec < speed {
				return false
			}
		}
	}

	// Check open source requirement
	if openSourceRequired, exists := requirements["open_source"]; exists {
		if required, ok := openSourceRequired.(bool); ok && required {
			if !model.OpenSource {
				return false
			}
		}
	}

	// Check free tier requirement
	if freeTierRequired, exists := requirements["free_tier"]; exists {
		if required, ok := freeTierRequired.(bool); ok && required {
			if !model.Pricing.FreeTier {
				return false
			}
		}
	}

	return true
}

func (ere *EnhancedRecommendationEngine) scoreModel(model models.EnhancedModel, req RecommendationRequest) ScoredRecommendation {
	weights := ere.getWeights(req.Priority)
	components := make(map[string]float64)

	// 1. Task Capability Alignment (40% default weight)
	capabilityScore := ere.getCapabilityScore(model, req.TaskType, req.Category)
	components["capability"] = capabilityScore

	// 2. Complexity Match (25% default weight)
	complexityScore := ere.getComplexityScore(model, req.Complexity, req.Category, req.TaskType)
	components["complexity"] = complexityScore

	// 3. Performance Metrics (20% default weight)
	performanceScore := ere.getPerformanceScore(model, req.Priority)
	components["performance"] = performanceScore

	// 4. Community Intelligence (10% default weight)
	communityScore := ere.getCommunityScore(model, req.Category)
	components["community"] = communityScore

	// 5. Benchmark Alignment (5% default weight)
	benchmarkScore := ere.getBenchmarkScore(model, req.Category, req.TaskType)
	components["benchmark"] = benchmarkScore

	// Calculate weighted overall score
	overallScore := (capabilityScore * weights["capability"]) +
		(complexityScore * weights["complexity"]) +
		(performanceScore * weights["performance"]) +
		(communityScore * weights["community"]) +
		(benchmarkScore * weights["benchmark"])

	// Apply priority-based adjustments
	overallScore = ere.applyPriorityModifiers(overallScore, req.Priority, model)

	// Calculate confidence
	confidence := ere.calculateConfidence(model, components)

	// Generate reasoning
	reasoning := ere.generateReasoning(req, model, components, overallScore)

	// Calculate cost estimate
	costEstimate := ere.estimateCost(req, model)

	// Generate warnings
	warnings := ere.generateWarnings(req, model)

	return ScoredRecommendation{
		Model:           model,
		OverallScore:    math.Min(overallScore, 1.0), // Cap at 1.0
		ComponentScores: components,
		Reasoning:       reasoning,
		Confidence:      confidence,
		CostEstimate:    costEstimate,
		Warnings:        warnings,
	}
}

func (ere *EnhancedRecommendationEngine) getCapabilityScore(model models.EnhancedModel, taskType, category string) float64 {
	if taskType == "text" {
		if taskCap, exists := model.TaskCapabilities.TextTasks[category]; exists {
			return taskCap.Score
		}
		// Fallback to Analytics AI indices
		if model.Benchmarks.CompositeIndices.AnalyticsAICoding != nil && category == "coding" {
			return *model.Benchmarks.CompositeIndices.AnalyticsAICoding
		}
		if model.Benchmarks.CompositeIndices.AnalyticsAIMath != nil && category == "math" {
			return *model.Benchmarks.CompositeIndices.AnalyticsAIMath
		}
		if model.Benchmarks.CompositeIndices.AnalyticsAIIntelligence != nil && category == "reasoning" {
			return *model.Benchmarks.CompositeIndices.AnalyticsAIIntelligence
		}
		return 0.7 // Default capability score
	} else if taskType == "image" {
		if genCap, exists := model.TaskCapabilities.GenerativeTasks["image_generation"]; exists {
			return genCap.Score
		}
	} else if taskType == "video" {
		if genCap, exists := model.TaskCapabilities.GenerativeTasks["video_generation"]; exists {
			return genCap.Score
		}
	} else if taskType == "audio" {
		if genCap, exists := model.TaskCapabilities.GenerativeTasks["audio_generation"]; exists {
			return genCap.Score
		}
	} else if taskType == "multimodal" {
		// For multimodal, use the average of available capabilities
		scores := []float64{}
		if taskCap, exists := model.TaskCapabilities.TextTasks[category]; exists {
			scores = append(scores, taskCap.Score)
		}
		if len(model.TaskCapabilities.GenerativeTasks) > 0 {
			for _, genCap := range model.TaskCapabilities.GenerativeTasks {
				scores = append(scores, genCap.Score)
			}
		}
		if len(scores) > 0 {
			return ere.average(scores)
		}
	}

	return 0.0
}

func (ere *EnhancedRecommendationEngine) getComplexityScore(model models.EnhancedModel, complexity, category, taskType string) float64 {
	if taskType == "text" {
		if taskCap, exists := model.TaskCapabilities.TextTasks[category]; exists {
			if ere.supportsComplexity(taskCap.ComplexityRange, complexity) {
				// Perfect match gets full score
				if ere.isOptimalComplexity(taskCap.ComplexityRange, complexity) {
					return 1.0
				}
				return 0.8 // Supports but not optimal
			}
			return 0.3 // Doesn't support required complexity
		}
	} else {
		// For generative tasks
		taskKey := taskType + "_generation"
		if genCap, exists := model.TaskCapabilities.GenerativeTasks[taskKey]; exists {
			if ere.complexityLevelMet(genCap.MaxComplexity, complexity) {
				return 0.9
			}
			return 0.4
		}
	}

	return 0.5 // Default neutral score
}

func (ere *EnhancedRecommendationEngine) isOptimalComplexity(supportedRanges []string, requiredComplexity string) bool {
	// Check if the required complexity is in the "sweet spot" of supported ranges
	for _, supported := range supportedRanges {
		if supported == requiredComplexity {
			return true
		}
	}
	return false
}

func (ere *EnhancedRecommendationEngine) getPerformanceScore(model models.EnhancedModel, priority string) float64 {
	score := 0.0
	components := 0

	// Latency scoring
	if model.Performance.Latency.AvgLatencyMs != nil {
		latency := float64(*model.Performance.Latency.AvgLatencyMs)
		// Normalize latency: lower is better, scale 0-1
		normalizedLatency := 1.0 - math.Min(latency/10000.0, 1.0) // 10s is very slow
		score += normalizedLatency
		components++
	}

	// Throughput scoring  
	if model.Performance.Latency.ThroughputTokensSec != nil {
		throughput := *model.Performance.Latency.ThroughputTokensSec
		// Normalize throughput: higher is better, scale 0-1
		normalizedThroughput := math.Min(throughput/200.0, 1.0) // 200 tokens/sec is very good
		score += normalizedThroughput
		components++
	}

	// Availability scoring
	if model.Performance.Availability.UptimePercentage != nil {
		uptime := *model.Performance.Availability.UptimePercentage
		score += uptime // Already 0-1 scale
		components++
	}

	if components > 0 {
		score = score / float64(components)
	} else {
		score = 0.7 // Default score
	}

	// Apply priority-based weighting
	if priority == "speed" {
		score = score * 1.2 // Boost performance importance for speed priority
	}

	return math.Min(score, 1.0)
}

func (ere *EnhancedRecommendationEngine) getCommunityScore(model models.EnhancedModel, category string) float64 {
	score := 0.0
	components := 0

	// Reddit sentiment
	if model.CommunityIntelligence.RedditSentiment != nil {
		score += *model.CommunityIntelligence.RedditSentiment
		components++
	}

	// Developer rating (convert 1-5 scale to 0-1)
	if model.CommunityIntelligence.DeveloperRating != nil {
		normalizedRating := (*model.CommunityIntelligence.DeveloperRating - 1) / 4
		score += normalizedRating
		components++
	}

	// GitHub activity (if available)
	if model.CommunityIntelligence.GitHubActivity.Stars != nil {
		stars := float64(*model.CommunityIntelligence.GitHubActivity.Stars)
		// Normalize stars: log scale for popularity
		normalizedStars := math.Min(math.Log10(stars+1)/5.0, 1.0) // 100k stars = 1.0
		score += normalizedStars
		components++
	}

	// Category-specific usage patterns
	categoryBonus := 0.0
	for _, useCase := range model.CommunityIntelligence.UsagePatterns.TopUseCases {
		if useCase == category {
			categoryBonus = 0.2
			break
		}
	}

	if components > 0 {
		score = score / float64(components)
	} else {
		score = 0.6 // Default score
	}

	return math.Min(score+categoryBonus, 1.0)
}

func (ere *EnhancedRecommendationEngine) getBenchmarkScore(model models.EnhancedModel, category, taskType string) float64 {
	if taskType != "text" {
		// For generative tasks, use generative benchmarks
		return ere.getGenerativeBenchmarkScore(model, taskType)
	}

	// For text tasks, use raw benchmarks
	benchmarks := model.Benchmarks.RawBenchmarks
	switch category {
	case "coding":
		if benchmarks.HumanEval != nil {
			return *benchmarks.HumanEval
		}
		if benchmarks.LiveCodeBench != nil {
			return *benchmarks.LiveCodeBench
		}
		if benchmarks.SWEBench != nil {
			return *benchmarks.SWEBench
		}
	case "math":
		if benchmarks.GSM8K != nil {
			return *benchmarks.GSM8K
		}
		if benchmarks.Math500 != nil {
			return *benchmarks.Math500
		}
		if benchmarks.AIME != nil {
			return *benchmarks.AIME
		}
	case "reasoning":
		if benchmarks.MMLU != nil {
			return *benchmarks.MMLU
		}
		if benchmarks.MMLUPro != nil {
			return *benchmarks.MMLUPro
		}
		if benchmarks.ARC != nil {
			return *benchmarks.ARC
		}
	}

	return 0.7 // Default benchmark score
}

func (ere *EnhancedRecommendationEngine) getGenerativeBenchmarkScore(model models.EnhancedModel, taskType string) float64 {
	switch taskType {
	case "image":
		if model.Benchmarks.GenerativeBenchmarks.Image.CLIPScore != nil {
			return *model.Benchmarks.GenerativeBenchmarks.Image.CLIPScore
		}
		if model.Benchmarks.GenerativeBenchmarks.Image.UserPreference != nil {
			return *model.Benchmarks.GenerativeBenchmarks.Image.UserPreference
		}
	case "video":
		if model.Benchmarks.GenerativeBenchmarks.Video.TemporalConsistency != nil {
			return *model.Benchmarks.GenerativeBenchmarks.Video.TemporalConsistency
		}
		if model.Benchmarks.GenerativeBenchmarks.Video.UserStudies != nil {
			return *model.Benchmarks.GenerativeBenchmarks.Video.UserStudies
		}
	case "audio":
		if model.Benchmarks.GenerativeBenchmarks.Audio.NaturalnessMOS != nil {
			// Convert MOS (1-5) to 0-1 scale
			return (*model.Benchmarks.GenerativeBenchmarks.Audio.NaturalnessMOS - 1) / 4
		}
		if model.Benchmarks.GenerativeBenchmarks.Audio.SimilarityScore != nil {
			return *model.Benchmarks.GenerativeBenchmarks.Audio.SimilarityScore
		}
	}

	return 0.7 // Default score
}

func (ere *EnhancedRecommendationEngine) applyPriorityModifiers(score float64, priority string, model models.EnhancedModel) float64 {
	switch priority {
	case "cost":
		// Boost score for cost-effective models
		if model.Pricing.FreeTier {
			score *= 1.1
		}
		if model.Pricing.Text.CostOutPer1K != nil && *model.Pricing.Text.CostOutPer1K < 0.01 {
			score *= 1.1 // Low cost models get boost
		}
	case "speed":
		// Already handled in performance scoring
		break
	case "quality":
		// Boost models with high confidence and benchmarks
		if model.ConfidenceScore > 0.9 {
			score *= 1.05
		}
	}

	return score
}

func (ere *EnhancedRecommendationEngine) calculateConfidence(model models.EnhancedModel, components map[string]float64) float64 {
	// Base confidence from model data quality
	baseConfidence := model.ConfidenceScore
	if baseConfidence == 0 {
		baseConfidence = 0.7
	}

	// Adjust based on data completeness
	completeness := 0.0
	if components["capability"] > 0 {
		completeness += 0.3
	}
	if components["performance"] > 0 {
		completeness += 0.2
	}
	if components["community"] > 0 {
		completeness += 0.2
	}
	if components["benchmark"] > 0 {
		completeness += 0.3
	}

	// Boost confidence for Analytics AI verified models
	analyticsBoost := 0.0
	for _, tag := range model.Tags {
		if tag == "analytics-ai-verified" {
			analyticsBoost = 0.1
			break
		}
	}

	return math.Min(baseConfidence*completeness+analyticsBoost, 1.0)
}

func (ere *EnhancedRecommendationEngine) generateReasoning(req RecommendationRequest, model models.EnhancedModel, components map[string]float64, score float64) string {
	reasons := []string{}

	// Score-based reasoning
	if score > 0.9 {
		reasons = append(reasons, "Exceptional match for "+req.Category+" tasks")
	} else if score > 0.8 {
		reasons = append(reasons, "Excellent choice for "+req.Category)
	} else if score > 0.7 {
		reasons = append(reasons, "Good fit for "+req.Category)
	} else if score > 0.6 {
		reasons = append(reasons, "Suitable for "+req.Category)
	}

	// Capability-specific reasoning
	if components["capability"] > 0.9 {
		reasons = append(reasons, "Strong specialized capabilities")
	}

	// Performance reasoning
	if components["performance"] > 0.8 {
		if req.Priority == "speed" {
			reasons = append(reasons, "Optimized for fast response times")
		} else {
			reasons = append(reasons, "Excellent performance metrics")
		}
	}

	// Community reasoning
	if components["community"] > 0.8 {
		reasons = append(reasons, "Highly rated by the community")
	}

	// Provider-specific reasoning
	if model.OpenSource {
		reasons = append(reasons, "Open source with transparent development")
	}
	if model.Pricing.FreeTier {
		reasons = append(reasons, "Offers free tier for testing")
	}

	// Usage pattern reasoning
	for _, useCase := range model.CommunityIntelligence.UsagePatterns.TopUseCases {
		if useCase == req.Category {
			reasons = append(reasons, "Popular choice for "+req.Category+" tasks")
			break
		}
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "Matches basic requirements")
	}

	return strings.Join(reasons, ". ")
}

func (ere *EnhancedRecommendationEngine) estimateCost(req RecommendationRequest, model models.EnhancedModel) float64 {
	if req.TaskType == "text" {
		// Estimate cost for text tasks
		if model.Pricing.Text.CostOutPer1K != nil {
			// Assume 1000 output tokens for estimation
			return *model.Pricing.Text.CostOutPer1K
		}
	} else if req.TaskType == "image" {
		if model.Pricing.Generative.CostPerImage != nil {
			return *model.Pricing.Generative.CostPerImage
		}
	} else if req.TaskType == "video" {
		if model.Pricing.Generative.CostPerVideoSecond != nil {
			// Assume 10 seconds for estimation
			return *model.Pricing.Generative.CostPerVideoSecond * 10
		}
	} else if req.TaskType == "audio" {
		if model.Pricing.Generative.CostPerAudioMinute != nil {
			// Assume 1 minute for estimation
			return *model.Pricing.Generative.CostPerAudioMinute
		}
	}

	return 0.0 // Unknown cost
}

func (ere *EnhancedRecommendationEngine) generateWarnings(req RecommendationRequest, model models.EnhancedModel) []string {
	warnings := []string{}

	// Cost warnings
	if req.Priority == "cost" {
		if model.Pricing.Text.CostOutPer1K != nil && *model.Pricing.Text.CostOutPer1K > 0.05 {
			warnings = append(warnings, "Higher cost model - consider usage volume")
		}
	}

	// Complexity warnings
	if req.Complexity == "expert" {
		if req.TaskType == "text" {
			if taskCap, exists := model.TaskCapabilities.TextTasks[req.Category]; exists {
				if !ere.supportsComplexity(taskCap.ComplexityRange, "expert") {
					warnings = append(warnings, "May not handle expert-level "+req.Category+" tasks optimally")
				}
			}
		}
	}

	// Availability warnings
	if model.Performance.Availability.UptimePercentage != nil && *model.Performance.Availability.UptimePercentage < 0.95 {
		warnings = append(warnings, "Lower availability model - consider backup options")
	}

	// Community warnings
	for _, weakness := range model.CommunityIntelligence.UsagePatterns.ReportedWeaknesses {
		if strings.Contains(strings.ToLower(weakness), strings.ToLower(req.Category)) {
			warnings = append(warnings, "Community reports issues with "+req.Category+": "+weakness)
		}
	}

	return warnings
}

// Helper functions
func (ere *EnhancedRecommendationEngine) getWeights(priority string) map[string]float64 {
	switch priority {
	case "quality":
		return map[string]float64{
			"capability":  0.50,
			"complexity":  0.25,
			"performance": 0.10,
			"community":   0.10,
			"benchmark":   0.05,
		}
	case "speed":
		return map[string]float64{
			"capability":  0.30,
			"complexity":  0.15,
			"performance": 0.40,
			"community":   0.10,
			"benchmark":   0.05,
		}
	case "cost":
		return map[string]float64{
			"capability":  0.30,
			"complexity":  0.20,
			"performance": 0.10,
			"community":   0.25,
			"benchmark":   0.15,
		}
	default: // balanced
		return map[string]float64{
			"capability":  0.40,
			"complexity":  0.25,
			"performance": 0.20,
			"community":   0.10,
			"benchmark":   0.05,
		}
	}
}

func (ere *EnhancedRecommendationEngine) getAppliedFilters(req RecommendationRequest) []string {
	filters := []string{}
	filters = append(filters, "model_type:"+req.TaskType)
	filters = append(filters, "category:"+req.Category)
	filters = append(filters, "complexity:"+req.Complexity)

	if req.Requirements != nil {
		if _, exists := req.Requirements["open_source"]; exists {
			filters = append(filters, "open_source")
		}
		if _, exists := req.Requirements["free_tier"]; exists {
			filters = append(filters, "free_tier")
		}
		if _, exists := req.Requirements["max_cost"]; exists {
			filters = append(filters, "cost_limit")
		}
	}

	return filters
}

func (ere *EnhancedRecommendationEngine) average(numbers []float64) float64 {
	if len(numbers) == 0 {
		return 0
	}
	sum := 0.0
	for _, num := range numbers {
		sum += num
	}
	return sum / float64(len(numbers))
}

func getCurrentTimeMs() float64 {
	return float64(0) // Placeholder - implement with actual time measurement
}