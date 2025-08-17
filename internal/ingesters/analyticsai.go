package ingesters

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Askeban/llm-router-go/internal/metrics"
	"github.com/Askeban/llm-router-go/internal/models"
)

// --- wire format from Analytics AI (subset we actually use) ---

type aaModel struct {
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Evals struct {
		Intel float64 `json:"artificial_analysis_intelligence_index"`
		Code  float64 `json:"artificial_analysis_coding_index"`
		Math  float64 `json:"artificial_analysis_math_index"`
		MMLU  float64 `json:"mmlu_pro"`
		GPQA  float64 `json:"gpqa"`
		LCB   float64 `json:"livecodebench"`
	} `json:"evaluations"`
	Pricing struct {
		BlendedPer1M float64 `json:"price_1m_blended_3_to_1"`
	} `json:"pricing"`
	MedianTPS float64 `json:"median_output_tokens_per_second"`
	TTFTSec   float64 `json:"median_time_to_first_token_seconds"`
}

type aaResp struct {
	Status int       `json:"status"`
	Data   []aaModel `json:"data"`
}

func fetchAA(ctx context.Context, apiKey string) ([]aaModel, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("analytics ai: missing API key (env ANALYTICS_AI_KEY)")
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://artificialanalysis.ai/api/v2/data/llms/models", nil)
	req.Header.Set("x-api-key", apiKey)
	httpc := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("analytics ai status %d: %s", resp.StatusCode, string(b))
	}
	var out aaResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func canonical(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "openai", "")
	s = strings.ReplaceAll(s, "anthropic", "")
	s = strings.ReplaceAll(s, "google", "")
	s = strings.ReplaceAll(s, "gemini", "")
	s = strings.ReplaceAll(s, "meta", "")
	re := regexp.MustCompile(`[^a-z0-9]+`)
	return re.ReplaceAllString(s, "")
}

// SyncAnalyticsAI pulls live metrics and stores them in the metrics table,
// and (when model name matches) updates cost/latency & baseline capabilities.
func SyncAnalyticsAI(ctx context.Context, apiKey string, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("nil db")
	}
	rows, err := fetchAA(ctx, apiKey)
	if err != nil {
		return err
	}

	profiles := models.NewProfiles(db, nil)
	all, err := profiles.ListModels(ctx)
	if err != nil {
		return err
	}

	// Build local index by canonical DisplayName so we can map AnalyticsAI names.
	local := map[string]models.ModelProfile{}
	for _, m := range all {
		local[canonical(m.DisplayName)] = m
	}

	store := metrics.NewStore(db)

	for _, r := range rows {
		// Try to match by canonicalized name/slug
		var mp models.ModelProfile
		var ok bool
		if mp, ok = local[canonical(r.Name)]; !ok {
			if mp, ok = local[canonical(r.Slug)]; !ok {
				// Give up if we can't tie this AA row to a known model ID in your DB.
				continue
			}
		}

		// -------- 1) Write normalized metrics (source passed to UpsertMetrics) --------
		var batch []metrics.NormalizedMetric

		if r.Evals.Intel > 0 {
			batch = append(batch, metrics.NormalizedMetric{
				ModelID:    mp.ID,
				Name:       "artificial_analysis_intelligence_index",
				Unit:       "index_0_100",
				Value:      r.Evals.Intel,
				Task:       "",
				Difficulty: "",
			})
		}
		if r.Evals.Code > 0 {
			batch = append(batch, metrics.NormalizedMetric{
				ModelID: mp.ID,
				Name:    "artificial_analysis_coding_index",
				Unit:    "index_0_100",
				Value:   r.Evals.Code,
			})
		}
		if r.Evals.Math > 0 {
			batch = append(batch, metrics.NormalizedMetric{
				ModelID: mp.ID,
				Name:    "artificial_analysis_math_index",
				Unit:    "index_0_100",
				Value:   r.Evals.Math,
			})
		}
		if r.Evals.MMLU > 0 {
			batch = append(batch, metrics.NormalizedMetric{
				ModelID: mp.ID, Name: "mmlu_pro", Unit: "pct", Value: r.Evals.MMLU,
			})
		}
		if r.Evals.GPQA > 0 {
			batch = append(batch, metrics.NormalizedMetric{
				ModelID: mp.ID, Name: "gpqa", Unit: "pct", Value: r.Evals.GPQA,
			})
		}
		if r.Evals.LCB > 0 {
			batch = append(batch, metrics.NormalizedMetric{
				ModelID: mp.ID, Name: "livecodebench", Unit: "score", Value: r.Evals.LCB,
			})
		}
		if len(batch) > 0 {
			if err := store.UpsertMetrics(ctx, "analytics_ai", batch); err != nil {
				return fmt.Errorf("upsert metrics: %w", err)
			}
		}

		// -------- 2) Update model costs & latency in the models table --------
		// Convert blended price ($/1M at 3:1) -> $/1K input (approx)
		var costPer1K float64
		if r.Pricing.BlendedPer1M > 0 {
			costPer1K = r.Pricing.BlendedPer1M / 1000.0
		}
		avgLatencyMs := int(r.TTFTSec * 1000.0) // TTFT as proxy

		_ = profiles.UpdateCostLatency(ctx, mp.ID, costPer1K, avgLatencyMs)

		// -------- 3) Update baseline capabilities from AA indices (normalize 0..1) --------
		caps := map[string]float64{}
		if r.Evals.Intel > 0 {
			caps["reasoning"] = r.Evals.Intel / 100.0
		}
		if r.Evals.Code > 0 {
			caps["code"] = r.Evals.Code / 100.0
		}
		if r.Evals.Math > 0 {
			caps["math"] = r.Evals.Math / 100.0
		}
		if len(caps) > 0 {
			_ = profiles.UpdateCapabilities(ctx, mp.ID, caps)
		}
	}

	return nil
}
