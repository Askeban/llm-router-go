package scoring

import (
    "testing"
    "llm-router-go/internal/types"
)

func TestComputeSubScoresRanges(t *testing.T) {
    prompt := "Write a Python function to add two numbers and return the result."
    ctx := &types.Context{Files: []types.File{{Path: "main.py", Content: "def add(a, b): return a + b"}}}
    l, c, task, ctxScore := ComputeSubScores(prompt, ctx)
    if l < 1 || l > 10 {
        t.Fatalf("linguistic score out of range: %v", l)
    }
    if c < 1 || c > 10 {
        t.Fatalf("conceptual score out of range: %v", c)
    }
    if task < 1 || task > 10 {
        t.Fatalf("task score out of range: %v", task)
    }
    if ctxScore < 1 || ctxScore > 10 {
        t.Fatalf("context score out of range: %v", ctxScore)
    }
}

func TestEstimateOutputTokensMinimum(t *testing.T) {
    // Even with zero input tokens, estimate should be at least 10
    n := EstimateOutputTokens(0, 5.0)
    if n < 10 {
        t.Fatalf("expected at least 10 output tokens, got %d", n)
    }
    // With small input tokens, ensure output is scaled appropriately
    m := EstimateOutputTokens(10, 5.0)
    if m < 10 {
        t.Fatalf("expected output >= 10 tokens, got %d", m)
    }
}