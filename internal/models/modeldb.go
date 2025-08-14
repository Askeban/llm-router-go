package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
)

type ModelProfile struct {
	ID, Provider, DisplayName string
	ContextWindow             int
	CostInPer1K, CostOutPer1K float64
	AvgLatencyMs              int
	OpenSource                bool
	Tags                      []string
	Capabilities              map[string]float64
	Notes                     string
}
type Profiles struct{ db *sql.DB }

func NewProfiles(db *sql.DB, _ any) *Profiles { return &Profiles{db: db} }
func SeedFromJSON(db *sql.DB, path string) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS model_profiles(id TEXT PRIMARY KEY, json TEXT NOT NULL);`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS benchmark_metrics(model_id TEXT, source TEXT, metric TEXT, value REAL, unit TEXT, task TEXT, difficulty TEXT, ts INTEGER, PRIMARY KEY(model_id,source,metric));`)
	if err != nil {
		return err
	}
	var cnt int
	_ = db.QueryRow(`SELECT COUNT(1) FROM model_profiles`).Scan(&cnt)
	if cnt > 0 {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read profiles: %w", err)
	}
	var list []ModelProfile
	if err := json.Unmarshal(b, &list); err != nil {
		return fmt.Errorf("json: %w", err)
	}
	tx, _ := db.Begin()
	defer tx.Rollback()
	for _, m := range list {
		j, _ := json.Marshal(m)
		if _, err := tx.Exec(`INSERT INTO model_profiles(id,json) VALUES(?,?)`, m.ID, string(j)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (p *Profiles) ListModels(ctx context.Context) ([]ModelProfile, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT json FROM model_profiles`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ModelProfile
	for rows.Next() {
		var js string
		if err := rows.Scan(&js); err != nil {
			return nil, err
		}
		var m ModelProfile
		if err := json.Unmarshal([]byte(js), &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}
