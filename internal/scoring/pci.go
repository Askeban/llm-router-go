package scoring

import (
    "regexp"
    "strings"

    "llm-router-go/internal/types"
)

// tokenize splits text into lowercase alphanumeric tokens.  It treats any
// sequence of non‑word characters as a separator.
// tokenize splits text into lowercase alphanumeric tokens.  It treats any
// sequence of non‑word characters as a separator.
func tokenize(text string) []string {
    re := regexp.MustCompile(`[\W_]+`)
    raw := re.Split(text, -1)
    tokens := make([]string, 0, len(raw))
    for _, t := range raw {
        if t == "" {
            continue
        }
        tokens = append(tokens, strings.ToLower(t))
    }
    return tokens
}

// Tokenize exposes the internal tokenizer for use by other packages.
func Tokenize(text string) []string {
    return tokenize(text)
}

// ComputeSubScores calculates the four sub‑scores used in the Prompt Complexity
// Index: linguistic, conceptual, task and context complexity.  Each score
// ranges from 1 to 10.  A nil context is treated as having no files.
func ComputeSubScores(prompt string, ctx *types.Context) (linguistic, conceptual, task, contextScore float64) {
    tokens := tokenize(prompt)
    nTokens := len(tokens)
    // Sentence segmentation based on .!?  Keep at least one sentence.
    sentences := regexp.MustCompile(`[.!?]+`).Split(prompt, -1)
    nonEmpty := 0
    for _, s := range sentences {
        if strings.TrimSpace(s) != "" {
            nonEmpty++
        }
    }
    if nonEmpty == 0 {
        nonEmpty = 1
    }
    avgSentenceLength := float64(nTokens) / float64(nonEmpty)
    // Linguistic: scale average sentence length to [1,10]; 20 tokens per
    // sentence corresponds to a score of 10.
    linguistic = avgSentenceLength / 20.0 * 10.0
    if linguistic < 1.0 {
        linguistic = 1.0
    } else if linguistic > 10.0 {
        linguistic = 10.0
    }
    // Conceptual: ratio of unique tokens to total tokens.  More unique
    // concepts yield a higher score.
    unique := make(map[string]struct{})
    for _, t := range tokens {
        unique[t] = struct{}{}
    }
    uniqueRatio := 0.0
    if nTokens > 0 {
        uniqueRatio = float64(len(unique)) / float64(nTokens)
    }
    conceptual = uniqueRatio * 10.0
    if conceptual < 1.0 {
        conceptual = 1.0
    } else if conceptual > 10.0 {
        conceptual = 10.0
    }
    // Task complexity: count imperative verbs and conjunctions as proxies for
    // the number of steps requested.  A base of 2 plus 1.5 times the count
    // maps to the score.
    imperativeKeywords := []string{
        "write", "create", "generate", "refactor", "explain", "summarize", "compare",
        "analyse", "design", "implement", "translate", "debug", "fix", "review",
        "test", "build",
    }
    instructions := 0
    lowerPrompt := strings.ToLower(prompt)
    for _, kw := range imperativeKeywords {
        instructions += strings.Count(lowerPrompt, kw)
    }
    instructions += strings.Count(lowerPrompt, " and ") + strings.Count(lowerPrompt, " then ")
    task = 2.0 + 1.5*float64(instructions)
    if task < 1.0 {
        task = 1.0
    }
    if task > 10.0 {
        task = 10.0
    }
    // Context complexity: scale by total number of tokens in the context files.
    contextTokens := 0
    if ctx != nil {
        for _, f := range ctx.Files {
            contextTokens += len(tokenize(f.Content))
        }
    }
    contextScore = float64(contextTokens) / 1000.0 * 10.0
    if contextScore < 1.0 {
        contextScore = 1.0
    }
    if contextScore > 10.0 {
        contextScore = 10.0
    }
    return
}

// ComputeComplexity combines the sub‑scores into a single Prompt Complexity
// Index (PCI) using fixed weights: 0.3 × linguistic + 0.4 × conceptual +
// 0.2 × task + 0.1 × context.
func ComputeComplexity(prompt string, ctx *types.Context) (float64, map[string]float64) {
    l, c, t, ctxScore := ComputeSubScores(prompt, ctx)
    pci := 0.3*l + 0.4*c + 0.2*t + 0.1*ctxScore
    return pci, map[string]float64{
        "linguistic": l,
        "conceptual": c,
        "task":       t,
        "context":    ctxScore,
    }
}

// EstimateOutputTokens estimates the number of output tokens given the number
// of input tokens and the complexity score.  It scales input tokens by a
// multiplier derived from complexity.  A minimum of 10 tokens is ensured.
func EstimateOutputTokens(tokensIn int, complexity float64) int {
    // complexity is assumed to lie in [1,10]; map to [0.5, 1.5] multiplier
    multiplier := 0.5 + (complexity / 10.0)
    out := float64(tokensIn) * multiplier * 0.6
    if out < 10.0 {
        out = 10.0
    }
    return int(out)
}