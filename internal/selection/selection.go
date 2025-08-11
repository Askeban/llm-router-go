package selection

import (
	"errors"
	"math"

	"llm-router-go/internal/classifier"
	"llm-router-go/internal/config"
	"llm-router-go/internal/policy"
	"llm-router-go/internal/scoring"
	"llm-router-go/internal/types"
)

// Result contains the selected model and explanation details.
type Result struct {
	RecommendedModel string                 `json:"recommended_model"`
	ModelName        string                 `json:"model_name"`
	Confidence       float64                `json:"confidence"`
	Explanation      map[string]interface{} `json:"explanation"`
}

// SelectModel chooses the best model from the registry for the given prompt.
// It uses the classifier, scoring and policy packages to compute the
// expected utility for each model.  The preferences map may contain a
// "prioritize" key (quality, cost or latency) and "allow_truncation"
// (bool).  If allow_truncation is false and the prompt/context tokens exceed
// a model’s context window, that model is skipped.  If no model is suitable,
// an error is returned.
func SelectModel(prompt string, ctx *types.Context, models []config.Model, preferences map[string]interface{}) (Result, error) {
	// Determine preference values
	prioritise := "quality"
	allowTrunc := false
	if preferences != nil {
		if v, ok := preferences["prioritize"].(string); ok {
			prioritise = v
		}
		if v, ok := preferences["allow_truncation"].(bool); ok {
			allowTrunc = v
		}
	}
	weights := policy.DetermineWeights(prioritise)
	// Classify prompt type and compute complexity
	pt := classifier.Classify(prompt, ctx)
	complexity, subScores := scoring.ComputeComplexity(prompt, ctx)
	// Compute input tokens
	tokensIn := len(scoring.Tokenize(prompt))
	if ctx != nil {
		for _, f := range ctx.Files {
			tokensIn += len(scoring.Tokenize(f.Content))
		}
	}
	var bestModel *config.Model
	var bestUtility float64 = -math.MaxFloat64
	var bestMetadata map[string]interface{}
	for i := range models {
		m := &models[i]
		// Skip models that cannot fit the input tokens if truncation not allowed
		if !allowTrunc && tokensIn > m.ContextWindow {
			continue
		}
		util, quality, successProb, cost, latencyPen, estOut := policy.ComputeUtility(*m, pt, tokensIn, complexity, weights)
		if util > bestUtility {
			bestUtility = util
			bestModel = m
			bestMetadata = map[string]interface{}{
				"utility":         util,
				"quality":         quality,
				"success_prob":    successProb,
				"cost_estimate":   cost,
				"latency_penalty": latencyPen,
				"output_tokens":   estOut,
			}
		}
	}
	if bestModel == nil {
		return Result{}, errors.New("no suitable model found for given prompt and context")
	}
	// Compute confidence as product of quality and success probability
	quality := bestMetadata["quality"].(float64)
	successProb := bestMetadata["success_prob"].(float64)
	confidence := quality * successProb
	if confidence > 1.0 {
		confidence = 1.0
	}
	res := Result{
		RecommendedModel: bestModel.ID,
		ModelName:        bestModel.Name,
		Confidence:       math.Round(confidence*1000) / 1000,
		Explanation: map[string]interface{}{
			"prompt_type":             pt,
			"complexity_score":        math.Round(complexity*1000) / 1000,
			"sub_scores":              subScores,
			"estimated_cost":          bestMetadata["cost_estimate"],
			"expected_latency_ms":     bestModel.LatencyMs,
			"tokens_in":               tokensIn,
			"estimated_output_tokens": bestMetadata["output_tokens"],
		},
	}
	return res, nil
}
