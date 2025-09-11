package recommendation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Askeban/llm-router-go/internal/classifier"
	"github.com/Askeban/llm-router-go/internal/models"
)

// Mock HTTP server for classifier service
func createMockClassifier() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/classify/advanced" {
			// Return advanced classification response
			response := classifier.EnhancedResp{
				PrimaryUseCase:       "coding",
				ComplexityScore:      0.75,
				CreativityScore:      0.3,
				TokenCountEstimate:   150,
				UrgencyLevel:         0.5,
				OutputLengthEstimate: 300,
				InteractionStyle:     "direct",
				DomainConfidence:     0.85,
				Difficulty:           "medium",
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/classify" {
			// Return basic classification response
			response := classifier.EnhancedResp{
				PrimaryUseCase:       "coding",
				ComplexityScore:      0.75,
				CreativityScore:      0.3,
				TokenCountEstimate:   150,
				UrgencyLevel:         0.5,
				OutputLengthEstimate: 300,
				InteractionStyle:     "direct",
				DomainConfidence:     0.85,
				Difficulty:           "medium",
			}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// Create sample models for testing
func createSampleModels() []models.ModelProfile {
	nowStr := time.Now().Format(time.RFC3339)

	return []models.ModelProfile{
		{
			ID:            "gpt-4",
			Provider:      "openai",
			DisplayName:   "GPT-4",
			CostInPer1K:   0.03,
			ContextWindow: 8192,
			AvgLatencyMs:  2000,
			BestAt:        []string{"coding", "reasoning"},
			Capabilities: map[string]float64{
				"coding":           0.9,
				"reasoning":        0.85,
				"creative_writing": 0.8,
			},
			BenchmarkScores: map[string]float64{
				"humaneval": 67.0,
				"gsm8k":     92.0,
				"mmlu":      86.4,
			},
			EnhancedCapabilities: map[string]models.CapabilityScore{
				"coding": {
					Score:       0.9,
					Confidence:  0.95,
					LastUpdated: &nowStr,
				},
			},
			ComplexityProfiles: map[string]models.ComplexityProfile{
				"high": {
					Score:      0.9,
					Confidence: 0.9,
				},
				"medium": {
					Score:      0.85,
					Confidence: 0.95,
				},
			},
			RoutingMetadata: &models.RoutingMetadata{
				UsageCount:    150,
				SuccessRate:   0.92,
				AverageRating: 4.5,
			},
		},
		{
			ID:            "claude-3-opus",
			Provider:      "anthropic",
			DisplayName:   "Claude 3 Opus",
			CostInPer1K:   0.075,
			ContextWindow: 200000,
			AvgLatencyMs:  1800,
			BestAt:        []string{"creative_writing", "reasoning"},
			Capabilities: map[string]float64{
				"creative_writing": 0.95,
				"reasoning":        0.9,
				"coding":           0.8,
			},
			BenchmarkScores: map[string]float64{
				"humaneval": 84.9,
				"gsm8k":     95.0,
				"mmlu":      86.8,
			},
			EnhancedCapabilities: map[string]models.CapabilityScore{
				"creative_writing": {
					Score:       0.95,
					Confidence:  0.98,
					LastUpdated: &nowStr,
				},
			},
			RoutingMetadata: &models.RoutingMetadata{
				UsageCount:    200,
				SuccessRate:   0.94,
				AverageRating: 4.7,
			},
		},
		{
			ID:            "llama-2-70b",
			Provider:      "meta",
			DisplayName:   "Llama 2 70B",
			CostInPer1K:   0.002,
			ContextWindow: 4096,
			AvgLatencyMs:  3000,
			BestAt:        []string{"general"},
			Capabilities: map[string]float64{
				"coding":           0.6,
				"reasoning":        0.7,
				"creative_writing": 0.65,
			},
			BenchmarkScores: map[string]float64{
				"humaneval": 45.0,
				"gsm8k":     78.0,
				"mmlu":      68.9,
			},
			RoutingMetadata: &models.RoutingMetadata{
				UsageCount:    50,
				SuccessRate:   0.78,
				AverageRating: 3.8,
			},
		},
	}
}

func TestEnhancedRecommendationEngine_BasicRecommendation(t *testing.T) {
	// Setup mock classifier
	mockClassifier := createMockClassifier()
	defer mockClassifier.Close()

	// Create engine
	engine := NewEnhancedRecommendationEngine(mockClassifier.URL)

	// Test data
	models := createSampleModels()
	prompt := "Write a Python function to implement binary search"

	// Test recommendation
	result, err := engine.RecommendModel(prompt, models, nil, PreferenceBalanced)

	if err != nil {
		t.Fatalf("Recommendation failed: %v", err)
	}

	// Validate result structure
	if result.RecommendedModel.ID == "" {
		t.Error("No recommended model returned")
	}

	if result.RecommendationScore <= 0 {
		t.Error("Invalid recommendation score")
	}

	if result.Strategy != PreferenceBalanced {
		t.Errorf("Expected strategy %s, got %s", PreferenceBalanced, result.Strategy)
	}

	if len(result.Stage2Candidates) == 0 {
		t.Error("No candidates were scored")
	}

	// Should filter some models but not all
	if result.Stage1Filtered == 0 {
		t.Log("Warning: No models were filtered in stage 1")
	}

	if result.Stage1Filtered >= len(models) {
		t.Error("All models were filtered out")
	}
}

func TestEnhancedRecommendationEngine_PreferenceStrategies(t *testing.T) {
	mockClassifier := createMockClassifier()
	defer mockClassifier.Close()

	engine := NewEnhancedRecommendationEngine(mockClassifier.URL)
	models := createSampleModels()
	prompt := "Write a creative story about AI"

	// Test each preference strategy
	preferences := []RecommendationPreference{
		PreferencePerformance,
		PreferenceBalanced,
		PreferenceCostSaver,
	}

	results := make(map[RecommendationPreference]*RecommendationResult)

	for _, pref := range preferences {
		result, err := engine.RecommendModel(prompt, models, nil, pref)
		if err != nil {
			t.Fatalf("Recommendation failed for %s: %v", pref, err)
		}
		results[pref] = result
	}

	// Performance preference should pick the highest scoring model
	perfResult := results[PreferencePerformance]
	if perfResult.Strategy != PreferencePerformance {
		t.Error("Performance preference not applied correctly")
	}

	// Cost saver should pick a cheaper model
	costResult := results[PreferenceCostSaver]
	if costResult.Strategy != PreferenceCostSaver {
		t.Error("Cost saver preference not applied correctly")
	}

	// Cost saver should generally be cheaper than performance (though not always guaranteed)
	if costResult.RecommendedModel.CostInPer1K > perfResult.RecommendedModel.CostInPer1K {
		t.Log("Warning: Cost saver picked more expensive model than performance")
	}

	// All should have valid reasoning
	for pref, result := range results {
		if strings.TrimSpace(result.Reasoning) == "" {
			t.Errorf("No reasoning provided for %s preference", pref)
		}
	}
}

func TestEnhancedRecommendationEngine_Constraints(t *testing.T) {
	mockClassifier := createMockClassifier()
	defer mockClassifier.Close()

	engine := NewEnhancedRecommendationEngine(mockClassifier.URL)
	models := createSampleModels()
	prompt := "Help me with a coding task"

	// Test cost constraint
	maxCost := 0.01
	constraints := &CustomerConstraints{
		MaxCostPer1K: &maxCost,
	}

	result, err := engine.RecommendModel(prompt, models, constraints, PreferenceBalanced)
	if err != nil {
		t.Fatalf("Recommendation with constraints failed: %v", err)
	}

	if result.RecommendedModel.CostInPer1K > maxCost {
		t.Errorf("Recommended model cost (%.3f) exceeds constraint (%.3f)",
			result.RecommendedModel.CostInPer1K, maxCost)
	}

	// Test provider constraint
	constraints = &CustomerConstraints{
		AllowedProviders: []string{"openai"},
	}

	result, err = engine.RecommendModel(prompt, models, constraints, PreferenceBalanced)
	if err != nil {
		t.Fatalf("Recommendation with provider constraint failed: %v", err)
	}

	if !strings.EqualFold(result.RecommendedModel.Provider, "openai") {
		t.Errorf("Recommended model provider (%s) not in allowed list",
			result.RecommendedModel.Provider)
	}

	// Test exclusion constraint
	constraints = &CustomerConstraints{
		ExcludedModels: []string{"gpt-4"},
	}

	result, err = engine.RecommendModel(prompt, models, constraints, PreferenceBalanced)
	if err != nil {
		t.Fatalf("Recommendation with exclusion constraint failed: %v", err)
	}

	if result.RecommendedModel.ID == "gpt-4" {
		t.Error("Excluded model was recommended")
	}
}

func TestEnhancedRecommendationEngine_ScoringComponents(t *testing.T) {
	mockClassifier := createMockClassifier()
	defer mockClassifier.Close()

	engine := NewEnhancedRecommendationEngine(mockClassifier.URL)
	models := createSampleModels()
	prompt := "Write complex algorithmic code"

	result, err := engine.RecommendModel(prompt, models, nil, PreferencePerformance)
	if err != nil {
		t.Fatalf("Recommendation failed: %v", err)
	}

	// Verify scoring details are present
	if len(result.Stage2Candidates) == 0 {
		t.Fatal("No candidates with scoring details")
	}

	for _, candidate := range result.Stage2Candidates {
		details := candidate.Details

		// Check that all scoring components are calculated
		if details.ComplexityScore < 0 || details.ComplexityScore > 1 {
			t.Errorf("Invalid complexity score: %f", details.ComplexityScore)
		}

		if details.CreativityScore < 0 || details.CreativityScore > 1 {
			t.Errorf("Invalid creativity score: %f", details.CreativityScore)
		}

		if details.BenchmarkScore < 0 || details.BenchmarkScore > 1 {
			t.Errorf("Invalid benchmark score: %f", details.BenchmarkScore)
		}

		if details.CapabilityScore < 0 || details.CapabilityScore > 1 {
			t.Errorf("Invalid capability score: %f", details.CapabilityScore)
		}

		if details.QualityScore < 0 || details.QualityScore > 1 {
			t.Errorf("Invalid quality score: %f", details.QualityScore)
		}

		// Check weighted scores
		if len(details.WeightedScores) == 0 {
			t.Error("No weighted scores calculated")
		}

		totalScore, exists := details.WeightedScores["total"]
		if !exists || totalScore != candidate.MatchScore {
			t.Error("Total weighted score doesn't match candidate match score")
		}
	}
}

func TestEnhancedRecommendationEngine_Stage1Filtering(t *testing.T) {
	mockClassifier := createMockClassifier()
	defer mockClassifier.Close()

	engine := NewEnhancedRecommendationEngine(mockClassifier.URL)
	models := createSampleModels()

	// Test context window filtering
	// Create a prompt that requires more tokens than some models can handle
	longPrompt := strings.Repeat("This is a very long prompt that requires many tokens. ", 100)

	result, err := engine.RecommendModel(longPrompt, models, nil, PreferenceBalanced)
	if err != nil {
		t.Fatalf("Recommendation failed: %v", err)
	}

	// Should filter some models due to context window
	if result.Stage1Filtered == 0 {
		t.Log("Warning: No models filtered by context window constraint")
	}

	// Test use case compatibility
	// Use a very specific use case that not all models support
	result, err = engine.RecommendModel("Solve advanced mathematics", models, nil, PreferenceBalanced)
	if err != nil {
		t.Fatalf("Recommendation failed: %v", err)
	}

	// Verify some filtering occurred
	totalAfterFiltering := len(models) - result.Stage1Filtered
	if totalAfterFiltering == len(models) {
		t.Log("Warning: No models were filtered in stage 1")
	}
	if totalAfterFiltering == 0 {
		t.Error("All models were filtered out")
	}
}

func TestEnhancedRecommendationEngine_ErrorHandling(t *testing.T) {
	// Test with invalid classifier URL
	engine := NewEnhancedRecommendationEngine("http://invalid-url")
	models := createSampleModels()

	_, err := engine.RecommendModel("test prompt", models, nil, PreferenceBalanced)
	if err == nil {
		t.Error("Expected error with invalid classifier URL")
	}

	// Test with empty models list
	mockClassifier := createMockClassifier()
	defer mockClassifier.Close()

	engine = NewEnhancedRecommendationEngine(mockClassifier.URL)

	_, err = engine.RecommendModel("test prompt", []models.ModelProfile{}, nil, PreferenceBalanced)
	if err == nil {
		t.Error("Expected error with empty models list")
	}

	// Test with empty prompt
	_, err = engine.RecommendModel("", models, nil, PreferenceBalanced)
	if err == nil {
		t.Error("Expected error with empty prompt")
	}
}

func TestEnhancedRecommendationEngine_Performance(t *testing.T) {
	mockClassifier := createMockClassifier()
	defer mockClassifier.Close()

	engine := NewEnhancedRecommendationEngine(mockClassifier.URL)
	models := createSampleModels()
	prompt := "Write a function to sort an array"

	// Measure performance
	start := time.Now()
	result, err := engine.RecommendModel(prompt, models, nil, PreferenceBalanced)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Recommendation failed: %v", err)
	}

	// Should complete reasonably quickly
	if elapsed > 5*time.Second {
		t.Errorf("Recommendation took too long: %v", elapsed)
	}

	// Processing time should be recorded
	if result.ProcessingTimeMs <= 0 {
		t.Error("Processing time not recorded")
	}

	// Processing time should be reasonable
	if result.ProcessingTimeMs > 3000 { // 3 seconds
		t.Errorf("Processing time too high: %d ms", result.ProcessingTimeMs)
	}
}

func TestEnhancedRecommendationEngine_BenchmarkScoring(t *testing.T) {
	mockClassifier := createMockClassifier()
	defer mockClassifier.Close()

	engine := NewEnhancedRecommendationEngine(mockClassifier.URL)
	models := createSampleModels()

	// Test coding prompt - should favor models with good coding benchmarks
	result, err := engine.RecommendModel("Implement a complex algorithm", models, nil, PreferencePerformance)
	if err != nil {
		t.Fatalf("Recommendation failed: %v", err)
	}

	// Find the candidate details for the recommended model
	var recommendedCandidate *CandidateModel
	for _, candidate := range result.Stage2Candidates {
		if candidate.Model.ID == result.RecommendedModel.ID {
			recommendedCandidate = &candidate
			break
		}
	}

	if recommendedCandidate == nil {
		t.Fatal("Recommended model not found in candidates")
	}

	// For coding tasks, benchmark score should be significant
	if recommendedCandidate.Details.BenchmarkScore <= 0.3 {
		t.Errorf("Low benchmark score (%.3f) for coding task",
			recommendedCandidate.Details.BenchmarkScore)
	}
}

func TestEnhancedRecommendationEngine_AlternativeModels(t *testing.T) {
	mockClassifier := createMockClassifier()
	defer mockClassifier.Close()

	engine := NewEnhancedRecommendationEngine(mockClassifier.URL)
	models := createSampleModels()
	prompt := "Help me write code"

	result, err := engine.RecommendModel(prompt, models, nil, PreferenceBalanced)
	if err != nil {
		t.Fatalf("Recommendation failed: %v", err)
	}

	// Should provide alternative recommendations
	if len(result.AlternativeModels) == 0 {
		t.Log("Warning: No alternative models provided")
	}

	// Alternatives should be different from the main recommendation
	for _, alt := range result.AlternativeModels {
		if alt.Model.ID == result.RecommendedModel.ID {
			t.Error("Alternative model is the same as recommended model")
		}
	}

	// Alternatives should be sorted by score (descending)
	for i := 1; i < len(result.AlternativeModels); i++ {
		if result.AlternativeModels[i-1].MatchScore < result.AlternativeModels[i].MatchScore {
			t.Error("Alternative models not sorted by score")
		}
	}
}

// Benchmark tests for performance measurement
func BenchmarkEnhancedRecommendationEngine_BasicRecommendation(b *testing.B) {
	mockClassifier := createMockClassifier()
	defer mockClassifier.Close()

	engine := NewEnhancedRecommendationEngine(mockClassifier.URL)
	models := createSampleModels()
	prompt := "Write a function to process data"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.RecommendModel(prompt, models, nil, PreferenceBalanced)
		if err != nil {
			b.Fatalf("Recommendation failed: %v", err)
		}
	}
}

func BenchmarkEnhancedRecommendationEngine_WithConstraints(b *testing.B) {
	mockClassifier := createMockClassifier()
	defer mockClassifier.Close()

	engine := NewEnhancedRecommendationEngine(mockClassifier.URL)
	models := createSampleModels()
	prompt := "Help me with coding"

	maxCost := 0.05
	constraints := &CustomerConstraints{
		MaxCostPer1K:     &maxCost,
		AllowedProviders: []string{"openai", "anthropic"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.RecommendModel(prompt, models, constraints, PreferenceBalanced)
		if err != nil {
			b.Fatalf("Recommendation failed: %v", err)
		}
	}
}

// Test helper function for integration testing
func TestIntegrationWithClassifierService(t *testing.T) {
	// This test requires the actual classifier service to be running
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test with real classifier service (if available)
	engine := NewEnhancedRecommendationEngine("http://localhost:5000")
	models := createSampleModels()
	prompt := "Write a Python function for data analysis"

	result, err := engine.RecommendModel(prompt, models, nil, PreferenceBalanced)

	// If classifier service is not available, this test will fail
	// That's expected in unit test environments
	if err != nil {
		t.Skipf("Classifier service not available: %v", err)
	}

	// If we reach here, validate the integration worked
	if result.RecommendedModel.ID == "" {
		t.Error("Integration test failed - no model recommended")
	}

	// Should have used the real classifier
	if result.ClassifierUsed == "" {
		t.Error("Classifier type not recorded")
	}
}
