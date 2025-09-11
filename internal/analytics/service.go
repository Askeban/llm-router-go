package analytics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type AnalyticsAPIResponse struct {
	Status        int           `json:"status"`
	PromptOptions PromptOptions `json:"prompt_options"`
	Data          []ModelData   `json:"data"`
}

type PromptOptions struct {
	ParallelQueries int `json:"parallel_queries"`
	PromptLength    int `json:"prompt_length"`
}

type ModelData struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	ReleaseDate string      `json:"release_date"`
	Creator     Creator     `json:"model_creator"`
	Evaluations Evaluations `json:"evaluations"`
	Pricing     Pricing     `json:"pricing"`

	// Performance metrics
	MedianOutputTokensPerSecond   float64 `json:"median_output_tokens_per_second"`
	MedianTimeToFirstTokenSeconds float64 `json:"median_time_to_first_token_seconds"`
	MedianTimeToFirstAnswerToken  float64 `json:"median_time_to_first_answer_token"`
}

type Creator struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Evaluations struct {
	// Main indexes
	ArtificialAnalysisIntelligenceIndex *float64 `json:"artificial_analysis_intelligence_index"`
	ArtificialAnalysisCodingIndex       *float64 `json:"artificial_analysis_coding_index"`
	ArtificialAnalysisMathIndex         *float64 `json:"artificial_analysis_math_index"`

	// Specific benchmarks
	MMLUPro           *float64 `json:"mmlu_pro"`
	GPQA              *float64 `json:"gpqa"`
	HLE               *float64 `json:"hle"`
	LiveCodeBench     *float64 `json:"livecodebench"`
	SciCode           *float64 `json:"scicode"`
	Math500           *float64 `json:"math_500"`
	AIME              *float64 `json:"aime"`
	AIME25            *float64 `json:"aime_25"`
	IFBench           *float64 `json:"ifbench"`
	LCR               *float64 `json:"lcr"`
	TerminalBenchHard *float64 `json:"terminalbench_hard"`
	Tau2              *float64 `json:"tau2"`
}

type Pricing struct {
	Price1MBlended3To1  float64 `json:"price_1m_blended_3_to_1"`
	Price1MInputTokens  float64 `json:"price_1m_input_tokens"`
	Price1MOutputTokens float64 `json:"price_1m_output_tokens"`
}

// Our internal model format
type Model struct {
	ID            string             `json:"id"`
	Provider      string             `json:"provider"`
	DisplayName   string             `json:"display_name"`
	ContextWindow int                `json:"context_window"`
	CostInPer1k   float64            `json:"cost_in_per_1k"`
	CostOutPer1k  float64            `json:"cost_out_per_1k"`
	AvgLatencyMS  *int               `json:"avg_latency_ms"`
	OpenSource    bool               `json:"open_source"`
	Tags          []string           `json:"tags"`
	Capabilities  map[string]float64 `json:"capabilities"`
	Notes         string             `json:"notes"`

	// Analytics AI metadata
	AnalyticsID     string    `json:"analytics_id,omitempty"`
	LastUpdated     time.Time `json:"last_updated,omitempty"`
	RealTimeMetrics bool      `json:"real_time_metrics,omitempty"`
}

type Service struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

func NewService() *Service {
	apiKey := os.Getenv("ANALYTICS_API_KEY")
	if apiKey == "" {
		apiKey = "aa_hvPVoBMuwefckQlniBWCrpQUmPdNSift" // fallback
	}

	return &Service{
		apiKey:  apiKey,
		baseURL: "https://artificialanalysis.ai/api/v2",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *Service) FetchModels() ([]ModelData, error) {
	return s.FetchModelsWithETag("")
}

func (s *Service) FetchModelsWithETag(etag string) ([]ModelData, error) {
	url := fmt.Sprintf("%s/data/llms/models", s.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Add ETag for conditional request
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		return nil, fmt.Errorf("data not modified (304)")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("analytics API error %d: %s", resp.StatusCode, string(body))
	}

	var apiResp AnalyticsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return apiResp.Data, nil
}

func (s *Service) GetResponseETag(etag string) (string, []ModelData, error) {
	url := fmt.Sprintf("%s/data/llms/models", s.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Add ETag for conditional request
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// Get ETag from response
	responseETag := resp.Header.Get("ETag")

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		return responseETag, nil, nil // No new data
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return responseETag, nil, fmt.Errorf("analytics API error %d: %s", resp.StatusCode, string(body))
	}

	var apiResp AnalyticsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return responseETag, nil, fmt.Errorf("decode response: %w", err)
	}

	return responseETag, apiResp.Data, nil
}

// ConvertToInternalModel converts Analytics AI model data to our internal format
func (s *Service) ConvertToInternalModel(data ModelData) Model {
	// Map creator slug to our provider format
	provider := mapProviderName(data.Creator.Slug)

	// Generate model ID in our format
	modelID := fmt.Sprintf("%s-%s", provider, data.Slug)

	// Calculate capabilities from evaluations
	capabilities := calculateCapabilities(data.Evaluations)

	// Determine context window (default estimates based on model)
	contextWindow := estimateContextWindow(data.Name, data.Slug)

	// Convert pricing from per-million to per-1k tokens
	costInPer1k := data.Pricing.Price1MInputTokens / 1000.0
	costOutPer1k := data.Pricing.Price1MOutputTokens / 1000.0

	// Calculate average latency in ms from time to first token
	var avgLatencyMS *int
	if data.MedianTimeToFirstTokenSeconds > 0 {
		latency := int(data.MedianTimeToFirstTokenSeconds * 1000)
		avgLatencyMS = &latency
	}

	// Determine tags
	tags := generateTags(data, capabilities)

	// Generate notes with benchmark highlights
	notes := generateNotes(data)

	return Model{
		ID:              modelID,
		Provider:        provider,
		DisplayName:     data.Name,
		ContextWindow:   contextWindow,
		CostInPer1k:     costInPer1k,
		CostOutPer1k:    costOutPer1k,
		AvgLatencyMS:    avgLatencyMS,
		OpenSource:      isOpenSource(data.Creator.Slug, data.Name),
		Tags:            tags,
		Capabilities:    capabilities,
		Notes:           notes,
		AnalyticsID:     data.ID,
		LastUpdated:     time.Now(),
		RealTimeMetrics: true,
	}
}

// Helper functions

func mapProviderName(creatorSlug string) string {
	mapping := map[string]string{
		"openai":     "openai",
		"anthropic":  "anthropic",
		"google":     "google",
		"meta":       "meta",
		"mistral-ai": "mistral",
		"xai":        "xai",
		"amazon":     "amazon",
		"alibaba":    "alibaba",
		"cohere":     "cohere",
		"ibm":        "ibm",
		"deepseek":   "deepseek",
	}

	if provider, exists := mapping[creatorSlug]; exists {
		return provider
	}
	return creatorSlug
}

func calculateCapabilities(eval Evaluations) map[string]float64 {
	capabilities := map[string]float64{
		"coding":    0.0,
		"math":      0.0,
		"reasoning": 0.0,
		"writing":   0.7, // default
		"support":   0.7, // default
	}

	// Use Analytics AI specific indexes if available
	if eval.ArtificialAnalysisCodingIndex != nil && *eval.ArtificialAnalysisCodingIndex > 0 {
		capabilities["coding"] = *eval.ArtificialAnalysisCodingIndex / 100.0
	} else if eval.LiveCodeBench != nil {
		capabilities["coding"] = *eval.LiveCodeBench
	}

	if eval.ArtificialAnalysisMathIndex != nil && *eval.ArtificialAnalysisMathIndex > 0 {
		capabilities["math"] = *eval.ArtificialAnalysisMathIndex / 100.0
	} else if eval.Math500 != nil {
		capabilities["math"] = *eval.Math500
	} else if eval.AIME != nil {
		capabilities["math"] = *eval.AIME
	}

	// Use intelligence index for reasoning, fallback to specific benchmarks
	if eval.ArtificialAnalysisIntelligenceIndex != nil && *eval.ArtificialAnalysisIntelligenceIndex > 0 {
		capabilities["reasoning"] = *eval.ArtificialAnalysisIntelligenceIndex / 100.0
	} else if eval.MMLUPro != nil {
		capabilities["reasoning"] = *eval.MMLUPro
	} else if eval.GPQA != nil {
		capabilities["reasoning"] = *eval.GPQA
	}

	return capabilities
}

func estimateContextWindow(name, slug string) int {
	// Context window estimates based on known models
	contextMap := map[string]int{
		"gpt-4o":           128000,
		"gpt-4":            128000,
		"gpt-5":            400000,
		"o3":               128000,
		"claude":           200000,
		"gemini-2.5-pro":   1000000,
		"gemini-2.5-flash": 1000000,
		"llama-4":          10000000, // Scout has 10M token context
		"llama-3":          128000,
	}

	// Check for specific patterns in name/slug
	for pattern, contextWindow := range contextMap {
		if contains(name, pattern) || contains(slug, pattern) {
			return contextWindow
		}
	}

	// Default context window
	return 128000
}

func contains(str, pattern string) bool {
	return len(str) >= len(pattern) &&
		(str[:len(pattern)] == pattern ||
			str[len(str)-len(pattern):] == pattern ||
			findSubstring(str, pattern))
}

func findSubstring(str, pattern string) bool {
	for i := 0; i <= len(str)-len(pattern); i++ {
		if str[i:i+len(pattern)] == pattern {
			return true
		}
	}
	return false
}

func isOpenSource(creatorSlug, modelName string) bool {
	openSourceCreators := map[string]bool{
		"meta":    true,
		"alibaba": true,
	}

	if openSourceCreators[creatorSlug] {
		return true
	}

	// Check for known open source model patterns
	openSourcePatterns := []string{"llama", "gemma", "mistral", "qwen"}
	for _, pattern := range openSourcePatterns {
		if contains(modelName, pattern) {
			return true
		}
	}

	return false
}

func generateTags(data ModelData, capabilities map[string]float64) []string {
	var tags []string

	// Open/closed source
	if isOpenSource(data.Creator.Slug, data.Name) {
		tags = append(tags, "open")
	} else {
		tags = append(tags, "closed")
	}

	// Capability-based tags
	if capabilities["coding"] > 0.6 {
		tags = append(tags, "coding")
	}
	if capabilities["reasoning"] > 0.7 {
		tags = append(tags, "reasoning")
	}
	if capabilities["math"] > 0.7 {
		tags = append(tags, "math")
	}

	// Cost-based tags
	if data.Pricing.Price1MInputTokens < 1.0 {
		tags = append(tags, "cheap")
	}

	// Model-specific tags
	if contains(data.Name, "vision") || contains(data.Name, "multimodal") {
		tags = append(tags, "multimodal")
	}
	if contains(data.Name, "thinking") {
		tags = append(tags, "reasoning")
	}
	if contains(data.Name, "agentic") {
		tags = append(tags, "agentic")
	}
	if contains(data.Name, "enterprise") {
		tags = append(tags, "enterprise")
	}

	return tags
}

func generateNotes(data ModelData) string {
	var benchmarks []string

	eval := data.Evaluations
	if eval.MMLUPro != nil && *eval.MMLUPro > 0 {
		benchmarks = append(benchmarks, fmt.Sprintf("MMLU Pro: %.1f%%", *eval.MMLUPro*100))
	}
	if eval.GPQA != nil && *eval.GPQA > 0 {
		benchmarks = append(benchmarks, fmt.Sprintf("GPQA: %.1f%%", *eval.GPQA*100))
	}
	if eval.LiveCodeBench != nil && *eval.LiveCodeBench > 0 {
		benchmarks = append(benchmarks, fmt.Sprintf("LiveCodeBench: %.1f%%", *eval.LiveCodeBench*100))
	}
	if eval.AIME25 != nil && *eval.AIME25 > 0 {
		benchmarks = append(benchmarks, fmt.Sprintf("AIME 2025: %.1f%%", *eval.AIME25*100))
	}

	baseNote := fmt.Sprintf("Real-time data from Analytics AI (Updated: %s)", time.Now().Format("2006-01-02"))

	if len(benchmarks) > 0 {
		benchmarkStr := ""
		for i, benchmark := range benchmarks {
			if i == 0 {
				benchmarkStr += benchmark
			} else {
				benchmarkStr += ", " + benchmark
			}
		}
		return fmt.Sprintf("Benchmark Highlights: %s. %s", benchmarkStr, baseNote)
	}

	return baseNote
}
