package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type ModelProfile struct {
	ID            string             `json:"id"`
	Provider      string             `json:"provider"`
	DisplayName   string             `json:"display_name"`
	ContextWindow int                `json:"context_window"`
	CostInPer1K   float64            `json:"cost_in_per_1k"`
	CostOutPer1K  float64            `json:"cost_out_per_1k"`
	AvgLatencyMs  int                `json:"avg_latency_ms"`
	OpenSource    bool               `json:"open_source"`
	Tags          []string           `json:"tags"`
	Capabilities  map[string]float64 `json:"capabilities"`
	Notes         string             `json:"notes"`
}

type Profiles struct{ db *sql.DB }

func NewProfiles(db *sql.DB, _ any) *Profiles { return &Profiles{db: db} }

// Creates table (idempotent) and seeds from JSON (INSERT OR REPLACE).
func SeedFromJSON(db *sql.DB, path string) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS models(
  id TEXT PRIMARY KEY,
  provider TEXT,
  display_name TEXT,
  context_window INTEGER,
  cost_in_per_1k REAL,
  cost_out_per_1k REAL,
  avg_latency_ms INTEGER,
  open_source INTEGER,
  tags TEXT,
  capabilities TEXT,
  notes TEXT
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
		_, err = tx.Exec(`INSERT OR REPLACE INTO models
(id,provider,display_name,context_window,cost_in_per_1k,cost_out_per_1k,avg_latency_ms,open_source,tags,capabilities,notes)
VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			m.ID, m.Provider, m.DisplayName, m.ContextWindow, m.CostInPer1K, m.CostOutPer1K, m.AvgLatencyMs,
			boolToInt(m.OpenSource), string(jtags), string(jcaps), m.Notes)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *Profiles) ListModels(ctx context.Context) ([]ModelProfile, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT id,provider,display_name,context_window,cost_in_per_1k,cost_out_per_1k,avg_latency_ms,open_source,tags,capabilities,notes FROM models`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ModelProfile{}
	for rows.Next() {
		var m ModelProfile
		var jtags, jcaps string
		var open int
		if err := rows.Scan(&m.ID, &m.Provider, &m.DisplayName, &m.ContextWindow, &m.CostInPer1K, &m.CostOutPer1K, &m.AvgLatencyMs, &open, &jtags, &jcaps, &m.Notes); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(jtags), &m.Tags)
		_ = json.Unmarshal([]byte(jcaps), &m.Capabilities)
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

