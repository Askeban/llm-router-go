package selection

import (
	"errors"
	"math"
	"sort"

	"llm-router-go/internal/classifier"
	"llm-router-go/internal/config"
	"llm-router-go/internal/policy"
	"llm-router-go/internal/scoring"
	"llm-router-go/internal/types"
)

type Preferences struct {
	Prioritize      string `json:"prioritize,omitempty"`
	AllowTruncation bool   `json:"allow_truncation,omitempty"`
}

type SelectRequest struct {
	Prompt      string         `json:"prompt"`
	Context     *types.Context `json:"context,omitempty"`
	Preferences *Preferences   `json:"preferences,omitempty"`
	Candidates  []string       `json:"candidates,omitempty"`
	TopK        int            `json:"top_k,omitempty"`
}

type RankedModel struct {
	ModelID               string   `json:"model_id"`
	Name                  string   `json:"name"`
	Score                 float64  `json:"score"`
	Percent               float64  `json:"percent"`
	Quality               float64  `json:"quality"`
	SuccessProb           float64  `json:"success_prob"`
	EstimatedCost         float64  `json:"estimated_cost"`
	ExpectedLatencyMs     int      `json:"expected_latency_ms"`
	Reasons               []string `json:"reasons"`
	EstimatedOutputTokens int      `json:"-"`
}

type RankResponse struct {
	PromptType            string             `json:"prompt_type"`
	ComplexityScore       float64            `json:"complexity_score"`
	SubScores             map[string]float64 `json:"sub_scores"`
	TokensIn              int                `json:"tokens_in"`
	EstimatedOutputTokens int                `json:"estimated_output_tokens"`
	Models                []RankedModel      `json:"models"`
}

func Selector(cfg *config.ConfigStore, req SelectRequest) (Result, error) {
	prefs := map[string]interface{}{}
	if req.Preferences != nil {
		if req.Preferences.Prioritize != "" {
			prefs["prioritize"] = req.Preferences.Prioritize
		}
		if req.Preferences.AllowTruncation {
			prefs["allow_truncation"] = req.Preferences.AllowTruncation
		}
	}
	models := cfg.GetModels()
	if len(req.Candidates) > 0 {
		cand := map[string]bool{}
		for _, c := range req.Candidates {
			cand[c] = true
		}
		filtered := make([]config.Model, 0, len(models))
		for _, m := range models {
			if cand[m.ID] {
				filtered = append(filtered, m)
			}
		}
		models = filtered
	}
	return SelectModel(req.Prompt, req.Context, models, prefs)
}

func Ranker(cfg *config.ConfigStore, req SelectRequest) (RankResponse, error) {
	prefs := map[string]interface{}{}
	prioritise := "quality"
	allowTrunc := false
	if req.Preferences != nil {
		if req.Preferences.Prioritize != "" {
			prioritise = req.Preferences.Prioritize
			prefs["prioritize"] = req.Preferences.Prioritize
		}
		if req.Preferences.AllowTruncation {
			allowTrunc = true
			prefs["allow_truncation"] = true
		}
	}
	models := cfg.GetModels()
	if len(req.Candidates) > 0 {
		cand := map[string]bool{}
		for _, c := range req.Candidates {
			cand[c] = true
		}
		filtered := make([]config.Model, 0, len(models))
		for _, m := range models {
			if cand[m.ID] {
				filtered = append(filtered, m)
			}
		}
		models = filtered
	}
	if len(models) == 0 {
		return RankResponse{}, errors.New("no models available")
	}
	weights := policy.DetermineWeights(prioritise)
	pt := classifier.Classify(req.Prompt, req.Context)
	complexity, subScores := scoring.ComputeComplexity(req.Prompt, req.Context)
	tokensIn := len(scoring.Tokenize(req.Prompt))
	if req.Context != nil {
		for _, f := range req.Context.Files {
			tokensIn += len(scoring.Tokenize(f.Content))
		}
	}
	ranked := make([]RankedModel, 0, len(models))
	for i := range models {
		m := &models[i]
		if !allowTrunc && tokensIn > m.ContextWindow {
			continue
		}
		util, quality, successProb, cost, _, estOut := policy.ComputeUtility(*m, pt, tokensIn, complexity, weights)
		ranked = append(ranked, RankedModel{
			ModelID:               m.ID,
			Name:                  m.Name,
			Score:                 util,
			Quality:               quality,
			SuccessProb:           successProb,
			EstimatedCost:         cost,
			ExpectedLatencyMs:     m.LatencyMs,
			Reasons:               []string{},
			EstimatedOutputTokens: estOut,
		})
	}
	if len(ranked) == 0 {
		return RankResponse{}, errors.New("no suitable model found for given prompt and context")
	}
	sumExp := 0.0
	for _, rm := range ranked {
		sumExp += math.Exp(rm.Score)
	}
	for i := range ranked {
		ranked[i].Percent = math.Exp(ranked[i].Score) / sumExp * 100
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].Percent > ranked[j].Percent })
	topK := req.TopK
	if topK <= 0 {
		topK = 5
	}
	if topK < len(ranked) {
		ranked = ranked[:topK]
	}
	resp := RankResponse{
		PromptType:            pt,
		ComplexityScore:       math.Round(complexity*1000) / 1000,
		SubScores:             subScores,
		TokensIn:              tokensIn,
		EstimatedOutputTokens: ranked[0].EstimatedOutputTokens,
		Models:                ranked,
	}
	return resp, nil
}
