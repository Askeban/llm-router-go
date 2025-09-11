package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Askeban/llm-router-go/internal/analytics"
)

// HybridModelService combines Analytics AI real-time data with static fallback
type HybridModelService struct {
	analyticsService *analytics.Service
	profiles         *Profiles
	staticPath       string

	// Cache management
	cache         []ModelProfile
	cacheExpiry   time.Time
	cacheDuration time.Duration
	mutex         sync.RWMutex

	// Advanced caching
	lastETag       string
	dailyRefreshAt time.Time

	// Metrics
	analyticsSuccessCount  int64
	analyticsFallbackCount int64
	staticFallbackCount    int64
}

func NewHybridModelService(db *sql.DB, staticPath string) *HybridModelService {
	service := &HybridModelService{
		analyticsService: analytics.NewService(),
		profiles:         NewProfiles(db, nil),
		staticPath:       staticPath,
		cacheDuration:    24 * time.Hour, // Cache for 24 hours
		cache:            []ModelProfile{},
	}

	// Schedule daily refresh at 2 AM
	service.scheduleDailyRefresh()

	return service
}

// GetModels returns models with Analytics AI data prioritized over static data
func (h *HybridModelService) GetModels(ctx context.Context) ([]ModelProfile, error) {
	h.mutex.RLock()

	// Check if daily refresh is needed
	now := time.Now()
	if now.After(h.dailyRefreshAt) && len(h.cache) > 0 {
		h.mutex.RUnlock()
		log.Printf("[HYBRID] Daily refresh time reached, refreshing cache...")
		return h.refreshModels(ctx)
	}

	// Return cached data if valid
	if len(h.cache) > 0 && now.Before(h.cacheExpiry) {
		defer h.mutex.RUnlock()
		log.Printf("[HYBRID] Returning cached models: %d models", len(h.cache))
		return h.cache, nil
	}
	h.mutex.RUnlock()

	// Cache expired or empty, refresh from Analytics AI
	return h.refreshModels(ctx)
}

// refreshModels fetches from Analytics AI and falls back to static data
func (h *HybridModelService) refreshModels(ctx context.Context) ([]ModelProfile, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	var finalModels []ModelProfile
	analyticsModels := make(map[string]ModelProfile) // keyed by model slug/name for deduplication

	// Step 1: Try to fetch from Analytics AI with ETag
	log.Printf("[HYBRID] Fetching models from Analytics AI (ETag: %s)...", h.lastETag)
	newETag, analyticsData, err := h.analyticsService.GetResponseETag(h.lastETag)
	if err != nil {
		log.Printf("[HYBRID] Analytics AI fetch failed: %v", err)
		h.analyticsFallbackCount++
	} else if analyticsData == nil {
		// 304 Not Modified - data hasn't changed
		log.Printf("[HYBRID] Analytics AI data not modified (304), using cached data")
		h.analyticsSuccessCount++
	} else {
		log.Printf("[HYBRID] Successfully fetched %d models from Analytics AI (ETag: %s)", len(analyticsData), newETag)
		h.analyticsSuccessCount++
		h.lastETag = newETag

		// Convert Analytics AI data to our internal format
		for _, data := range analyticsData {
			model := h.analyticsService.ConvertToInternalModel(data)

			// Use slug as key for deduplication
			analyticsModels[data.Slug] = h.convertToModelProfile(model)
		}
	}

	// Step 2: Load static models as fallback/supplement
	log.Printf("[HYBRID] Loading static models from %s...", h.staticPath)
	staticModels, err := h.loadStaticModels()
	if err != nil {
		log.Printf("[HYBRID] Failed to load static models: %v", err)
		h.staticFallbackCount++
	} else {
		log.Printf("[HYBRID] Loaded %d static models", len(staticModels))
	}

	// Step 3: Merge models (Analytics AI takes priority)
	modelMap := make(map[string]ModelProfile)

	// First add static models
	for _, model := range staticModels {
		modelMap[model.ID] = model
	}

	// Then overlay Analytics AI models (they take priority)
	for _, model := range analyticsModels {
		// Try to find matching static model by name similarity
		staticMatch := h.findStaticMatch(model, staticModels)
		if staticMatch != nil {
			// Merge: use Analytics AI data but keep static context window, provider info if missing
			merged := h.mergeModels(model, *staticMatch)
			modelMap[merged.ID] = merged
		} else {
			// Pure Analytics AI model
			modelMap[model.ID] = model
		}
	}

	// Convert map to slice
	for _, model := range modelMap {
		finalModels = append(finalModels, model)
	}

	log.Printf("[HYBRID] Final model count: %d (Analytics: %d, Static: %d)",
		len(finalModels), len(analyticsModels), len(staticModels))

	// Update cache
	h.cache = finalModels
	h.cacheExpiry = time.Now().Add(h.cacheDuration)

	// Schedule next daily refresh
	h.scheduleDailyRefresh()

	return finalModels, nil
}

// loadStaticModels loads models from the static JSON file
func (h *HybridModelService) loadStaticModels() ([]ModelProfile, error) {
	data, err := os.ReadFile(h.staticPath)
	if err != nil {
		return nil, fmt.Errorf("read static models file: %w", err)
	}

	var models []ModelProfile
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, fmt.Errorf("parse static models JSON: %w", err)
	}

	// Mark as static data
	for i := range models {
		models[i].DataProvenance = DataProvenance{
			StaticData: map[string]string{
				"all": time.Now().Format(time.RFC3339),
			},
			LastConsolidated: time.Now().Format(time.RFC3339),
			DataQuality:      0.8, // Static data quality
		}
	}

	return models, nil
}

// convertToModelProfile converts analytics.Model to ModelProfile
func (h *HybridModelService) convertToModelProfile(model analytics.Model) ModelProfile {
	profile := ModelProfile{
		ID:            model.ID,
		Provider:      model.Provider,
		DisplayName:   model.DisplayName,
		ContextWindow: model.ContextWindow,
		CostInPer1K:   model.CostInPer1k,
		CostOutPer1K:  model.CostOutPer1k,
		OpenSource:    model.OpenSource,
		Tags:          model.Tags,
		Capabilities:  model.Capabilities,
		Notes:         model.Notes,
	}

	if model.AvgLatencyMS != nil {
		profile.AvgLatencyMs = *model.AvgLatencyMS
	}

	// Mark as real-time data
	profile.DataProvenance = DataProvenance{
		APIData: map[string]string{
			"all": time.Now().Format(time.RFC3339),
		},
		LastConsolidated: time.Now().Format(time.RFC3339),
		DataQuality:      1.0, // Real-time data has highest quality
	}

	return profile
}

// findStaticMatch finds a static model that matches the Analytics AI model
func (h *HybridModelService) findStaticMatch(analyticsModel ModelProfile, staticModels []ModelProfile) *ModelProfile {
	// Try exact name match first
	for i := range staticModels {
		if staticModels[i].DisplayName == analyticsModel.DisplayName {
			return &staticModels[i]
		}
	}

	// Try provider + name similarity
	for i := range staticModels {
		if staticModels[i].Provider == analyticsModel.Provider {
			// Check for name similarity (contains key parts)
			if h.namesSimilar(staticModels[i].DisplayName, analyticsModel.DisplayName) {
				return &staticModels[i]
			}
		}
	}

	return nil
}

// namesSimilar checks if two model names are similar enough to be the same model
func (h *HybridModelService) namesSimilar(name1, name2 string) bool {
	// Simple similarity check - could be enhanced
	name1Lower := toLower(name1)
	name2Lower := toLower(name2)

	// Check if one name contains the other or they share key terms
	keyTerms := []string{"gpt", "claude", "gemini", "llama", "opus", "sonnet", "pro", "mini", "4o", "3.5"}

	matches := 0
	for _, term := range keyTerms {
		if contains(name1Lower, term) && contains(name2Lower, term) {
			matches++
		}
	}

	return matches > 0
}

// mergeModels merges Analytics AI model with static model data
func (h *HybridModelService) mergeModels(analyticsModel, staticModel ModelProfile) ModelProfile {
	merged := analyticsModel // Start with Analytics AI data

	// Use static data for fields that might be missing or inaccurate in Analytics AI
	if merged.ContextWindow == 0 && staticModel.ContextWindow > 0 {
		merged.ContextWindow = staticModel.ContextWindow
	}

	// Use static provider info if more accurate
	if len(staticModel.Provider) > 0 && merged.Provider != staticModel.Provider {
		// Keep Analytics AI provider, but note the difference
		merged.Notes += fmt.Sprintf(" (Static provider: %s)", staticModel.Provider)
	}

	// Merge tags (union of both sets)
	tagSet := make(map[string]bool)
	for _, tag := range merged.Tags {
		tagSet[tag] = true
	}
	for _, tag := range staticModel.Tags {
		tagSet[tag] = true
	}

	merged.Tags = make([]string, 0, len(tagSet))
	for tag := range tagSet {
		merged.Tags = append(merged.Tags, tag)
	}

	// Merge capabilities (Analytics AI takes priority, but keep static as backup)
	if merged.Capabilities == nil {
		merged.Capabilities = make(map[string]float64)
	}
	for capability, score := range staticModel.Capabilities {
		if _, exists := merged.Capabilities[capability]; !exists && score > 0 {
			merged.Capabilities[capability] = score
		}
	}

	// Update data provenance to show merge
	merged.DataProvenance = DataProvenance{
		APIData: map[string]string{
			"primary": time.Now().Format(time.RFC3339),
		},
		StaticData: map[string]string{
			"supplementary": time.Now().Format(time.RFC3339),
		},
		LastConsolidated: time.Now().Format(time.RFC3339),
		DataQuality:      0.95, // High quality merged data
	}

	return merged
}

// GetModelByID returns a specific model by ID
func (h *HybridModelService) GetModelByID(ctx context.Context, id string) (*ModelProfile, error) {
	models, err := h.GetModels(ctx)
	if err != nil {
		return nil, err
	}

	for i := range models {
		if models[i].ID == id {
			return &models[i], nil
		}
	}

	return nil, fmt.Errorf("model not found: %s", id)
}

// RefreshCache forces a cache refresh
func (h *HybridModelService) RefreshCache(ctx context.Context) error {
	h.mutex.Lock()
	h.cacheExpiry = time.Time{} // Expire cache
	h.mutex.Unlock()

	_, err := h.GetModels(ctx)
	return err
}

// GetMetrics returns service metrics
func (h *HybridModelService) GetMetrics() map[string]interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return map[string]interface{}{
		"analytics_success_count":  h.analyticsSuccessCount,
		"analytics_fallback_count": h.analyticsFallbackCount,
		"static_fallback_count":    h.staticFallbackCount,
		"cache_size":               len(h.cache),
		"cache_expiry":             h.cacheExpiry.Format(time.RFC3339),
		"cache_valid":              time.Now().Before(h.cacheExpiry),
	}
}

// Helper functions

func toLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findInString(s, substr)
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// scheduleDailyRefresh sets up the next daily refresh time at 2 AM
func (h *HybridModelService) scheduleDailyRefresh() {
	now := time.Now()

	// Set refresh time to 2 AM tomorrow
	tomorrow := now.AddDate(0, 0, 1)
	refreshTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 2, 0, 0, 0, now.Location())

	// If it's already past 2 AM today, set for 2 AM tomorrow
	if now.Hour() >= 2 {
		h.dailyRefreshAt = refreshTime
	} else {
		// It's before 2 AM today, refresh at 2 AM today
		todayRefresh := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
		h.dailyRefreshAt = todayRefresh
	}

	log.Printf("[HYBRID] Next daily refresh scheduled for: %s", h.dailyRefreshAt.Format(time.RFC3339))
}

// ForceRefresh forces an immediate cache refresh (on-demand)
func (h *HybridModelService) ForceRefresh(ctx context.Context) error {
	log.Printf("[HYBRID] Force refresh requested")
	h.mutex.Lock()
	h.cacheExpiry = time.Time{} // Expire cache immediately
	h.mutex.Unlock()

	_, err := h.refreshModels(ctx)
	return err
}
