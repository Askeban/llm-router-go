package selection

import (
	"testing"

	"llm-router-go/internal/config"
	"llm-router-go/internal/types"
)

// sampleModels returns a small model registry for testing.
func sampleModels() []config.Model {
	return []config.Model{
		{
			ID:   "modelA",
			Name: "Model A",
			Capabilities: map[string]float64{
				"code": 0.8,
				"text": 0.7,
			},
			CostInput:     0.001,
			CostOutput:    0.002,
			ContextWindow: 1000,
			LatencyMs:     200,
		},
		{
			ID:   "modelB",
			Name: "Model B",
			Capabilities: map[string]float64{
				"code": 0.6,
				"text": 0.9,
			},
			CostInput:     0.0005,
			CostOutput:    0.0005,
			ContextWindow: 500,
			LatencyMs:     400,
		},
	}
}

func TestSelectModelReturnsResult(t *testing.T) {
	models := sampleModels()
	prompt := "Refactor the following function to improve readability"
	ctx := &types.Context{Files: []types.File{{Path: "code.py", Content: "def foo(x): return x+1"}}}
	res, err := SelectModel(prompt, ctx, models, map[string]interface{}{"prioritize": "quality"})
	if err != nil {
		t.Fatalf("unexpected error selecting model: %v", err)
	}
	if res.RecommendedModel == "" {
		t.Fatalf("expected a recommended model, got empty string")
	}
	if res.Confidence < 0 || res.Confidence > 1 {
		t.Fatalf("confidence out of range: %v", res.Confidence)
	}
	if res.Explanation["prompt_type"] == nil {
		t.Fatalf("missing prompt_type in explanation")
	}
}

// TestSelectModelChoosesHighestUtility ensures that the model with the highest
// expected utility is selected.  With the sample models, modelA has higher
// utility for code-heavy prompts when quality is prioritised, so it should be
// recommended.
func TestSelectModelChoosesHighestUtility(t *testing.T) {
	models := sampleModels()
	prompt := "Refactor the following function to improve readability"
	ctx := &types.Context{Files: []types.File{{Path: "code.py", Content: "def foo(x): return x+1"}}}
	res, err := SelectModel(prompt, ctx, models, map[string]interface{}{"prioritize": "quality"})
	if err != nil {
		t.Fatalf("unexpected error selecting model: %v", err)
	}
	if res.RecommendedModel != "modelA" {
		t.Fatalf("expected modelA to be recommended, got %s", res.RecommendedModel)
	}
}
