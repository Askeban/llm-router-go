package metrics

import (
	"context"
	"fmt"
	"strings"
)

// Reuse your existing MetricRow type and Store from internal/metrics/store.go

// helper to DRY the row scan
func (s *Store) queryMetrics(ctx context.Context, query string, args ...any) ([]MetricRow, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MetricRow
	for rows.Next() {
		var r MetricRow
		if err := rows.Scan(&r.ModelID, &r.Source, &r.Metric, &r.Value, &r.Unit, &r.Task, &r.Difficulty, &r.TS); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Exact (case-sensitive) → already have GetMetrics, keep it.
// Case-insensitive exact match
func (s *Store) GetMetricsByModelLoose(ctx context.Context, model string) ([]MetricRow, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return []MetricRow{}, nil
	}

	// 1) Try exact (your existing method)
	if rows, err := s.GetMetrics(ctx, model); err == nil && len(rows) > 0 {
		return rows, nil
	}

	// 2) Try case-insensitive exact
	const q = `
SELECT model_id, source, metric, value, unit, IFNULL(task,''), IFNULL(difficulty,''), ts
FROM benchmark_metrics
WHERE LOWER(model_id) = LOWER(?)
ORDER BY ts DESC`
	return s.queryMetrics(ctx, q, model)
}

// Fuzzy fallback: contains/space-insensitive LIKE matches.
// Useful for queries like "GPT-4o mini" when DB has "openai-gpt-4o-mini".
func (s *Store) GetMetricsByModelFuzzy(ctx context.Context, slug string, limit int) ([]MetricRow, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return []MetricRow{}, nil
	}
	// Build a few liberal patterns
	p1 := "%" + slug + "%"                                     // %GPT-4o mini%
	p2 := "%" + strings.ReplaceAll(slug, " ", "%") + "%"        // %GPT-4o%mini%
	p3 := "%" + strings.ReplaceAll(strings.ToLower(slug), " ", "-") + "%" // %gpt-4o-mini%
	const q = `
SELECT model_id, source, metric, value, unit, IFNULL(task,''), IFNULL(difficulty,''), ts
FROM benchmark_metrics
WHERE LOWER(model_id) = LOWER(?)
   OR LOWER(model_id) LIKE LOWER(?)
   OR LOWER(model_id) LIKE LOWER(?)
   OR LOWER(model_id) LIKE LOWER(?)
ORDER BY ts DESC
LIMIT ?`

	return s.queryMetrics(ctx, q, slug, p1, p2, p3, limit)
}

// Handy list endpoint with optional source filter
func (s *Store) ListModelsBySource(ctx context.Context, source string, limit int) ([]string, error) {
	where := ""
	args := []any{}
	if strings.TrimSpace(source) != "" {
		where = "WHERE source = ?"
		args = append(args, source)
	}
	if limit <= 0 {
		limit = 1000
	}
	args = append(args, limit)

	q := fmt.Sprintf(`
SELECT DISTINCT model_id
FROM benchmark_metrics
%s
ORDER BY model_id
LIMIT ?`, where)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

