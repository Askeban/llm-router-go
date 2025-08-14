package metrics

import (
	"context"
	"database/sql"
	"time"
)

type NormalizedMetric struct {
	ModelID    string  `json:"model_id"`
	Name       string  `json:"name"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
	Task       string  `json:"task,omitempty"`
	Difficulty string  `json:"difficulty,omitempty"`
}
type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }
func (s *Store) UpsertMetrics(ctx context.Context, source string, ms []NormalizedMetric) error {
	tx, _ := s.db.BeginTx(ctx, nil)
	defer tx.Rollback()
	now := time.Now().Unix()
	for _, m := range ms {
		if _, err := tx.ExecContext(ctx, `INSERT INTO benchmark_metrics(model_id,source,metric,value,unit,task,difficulty,ts) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(model_id,source,metric) DO UPDATE SET value=excluded.value, unit=excluded.unit, task=excluded.task, difficulty=excluded.difficulty, ts=excluded.ts`, m.ModelID, source, m.Name, m.Value, m.Unit, m.Task, m.Difficulty, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type MetricRow struct {
	ModelID, Source, Metric, Unit, Task, Difficulty string
	Value                                           float64
	TS                                              int64
}

func (s *Store) ListModels(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT model_id FROM benchmark_metrics ORDER BY model_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var m string
		_ = rows.Scan(&m)
		out = append(out, m)
	}
	return out, nil
}
func (s *Store) GetMetrics(ctx context.Context, model string) ([]MetricRow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT model_id,source,metric,value,unit,task,difficulty,ts FROM benchmark_metrics WHERE model_id=?`, model)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MetricRow
	for rows.Next() {
		var k MetricRow
		_ = rows.Scan(&k.ModelID, &k.Source, &k.Metric, &k.Value, &k.Unit, &k.Task, &k.Difficulty, &k.TS)
		out = append(out, k)
	}
	return out, nil
}
func (s *Store) GetAll(ctx context.Context) ([]MetricRow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT model_id,source,metric,value,unit,task,difficulty,ts FROM benchmark_metrics`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MetricRow
	for rows.Next() {
		var k MetricRow
		_ = rows.Scan(&k.ModelID, &k.Source, &k.Metric, &k.Value, &k.Unit, &k.Task, &k.Difficulty, &k.TS)
		out = append(out, k)
	}
	return out, nil
}
