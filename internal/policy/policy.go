package policy

import (
    "strings"

    "llm-router-go/internal/config"
    "llm-router-go/internal/scoring"
)

// Weights holds alpha and beta coefficients controlling the cost and latency
// penalties in the utility function.  Higher alpha penalises cost more and
// higher beta penalises latency more.
type Weights struct {
    Alpha float64
    Beta  float64
}

// determineWeights returns weights based on a prioritisation preference.  The
// preference string may be "quality", "cost" or "latency" (case-insensitive).
// Unknown preferences default to quality.
// DetermineWeights returns weights based on a prioritisation preference.
// The preference string may be "quality", "cost" or "latency" (case-insensitive).
// Unknown preferences default to quality.  It is exported so that callers
// outside this package can obtain the weights directly.
func DetermineWeights(preference string) Weights {
    switch strings.ToLower(preference) {
    case "cost":
        return Weights{Alpha: 0.6, Beta: 0.2}
    case "latency":
        return Weights{Alpha: 0.4, Beta: 0.5}
    case "quality":
        fallthrough
    default:
        return Weights{Alpha: 0.3, Beta: 0.2}
    }
}

// ComputeUtility computes the expected utility of a model for a given prompt.
// It returns the utility value along with quality, success probability, cost,
// latency penalty and estimated output tokens.  The baseline latency for
// penalty calculation is fixed at 500 ms.
func ComputeUtility(
    model config.Model,
    promptType string,
    tokensIn int,
    complexity float64,
    weights Weights,
) (utility float64, quality float64, successProb float64, cost float64, latencyPenalty float64, estOutputTokens int) {
    // Determine quality based on model capabilities
    if model.Capabilities != nil {
        if v, ok := model.Capabilities[promptType]; ok {
            quality = v
        } else if len(model.Capabilities) > 0 {
            sum := 0.0
            for _, q := range model.Capabilities {
                sum += q
            }
            quality = sum / float64(len(model.Capabilities))
        } else {
            quality = 0.5
        }
    } else {
        quality = 0.5
    }
    successProb = quality
    // Estimate output tokens and cost
    estOutputTokens = scoring.EstimateOutputTokens(tokensIn, complexity)
    cost = float64(tokensIn)*model.CostInput + float64(estOutputTokens)*model.CostOutput
    // Latency penalty relative to baseline 500 ms
    baseline := 500.0
    if model.LatencyMs > 0 {
        latencyPenalty = (float64(model.LatencyMs) - baseline) / baseline
        if latencyPenalty < 0 {
            latencyPenalty = 0
        }
    } else {
        latencyPenalty = 0.0
    }
    utility = quality*successProb - weights.Alpha*cost - weights.Beta*latencyPenalty
    return
}