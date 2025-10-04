package models

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
)

// EnhancedModel represents the complete model structure from model_1.json
type EnhancedModel struct {
	ID                      string                 `json:"id"`
	Provider                string                 `json:"provider"`
	DisplayName             string                 `json:"display_name"`
	ModelType               string                 `json:"model_type"`
	ReleaseDate             string                 `json:"release_date"`
	TechnicalSpecs          TechnicalSpecs         `json:"technical_specs"`
	Benchmarks              Benchmarks             `json:"benchmarks"`
	Pricing                 PricingStructure       `json:"pricing"`
	Performance             Performance            `json:"performance"`
	CommunityFeedback       CommunityFeedback      `json:"community_feedback"`
	CommunityIntelligence   CommunityIntelligence  `json:"community_intelligence,omitempty"`
	ComplexityRecommendations ComplexityRecommendations `json:"complexity_recommendations"`
	TaskCapabilities        TaskCapabilities       `json:"task_capabilities,omitempty"`
	LastUpdated             string                 `json:"last_updated"`
	ConfidenceScore         float64                `json:"confidence_score"`
	Sources                 []string               `json:"sources"`
	Tags                    []string               `json:"tags"`
	OpenSource              bool                   `json:"open_source"`
	DataProvenance          DataProvenance         `json:"data_provenance"`
}

// CommunityIntelligence contains community-sourced data
type CommunityIntelligence struct{
	RedditSentiment  *float64        `json:"reddit_sentiment,omitempty"`
	DeveloperRating  *float64        `json:"developer_rating,omitempty"`
	GitHubActivity   GitHubActivity  `json:"github_activity,omitempty"`
	UsagePatterns    *UsagePatterns  `json:"usage_patterns,omitempty"`
}

type GitHubActivity struct {
	Stars *int `json:"stars,omitempty"`
}

type UsagePatterns struct {
	TopUseCases          []string `json:"top_use_cases,omitempty"`
	ReportedWeaknesses   []string `json:"reported_weaknesses,omitempty"`
}

// TechnicalSpecs contains model technical specifications
type TechnicalSpecs struct {
	ContextWindow int     `json:"context_window"`
	Parameters    string  `json:"parameters"`
	MaxResolution *string `json:"max_resolution"`
	MaxDuration   *string `json:"max_duration"`
}

// Benchmarks contains performance benchmarks
type Benchmarks struct {
	Text                  map[string]float64    `json:"text,omitempty"`
	Image                 map[string]float64    `json:"image,omitempty"`
	Video                 map[string]float64    `json:"video,omitempty"`
	Audio                 map[string]float64    `json:"audio,omitempty"`
	CompositeIndices      CompositeIndices      `json:"composite_indices,omitempty"`
	RawBenchmarks         *RawBenchmarks        `json:"raw_benchmarks,omitempty"`
	GenerativeBenchmarks  *GenerativeBenchmarks `json:"generative_benchmarks,omitempty"`
}

type RawBenchmarks struct {
	// Coding benchmarks
	HumanEval     *float64 `json:"humaneval,omitempty"`
	LiveCodeBench *float64 `json:"livecodebench,omitempty"`
	SWEBench      *float64 `json:"swebench,omitempty"`

	// Math benchmarks
	GSM8K   *float64 `json:"gsm8k,omitempty"`
	Math500 *float64 `json:"math500,omitempty"`
	AIME    *float64 `json:"aime,omitempty"`

	// Reasoning benchmarks
	MMLU    *float64 `json:"mmlu,omitempty"`
	MMLUPro *float64 `json:"mmlu_pro,omitempty"`
	ARC     *float64 `json:"arc,omitempty"`
}

type GenerativeBenchmarks struct {
	ImageQuality *float64 `json:"image_quality,omitempty"`
	VideoQuality *float64 `json:"video_quality,omitempty"`
	AudioQuality *float64 `json:"audio_quality,omitempty"`
	Image        *ImageGenerativeBenchmark `json:"image,omitempty"`
	Video        *VideoGenerativeBenchmark `json:"video,omitempty"`
	Audio        *AudioGenerativeBenchmark `json:"audio,omitempty"`
}

type ImageGenerativeBenchmark struct {
	Quality        *float64 `json:"quality,omitempty"`
	CLIPScore      *float64 `json:"clip_score,omitempty"`
	UserPreference *float64 `json:"user_preference,omitempty"`
}

type VideoGenerativeBenchmark struct {
	Quality             *float64 `json:"quality,omitempty"`
	TemporalConsistency *float64 `json:"temporal_consistency,omitempty"`
	UserStudies         *float64 `json:"user_studies,omitempty"`
}

type AudioGenerativeBenchmark struct {
	Quality         *float64 `json:"quality,omitempty"`
	NaturalnessMOS  *float64 `json:"naturalness_mos,omitempty"`
	SimilarityScore *float64 `json:"similarity_score,omitempty"`
}

// CompositeIndices contains Analytics AI indices
type CompositeIndices struct {
	AnalyticsAIIntelligence *float64 `json:"analytics_ai_intelligence,omitempty"`
	AnalyticsAICoding       *float64 `json:"analytics_ai_coding,omitempty"`
	AnalyticsAIMath         *float64 `json:"analytics_ai_math,omitempty"`
}

// PricingStructure contains pricing information
type PricingStructure struct {
	Text       TextPricing       `json:"text,omitempty"`
	Image      *ImagePricing     `json:"image,omitempty"`
	Video      *VideoPricing     `json:"video,omitempty"`
	Audio      *AudioPricing     `json:"audio,omitempty"`
	Generative *GenerativePricing `json:"generative,omitempty"`
	FreeTier   bool              `json:"free_tier"`

	// Legacy fields for backward compatibility with model_1.json
	CostInPer1K          *float64 `json:"cost_in_per_1k,omitempty"`
	CostOutPer1K         *float64 `json:"cost_out_per_1k,omitempty"`
	CostPerImage         *float64 `json:"cost_per_image,omitempty"`
	CostPerVideoSecond   *float64 `json:"cost_per_video_second,omitempty"`
	CostPerAudioMinute   *float64 `json:"cost_per_audio_minute,omitempty"`
}

type GenerativePricing struct {
	CostPerImage        *float64 `json:"cost_per_image,omitempty"`
	CostPerVideo        *float64 `json:"cost_per_video,omitempty"`
	CostPerVideoSecond  *float64 `json:"cost_per_video_second,omitempty"`
	CostPerAudio        *float64 `json:"cost_per_audio,omitempty"`
	CostPerAudioMinute  *float64 `json:"cost_per_audio_minute,omitempty"`
}

type TextPricing struct {
	CostInPer1K  *float64 `json:"cost_in_per_1k,omitempty"`
	CostOutPer1K *float64 `json:"cost_out_per_1k,omitempty"`
}

type ImagePricing struct {
	CostPerImage *float64 `json:"cost_per_image,omitempty"`
}

type VideoPricing struct {
	CostPerSecond *float64 `json:"cost_per_second,omitempty"`
}

type AudioPricing struct {
	CostPerMinute *float64 `json:"cost_per_minute,omitempty"`
}

// Performance contains performance metrics
type Performance struct {
	AvgLatencyMs int                   `json:"avg_latency_ms"`
	Throughput   float64               `json:"throughput"`
	Availability AvailabilityMetrics   `json:"availability,omitempty"`
	Latency      LatencyMetrics        `json:"latency,omitempty"`
}

type AvailabilityMetrics struct {
	UptimePercentage *float64 `json:"uptime_percentage,omitempty"`
}

type LatencyMetrics struct {
	TimeToFirstTokenMs  *int     `json:"ttft_ms,omitempty"`
	ThroughputTokensSec *float64 `json:"throughput_tokens_sec,omitempty"`
	AvgLatencyMs        *int     `json:"avg_latency_ms,omitempty"`
}

// CommunityFeedback contains user feedback
type CommunityFeedback struct {
	RedditSentiment float64  `json:"reddit_sentiment"`
	GithubStars     float64  `json:"github_stars"`
	UserRating      float64  `json:"user_rating"`
	Strengths       []string `json:"strengths"`
	Weaknesses      []string `json:"weaknesses"`
	BestUseCases    []string `json:"best_use_cases"`
}

// ComplexityRecommendations contains task complexity guidance
type ComplexityRecommendations struct {
	SimpleTasks     bool     `json:"simple_tasks"`
	MediumTasks     bool     `json:"medium_tasks"`
	ComplexTasks    bool     `json:"complex_tasks"`
	Specializations []string `json:"specializations"`
}

// TaskCapabilities contains task-specific capabilities
type TaskCapabilities struct {
	TextTasks       map[string]TaskCapability `json:"text_tasks,omitempty"`
	ImageTasks      map[string]TaskCapability `json:"image_tasks,omitempty"`
	VideoTasks      map[string]TaskCapability `json:"video_tasks,omitempty"`
	AudioTasks      map[string]TaskCapability `json:"audio_tasks,omitempty"`
	GenerativeTasks map[string]GenerativeCapability `json:"generative_tasks,omitempty"`
}

type TaskCapability struct {
	Score           float64  `json:"score"`
	Confidence      float64  `json:"confidence"`
	ComplexityRange []string `json:"complexity_range"`
}

type GenerativeCapability struct {
	Score         float64 `json:"score"`
	Confidence    float64 `json:"confidence"`
	MaxComplexity string  `json:"max_complexity"`
}

// ModelData represents the top-level structure in model_1.json
type ModelData struct {
	Models []EnhancedModel `json:"models"`
}

// EnhancedModelService manages enhanced models from model_1.json
type EnhancedModelService struct {
	modelPath string
	models    map[string]EnhancedModel
	mutex     sync.RWMutex
}

// NewEnhancedModelService creates a new enhanced model service
func NewEnhancedModelService(modelPath string) *EnhancedModelService {
	return &EnhancedModelService{
		modelPath: modelPath,
		models:    make(map[string]EnhancedModel),
	}
}

// LoadModels loads models from model_1.json
func (s *EnhancedModelService) LoadModels() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	log.Printf("[ENHANCED-MODEL] Loading models from: %s", s.modelPath)

	data, err := os.ReadFile(s.modelPath)
	if err != nil {
		return fmt.Errorf("failed to read model file: %w", err)
	}

	var modelData ModelData
	if err := json.Unmarshal(data, &modelData); err != nil {
		return fmt.Errorf("failed to parse model JSON: %w", err)
	}

	// Store models in map for quick lookup
	s.models = make(map[string]EnhancedModel, len(modelData.Models))
	for _, model := range modelData.Models {
		s.models[model.ID] = model

		// Normalize pricing structure - handle both old and new formats
		if model.Pricing.CostInPer1K != nil && model.Pricing.Text.CostInPer1K == nil {
			model.Pricing.Text.CostInPer1K = model.Pricing.CostInPer1K
		}
		if model.Pricing.CostOutPer1K != nil && model.Pricing.Text.CostOutPer1K == nil {
			model.Pricing.Text.CostOutPer1K = model.Pricing.CostOutPer1K
		}

		s.models[model.ID] = model
	}

	log.Printf("[ENHANCED-MODEL] Loaded %d models successfully", len(s.models))
	return nil
}

// GetAllModels returns all loaded models
func (s *EnhancedModelService) GetAllModels() []EnhancedModel {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	models := make([]EnhancedModel, 0, len(s.models))
	for _, model := range s.models {
		models = append(models, model)
	}
	return models
}

// GetModelByID retrieves a specific model by ID
func (s *EnhancedModelService) GetModelByID(id string) (EnhancedModel, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	model, exists := s.models[id]
	return model, exists
}

// GetModelsByProvider returns all models from a specific provider
func (s *EnhancedModelService) GetModelsByProvider(provider string) []EnhancedModel {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var filtered []EnhancedModel
	for _, model := range s.models {
		if model.Provider == provider {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

// GetModelsByType returns all models of a specific type (text, image, video, audio)
func (s *EnhancedModelService) GetModelsByType(modelType string) []EnhancedModel {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var filtered []EnhancedModel
	for _, model := range s.models {
		if model.ModelType == modelType {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

// GetOpenSourceModels returns all open source models
func (s *EnhancedModelService) GetOpenSourceModels() []EnhancedModel {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var filtered []EnhancedModel
	for _, model := range s.models {
		if model.OpenSource {
			filtered = append(filtered, model)
		}
	}
	return filtered
}
