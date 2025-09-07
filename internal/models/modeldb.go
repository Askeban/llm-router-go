package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Enhanced capability score with metadata
type CapabilityScore struct {
	Score       float64  `json:"score"`                // 0.0 to 1.0
	Confidence  float64  `json:"confidence"`           // Confidence in this score
	Benchmarks  []string `json:"benchmarks,omitempty"` // Supporting benchmarks
	LastUpdate  string   `json:"last_update"`          // ISO timestamp
	LastUpdated *string  `json:"last_updated"`         // Pointer for compatibility
}

// Performance profile for different complexity levels
type ComplexityProfile struct {
	Score                 float64 `json:"score"`                   // Main score (0-1)
	Confidence           float64 `json:"confidence"`              // Confidence (0-1)
	PerformanceMultiplier float64 `json:"performance_multiplier"` // Adjustment factor
	SpeedBoost            float64 `json:"speed_boost"`            // Relative speed
	QualityScore          float64 `json:"quality_score"`          // Output quality
}

// Routing metadata for intelligent model selection  
type RoutingMetadata struct {
	UsageCount           int      `json:"usage_count"`
	SuccessRate          float64  `json:"success_rate"`
	AverageRating        float64  `json:"average_rating"`
	BestCreativityScore  float64  `json:"best_creativity_score"`
	HandlesUrgency       bool     `json:"handles_urgency"`
	InteractionStyles    []string `json:"interaction_styles"`
	MaxOutputTokens      int      `json:"max_output_tokens"`
	Specializations      []string `json:"specializations"`
	PreferredUseCase     string   `json:"preferred_use_case"`
	AvoidedScenarios     []string `json:"avoided_scenarios"`
}

// Data source tracking for quality assurance
type DataProvenance struct {
	StaticData   map[string]string `json:"static_data"`   // Field -> timestamp
	ScrapedData  map[string]string `json:"scraped_data"`  // Field -> timestamp  
	APIData      map[string]string `json:"api_data"`      // Field -> timestamp
	LastConsolidated string        `json:"last_consolidated"`
	DataQuality      float64       `json:"data_quality"` // 0.0 to 1.0
}

// Enhanced ModelProfile with classifier integration
type ModelProfile struct {
	// Core identification
	ID            string `json:"id"`
	Provider      string `json:"provider"`
	DisplayName   string `json:"display_name"`
	APIAlias      string `json:"api_alias,omitempty"`      // API endpoint name
	ReleaseDate   string `json:"release_date,omitempty"`   // Release date
	
	// Technical specifications
	ContextWindow       int     `json:"context_window"`
	MaxOutputTokens     int     `json:"max_output_tokens,omitempty"`
	CostInPer1K         float64 `json:"cost_in_per_1k"`
	CostOutPer1K        float64 `json:"cost_out_per_1k"`
	CachedCostInPer1K   float64 `json:"cached_cost_in_per_1k,omitempty"`
	AvgLatencyMs        int     `json:"avg_latency_ms"`
	OpenSource          bool    `json:"open_source"`
	
	// Enhanced capabilities (classifier-aligned)
	EnhancedCapabilities map[string]CapabilityScore `json:"enhanced_capabilities,omitempty"`
	ComplexityProfiles   map[string]ComplexityProfile `json:"complexity_profiles,omitempty"`
	BenchmarkScores      map[string]float64 `json:"benchmark_scores,omitempty"`
	
	// Routing intelligence
	RoutingMetadata *RoutingMetadata `json:"routing_metadata,omitempty"`
	
	// Legacy compatibility
	Capabilities map[string]float64 `json:"capabilities"` // Keep for backward compatibility
	BestAt       []string           `json:"best_at"`      // Legacy field for compatibility
	Tags         []string           `json:"tags"`
	Notes        string             `json:"notes"`
	
	// Data quality and provenance
	DataProvenance DataProvenance `json:"data_provenance"`
}

type Profiles struct{ db *sql.DB }

func NewProfiles(db *sql.DB, _ any) *Profiles { return &Profiles{db: db} }

// Creates enhanced table (idempotent) and seeds from JSON (INSERT OR REPLACE).
func SeedFromJSON(db *sql.DB, path string) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS models(
  id TEXT PRIMARY KEY,
  provider TEXT,
  display_name TEXT,
  api_alias TEXT,
  release_date TEXT,
  context_window INTEGER,
  max_output_tokens INTEGER,
  cost_in_per_1k REAL,
  cost_out_per_1k REAL,
  cached_cost_in_per_1k REAL,
  avg_latency_ms INTEGER,
  open_source INTEGER,
  enhanced_capabilities TEXT,
  complexity_profiles TEXT,
  routing_metadata TEXT,
  tags TEXT,
  capabilities TEXT,
  notes TEXT,
  data_provenance TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var arr []ModelProfile
	if err := json.Unmarshal(b, &arr); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, m := range arr {
		jtags, _ := json.Marshal(m.Tags)
		jcaps, _ := json.Marshal(m.Capabilities)
		jenhcaps, _ := json.Marshal(m.EnhancedCapabilities)
		jcomplex, _ := json.Marshal(m.ComplexityProfiles)
		jrouting, _ := json.Marshal(m.RoutingMetadata)
		jprov, _ := json.Marshal(m.DataProvenance)
		
		_, err = tx.Exec(`INSERT OR REPLACE INTO models
(id,provider,display_name,api_alias,release_date,context_window,max_output_tokens,cost_in_per_1k,cost_out_per_1k,cached_cost_in_per_1k,avg_latency_ms,open_source,enhanced_capabilities,complexity_profiles,routing_metadata,tags,capabilities,notes,data_provenance)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			m.ID, m.Provider, m.DisplayName, m.APIAlias, m.ReleaseDate, m.ContextWindow, m.MaxOutputTokens, 
			m.CostInPer1K, m.CostOutPer1K, m.CachedCostInPer1K, m.AvgLatencyMs, boolToInt(m.OpenSource), 
			string(jenhcaps), string(jcomplex), string(jrouting), string(jtags), string(jcaps), m.Notes, string(jprov))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *Profiles) ListModels(ctx context.Context) ([]ModelProfile, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT id,provider,display_name,api_alias,release_date,context_window,max_output_tokens,cost_in_per_1k,cost_out_per_1k,cached_cost_in_per_1k,avg_latency_ms,open_source,enhanced_capabilities,complexity_profiles,routing_metadata,tags,capabilities,notes,data_provenance FROM models`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ModelProfile{}
	for rows.Next() {
		var m ModelProfile
		var jtags, jcaps, jenhcaps, jcomplex, jrouting, jprov string
		var open int
		var apiAlias, releaseDate sql.NullString
		var maxOutput, cachedCost sql.NullFloat64
		
		if err := rows.Scan(&m.ID, &m.Provider, &m.DisplayName, &apiAlias, &releaseDate, 
			&m.ContextWindow, &maxOutput, &m.CostInPer1K, &m.CostOutPer1K, &cachedCost, 
			&m.AvgLatencyMs, &open, &jenhcaps, &jcomplex, &jrouting, &jtags, &jcaps, &m.Notes, &jprov); err != nil {
			return nil, err
		}
		
		// Handle nullable fields
		if apiAlias.Valid {
			m.APIAlias = apiAlias.String
		}
		if releaseDate.Valid {
			m.ReleaseDate = releaseDate.String
		}
		if maxOutput.Valid {
			m.MaxOutputTokens = int(maxOutput.Float64)
		}
		if cachedCost.Valid {
			m.CachedCostInPer1K = cachedCost.Float64
		}
		
		// Unmarshal JSON fields
		_ = json.Unmarshal([]byte(jtags), &m.Tags)
		_ = json.Unmarshal([]byte(jcaps), &m.Capabilities)
		_ = json.Unmarshal([]byte(jenhcaps), &m.EnhancedCapabilities)
		_ = json.Unmarshal([]byte(jcomplex), &m.ComplexityProfiles)
		_ = json.Unmarshal([]byte(jrouting), &m.RoutingMetadata)
		_ = json.Unmarshal([]byte(jprov), &m.DataProvenance)
		
		m.OpenSource = open == 1
		out = append(out, m)
	}
	return out, nil
}

// UpdateCostLatency safely updates cost_in_per_1k and avg_latency_ms when new values are > 0.
func (p *Profiles) UpdateCostLatency(ctx context.Context, id string, costInPer1K float64, avgLatencyMs int) error {
	if id == "" {
		return fmt.Errorf("empty id")
	}
	_, err := p.db.ExecContext(ctx, `
UPDATE models
   SET cost_in_per_1k = CASE WHEN ? > 0 THEN ? ELSE cost_in_per_1k END,
       avg_latency_ms = CASE WHEN ? > 0 THEN ? ELSE avg_latency_ms END
 WHERE id = ?`, costInPer1K, costInPer1K, avgLatencyMs, avgLatencyMs, id)
	return err
}

// UpdateCapabilities merges new capability scores (lowercased keys) into the JSON map.
func (p *Profiles) UpdateCapabilities(ctx context.Context, id string, caps map[string]float64) error {
	if id == "" || len(caps) == 0 {
		return nil
	}
	var jcaps string
	if err := p.db.QueryRowContext(ctx, `SELECT capabilities FROM models WHERE id = ?`, id).Scan(&jcaps); err != nil {
		return err
	}
	existing := map[string]float64{}
	_ = json.Unmarshal([]byte(jcaps), &existing)
	for k, v := range caps {
		k = strings.ToLower(strings.TrimSpace(k))
		if v > 0 {
			existing[k] = v
		}
	}
	newj, _ := json.Marshal(existing)
	_, err := p.db.ExecContext(ctx, `UPDATE models SET capabilities = ? WHERE id = ?`, string(newj), id)
	return err
}

func boolToInt(b bool) int { if b { return 1 }; return 0 }

