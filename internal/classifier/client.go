package classifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// -----------------------------------------------------------------------------
// Public client
// -----------------------------------------------------------------------------

type Client struct {
	Base     string
	http     *http.Client
	cfg      *RuleConfig       // parsed config (patterns, weights)
	cfgState compiledRuleState // precompiled regex for performance
}

func New(base string) *Client {
	c := &Client{
		Base: base,
		http: &http.Client{Timeout: 8 * time.Second},
	}
	cfg := loadRuleConfig()
	c.cfg = cfg
	c.cfgState = compileRuleConfig(cfg)
	return c
}

// Back-compat remote payload
type req struct {
	Prompt string `json:"prompt"`
}
type resp struct {
	Category   string  `json:"category"`
	Difficulty string  `json:"difficulty"`
	Confidence float64 `json:"confidence"`
	Sentiment  string  `json:"sentiment,omitempty"`
}

// Classify returns (category, difficulty) to match existing call sites.
func (c *Client) Classify(ctx context.Context, prompt string) (string, string, error) {
	// Try remote first if configured and healthy
	if c.Base != "" {
		if out, err := c.remote(ctx, prompt); err == nil && out.Category != "" && out.Difficulty != "" {
			return out.Category, out.Difficulty, nil
		}
	}
	// Fallback to local rules
	out := c.localAnalyze(prompt)
	return out.Category, out.Difficulty, nil
}

// ClassifyFull (optional) exposes sentiment + confidence for callers that want it.
func (c *Client) ClassifyFull(ctx context.Context, prompt string) (resp, error) {
	if c.Base != "" {
		if out, err := c.remote(ctx, prompt); err == nil && out.Category != "" && out.Difficulty != "" {
			return *out, nil
		}
	}
	return c.localAnalyze(prompt), nil
}

func (c *Client) remote(ctx context.Context, prompt string) (*resp, error) {
	rb, _ := json.Marshal(req{Prompt: prompt})
	u := strings.TrimRight(c.Base, "/") + "/classify"
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(rb))
	hreq.Header.Set("content-type", "application/json")
	res, err := c.http.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New("remote classify: non-200")
	}
	var out resp
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// -----------------------------------------------------------------------------
// Rules-driven local classifier
// -----------------------------------------------------------------------------

// RuleConfig is the JSON schema you edit in configs/classifier_rules.json.
type RuleConfig struct {
	Priority   []string                `json:"priority"` // tie-break order (first wins)
	Categories map[string]CategoryRule `json:"categories"`
	Difficulty DifficultyRule          `json:"difficulty"`
	Sentiment  SentimentRule           `json:"sentiment"`
}

type CategoryRule struct {
	// Weighted pattern groups (all optional)
	Contains     []WeightedStrings `json:"contains,omitempty"`
	StartsWith   []WeightedStrings `json:"startswith,omitempty"`
	EndsWith     []WeightedStrings `json:"endswith,omitempty"`
	Regex        []WeightedRegex   `json:"regex,omitempty"`
	Cooccur      []CooccurRule     `json:"cooccur,omitempty"`       // e.g., language AND action
	Threshold    float64           `json:"threshold,omitempty"`     // short-circuit if score >= threshold
	HardStop     bool              `json:"hard_stop,omitempty"`     // stop evaluating other categories if this one meets threshold
	MaxPerGroup  int               `json:"max_per_group,omitempty"` // cap matches counted per group (0 = unlimited)
	WeightScalar float64           `json:"weight_scalar,omitempty"` // final multiplier
}

type WeightedStrings struct {
	Terms  []string `json:"terms"`
	Weight float64  `json:"weight"`
}

type WeightedRegex struct {
	Pattern string  `json:"pattern"`
	Weight  float64 `json:"weight"`
}

type CooccurRule struct {
	AnyOfA []string `json:"any_of_a"` // if (any in A) AND (any in B) => add weight
	AnyOfB []string `json:"any_of_b"`
	Weight float64  `json:"weight"`
}

// DifficultyRule controls easy/medium/hard heuristics.
type DifficultyRule struct {
	// Global weights
	Weights struct {
		Length       float64 `json:"length"`
		Structure    float64 `json:"structure"`
		Requirements float64 `json:"requirements"`
		Domain       float64 `json:"domain"`
	} `json:"weights"`
	// Length normalization (tokens)
	LengthCapTokens int `json:"length_cap_tokens"`

	// Structure
	NewlinesCap int `json:"newlines_cap"`

	// Signals
	RequirementTerms []string         `json:"requirement_terms"` // e.g., constraints, optimize, SLA…
	DomainBonuses    map[string]Bonus `json:"domain_bonuses"`    // per-category bonuses + extra hard terms
}

type Bonus struct {
	Base      float64  `json:"base"`       // fixed bonus if category matches
	HardTerms []string `json:"hard_terms"` // extra terms that add increments
	Inc       float64  `json:"increment"`  // per hard-term increment
	Max       float64  `json:"max"`        // clamp bonus to this
}

// SentimentRule controls lexicon + negation patterns.
type SentimentRule struct {
	Positive         []string `json:"positive"`
	Negative         []string `json:"negative"`
	NegatePosRegexes []string `json:"negate_positive_regexes"` // e.g., "not good"
	NegateNegRegexes []string `json:"negate_negative_regexes"` // e.g., "not bad"
	MinGap           int      `json:"min_gap"`                 // >gap => decide; else neutral
}

// In-memory compiled regex cache
type compiledRuleState struct {
	categoryRegex map[string][]*regexp.Regexp
	sentNegPos    []*regexp.Regexp
	sentNegNeg    []*regexp.Regexp
}

func loadRuleConfig() *RuleConfig {
	path := strings.TrimSpace(os.Getenv("CLASSIFIER_RULES_PATH"))
	if path == "" {
		path = filepath.Join(".", "configs", "classifier_rules.json")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		// if missing, return a safe default
		return defaultRuleConfig()
	}
	var cfg RuleConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return defaultRuleConfig()
	}
	return fillMissingWithDefaults(&cfg)
}

func compileRuleConfig(cfg *RuleConfig) compiledRuleState {
	st := compiledRuleState{
		categoryRegex: make(map[string][]*regexp.Regexp),
	}
	for cat, rule := range cfg.Categories {
		for _, rr := range rule.Regex {
			if rr.Pattern == "" {
				continue
			}
			if rx, err := regexp.Compile(rr.Pattern); err == nil {
				st.categoryRegex[cat] = append(st.categoryRegex[cat], rx)
			}
		}
	}
	for _, p := range cfg.Sentiment.NegatePosRegexes {
		if rx, err := regexp.Compile(p); err == nil {
			st.sentNegPos = append(st.sentNegPos, rx)
		}
	}
	for _, p := range cfg.Sentiment.NegateNegRegexes {
		if rx, err := regexp.Compile(p); err == nil {
			st.sentNegNeg = append(st.sentNegNeg, rx)
		}
	}
	return st
}

func (c *Client) localAnalyze(prompt string) resp {
	s := strings.ToLower(strings.TrimSpace(prompt))

	// --- Category scoring ---
	score := map[string]float64{}
	for cat, rule := range c.cfg.Categories {
		score[cat] = c.scoreCategory(cat, s, rule)
	}

	// choose top by score, break ties by Priority
	category := topCategory(score, c.cfg.Priority)

	// --- Difficulty ---
	diff := c.computeDifficulty(s, category)

	// --- Sentiment ---
	sent := c.computeSentiment(s)

	// simple confidence: relative margin between top1 and top2
	conf := 0.65
	if len(score) >= 2 {
		type kv struct {
			k string
			v float64
		}
		arr := make([]kv, 0, len(score))
		for k, v := range score {
			arr = append(arr, kv{k, v})
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].v > arr[j].v })
		if arr[0].v > 0 {
			conf = 0.6 + 0.4*clamp01((arr[0].v-arr[1].v)/(arr[0].v+1e-6))
		}
	}

	return resp{
		Category:   category,
		Difficulty: diff,
		Sentiment:  sent,
		Confidence: conf,
	}
}

func (c *Client) scoreCategory(cat, s string, rule CategoryRule) float64 {
	score := 0.0
	addTerms := func(groups []WeightedStrings, fn func(hay, needle string) bool, capN int, w float64) {
		for _, g := range groups {
			count := 0
			for _, term := range g.Terms {
				if fn(s, term) {
					score += g.Weight
					count++
					if capN > 0 && count >= capN {
						break
					}
				}
			}
		}
	}
	// contains
	addTerms(rule.Contains, strings.Contains, rule.MaxPerGroup, 0)
	// startswith
	addTerms(rule.StartsWith, func(h, n string) bool { return strings.HasPrefix(strings.TrimSpace(h), n) }, rule.MaxPerGroup, 0)
	// endswith
	addTerms(rule.EndsWith, func(h, n string) bool { return strings.HasSuffix(strings.TrimSpace(h), n) }, rule.MaxPerGroup, 0)
	// regex
	for _, rx := range c.cfgState.categoryRegex[cat] {
		if rx.MatchString(s) {
			// find weight for this pattern
			for _, rr := range rule.Regex {
				if rr.Pattern == rx.String() {
					score += rr.Weight
					break
				}
			}
		}
	}
	// co-occur: (any A) AND (any B)
	for _, co := range rule.Cooccur {
		if anyContains(s, co.AnyOfA) && anyContains(s, co.AnyOfB) {
			score += co.Weight
		}
	}
	if rule.WeightScalar != 0 {
		score *= rule.WeightScalar
	}
	return score
}

func topCategory(score map[string]float64, priority []string) string {
	// pick max score; break ties by priority order
	var best string
	bestVal := -1e9
	for k, v := range score {
		if v > bestVal {
			bestVal, best = v, k
		} else if v == bestVal {
			// tie — use priority
			if priorIndex(k, priority) < priorIndex(best, priority) {
				best = k
			}
		}
	}
	if best == "" {
		best = "general"
	}
	return best
}

func priorIndex(k string, pri []string) int {
	for i, v := range pri {
		if v == k {
			return i
		}
	}
	return 1 << 30
}

// Difficulty computation
func (c *Client) computeDifficulty(s string, category string) string {
	// length approx (chars/4 tokens)
	tokens := float64(len([]rune(s))) / 4.0
	lenCap := float64(max(1, c.cfg.Difficulty.LengthCapTokens))
	lenScore := clamp01(tokens / lenCap)

	// structure
	newlines := float64(strings.Count(s, "\n"))
	newlineCap := float64(max(1, c.cfg.Difficulty.NewlinesCap))
	structScore := clamp01(newlines / newlineCap)

	// requirements/constraints signals
	reqSignals := float64(countAny(s, c.cfg.Difficulty.RequirementTerms))
	reqScore := clamp01(reqSignals / 5.0)

	// domain bonus
	dom := 0.0
	if bon, ok := c.cfg.Difficulty.DomainBonuses[category]; ok {
		dom += bon.Base
		if bon.Inc != 0 && len(bon.HardTerms) > 0 {
			hits := float64(countAny(s, bon.HardTerms))
			dom += bon.Inc * hits
		}
		if bon.Max > 0 && dom > bon.Max {
			dom = bon.Max
		}
	}
	// weighted sum
	w := c.cfg.Difficulty.Weights
	raw := w.Length*lenScore + w.Structure*structScore + w.Requirements*reqScore + w.Domain*dom
	diff := clamp01(raw)
	switch {
	case diff < 0.33:
		return "easy"
	case diff > 0.66:
		return "hard"
	default:
		return "medium"
	}
}

// Sentiment
func (c *Client) computeSentiment(s string) string {
	pos := countAny(s, c.cfg.Sentiment.Positive)
	neg := countAny(s, c.cfg.Sentiment.Negative)
	// negations flip the polarity of nearby phrases (very simple heuristic)
	for _, rx := range c.cfgState.sentNegPos {
		if rx.MatchString(s) {
			// "not good" => decrease pos, increase neg
			if pos > 0 {
				pos--
			}
			neg++
		}
	}
	for _, rx := range c.cfgState.sentNegNeg {
		if rx.MatchString(s) {
			// "not bad" => decrease neg, increase pos
			if neg > 0 {
				neg--
			}
			pos++
		}
	}
	gap := pos - neg
	if gap >= c.cfg.Sentiment.MinGap {
		return "positive"
	}
	if -gap >= c.cfg.Sentiment.MinGap {
		return "negative"
	}
	return "neutral"
}

// -----------------------------------------------------------------------------
// Utilities + defaults
// -----------------------------------------------------------------------------

func anyContains(s string, arr []string) bool {
	for _, a := range arr {
		if strings.Contains(s, a) {
			return true
		}
	}
	return false
}

func countAny(s string, arr []string) int {
	n := 0
	for _, a := range arr {
		n += strings.Count(s, a)
	}
	return n
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func max(a, b int) int {
	if a >= b {
		return a
	}
	return b
}

// Default config used if configs/classifier_rules.json is missing or invalid.
func defaultRuleConfig() *RuleConfig {
	cfg := &RuleConfig{
		Priority: []string{"coding", "math", "question", "reasoning", "business", "chat", "general"},
		Categories: map[string]CategoryRule{
			"coding": {
				Contains: []WeightedStrings{
					{Terms: []string{"```", "unit test", "stack trace", "write a function", "refactor", "compile error"}, Weight: 2.0},
					{Terms: []string{"go", "golang", "python", "java", "javascript", "typescript", "rust", "c++", "c#", "regex", "sql", "dockerfile"}, Weight: 1.5},
					{Terms: []string{"package ", "import ", "def ", "class ", "#include", "public static", "fn ", "console.log", "<?php", "using ", " => ", " := "}, Weight: 2.5},
				},
				Cooccur: []CooccurRule{
					{AnyOfA: []string{"go", "golang", "python", "javascript", "typescript", "java", "rust", "c++", "c#"}, AnyOfB: []string{"write", "implement", "generate", "create", "build"}, Weight: 2.5},
				},
				Threshold:    3.0,
				WeightScalar: 1.0,
			},
			"math": {
				Contains: []WeightedStrings{
					{Terms: []string{"solve", "equation", "integral", "derivative", "matrix", "vector", "limit", "proof", "theorem", "lemma", "corollary"}, Weight: 2.0},
					{Terms: []string{"∑", "∫", "√", "±", "÷", "×", "≤", "≥", "\\frac", "\\sum_", "\\int_"}, Weight: 2.0},
				},
				Regex: []WeightedRegex{
					{Pattern: `\b[a-zA-Z]\s*=\s*[^=]`, Weight: 1.0}, // simple variable equation
				},
				Threshold: 3.0,
			},
			"question": {
				Regex: []WeightedRegex{
					{Pattern: `^(who|what|where|when|why|how|which)\b.*\?`, Weight: 2.0},
					{Pattern: `\?$`, Weight: 1.0},
				},
			},
			"reasoning": {
				Contains: []WeightedStrings{
					{Terms: []string{"explain", "analyze", "compare", "contrast", "evaluate", "assess", "justify", "derive", "step by step", "reason about"}, Weight: 1.0},
				},
				Regex: []WeightedRegex{
					{Pattern: `\b(step by step|chain of thought|show your work)\b`, Weight: 2.0},
				},
			},
			"business": {
				Contains: []WeightedStrings{
					{Terms: []string{"market", "business", "financial", "revenue", "profit", "roi", "budget", "invoice", "pricing", "segmentation", "forecast", "p&l", "capex", "opex"}, Weight: 1.0},
				},
			},
			"chat": {
				Contains: []WeightedStrings{
					{Terms: []string{"hello", "hi ", "hey", "how are you", "what's up", "let's chat", "good morning"}, Weight: 1.0},
				},
				StartsWith: []WeightedStrings{
					{Terms: []string{"hello", "hi ", "hey"}, Weight: 1.0},
				},
			},
			"general": {},
		},
		Difficulty: DifficultyRule{
			Weights: struct {
				Length       float64 `json:"length"`
				Structure    float64 `json:"structure"`
				Requirements float64 `json:"requirements"`
				Domain       float64 `json:"domain"`
			}{Length: 0.45, Structure: 0.15, Requirements: 0.25, Domain: 0.15},
			LengthCapTokens:  400,
			NewlinesCap:      10,
			RequirementTerms: []string{"requirement", "requirements", "constraint", "constraints", "edge case", "optimize", "time complexity", "big-o", "latency", "throughput", "SLA", "security", "privacy", "compliance"},
			DomainBonuses: map[string]Bonus{
				"math":   {Base: 0.20, HardTerms: []string{"prove", "theorem", "spectral", "fourier", "laplace", "nonlinear"}, Inc: 0.10, Max: 0.40},
				"coding": {Base: 0.00, HardTerms: []string{"concurrency", "deadlock", "lock-free", "cap theorem", "distributed", "consensus", "regex", "parser", "time complexity"}, Inc: 0.10, Max: 0.40},
			},
		},
		Sentiment: SentimentRule{
			Positive:         []string{"love", "like", "great", "excellent", "awesome", "thanks", "amazing", "perfect", "nice", "cool"},
			Negative:         []string{"hate", "terrible", "awful", "bad", "useless", "angry", "wtf", "stupid", "garbage", "broken", "doesn't work"},
			NegatePosRegexes: []string{`not\s+(good|great|awesome|perfect|amazing|helpful|nice|cool)`},
			NegateNegRegexes: []string{`not\s+(bad|awful|terrible|useless|stupid|annoying|broken)`},
			MinGap:           2,
		},
	}
	return cfg
}

func fillMissingWithDefaults(in *RuleConfig) *RuleConfig {
	if in == nil {
		return defaultRuleConfig()
	}
	def := defaultRuleConfig()
	if len(in.Priority) == 0 {
		in.Priority = def.Priority
	}
	if in.Categories == nil || len(in.Categories) == 0 {
		in.Categories = def.Categories
	} else {
		// ensure "general" exists
		if _, ok := in.Categories["general"]; !ok {
			in.Categories["general"] = CategoryRule{}
		}
	}
	// Difficulty defaults
	if in.Difficulty.LengthCapTokens == 0 {
		in.Difficulty.LengthCapTokens = def.Difficulty.LengthCapTokens
	}
	if in.Difficulty.NewlinesCap == 0 {
		in.Difficulty.NewlinesCap = def.Difficulty.NewlinesCap
	}
	if len(in.Difficulty.RequirementTerms) == 0 {
		in.Difficulty.RequirementTerms = def.Difficulty.RequirementTerms
	}
	if in.Difficulty.DomainBonuses == nil {
		in.Difficulty.DomainBonuses = def.Difficulty.DomainBonuses
	}
	// Weights
	if in.Difficulty.Weights.Length == 0 && in.Difficulty.Weights.Structure == 0 &&
		in.Difficulty.Weights.Requirements == 0 && in.Difficulty.Weights.Domain == 0 {
		in.Difficulty.Weights = def.Difficulty.Weights
	}
	// Sentiment
	if len(in.Sentiment.Positive) == 0 {
		in.Sentiment.Positive = def.Sentiment.Positive
	}
	if len(in.Sentiment.Negative) == 0 {
		in.Sentiment.Negative = def.Sentiment.Negative
	}
	if len(in.Sentiment.NegatePosRegexes) == 0 {
		in.Sentiment.NegatePosRegexes = def.Sentiment.NegatePosRegexes
	}
	if len(in.Sentiment.NegateNegRegexes) == 0 {
		in.Sentiment.NegateNegRegexes = def.Sentiment.NegateNegRegexes
	}
	if in.Sentiment.MinGap == 0 {
		in.Sentiment.MinGap = def.Sentiment.MinGap
	}
	return in
}

// ReloadFrom re-reads a rules file (use in an admin endpoint)
func (c *Client) ReloadFrom(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cfg RuleConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return err
	}
	c.cfg = fillMissingWithDefaults(&cfg)
	c.cfgState = compileRuleConfig(c.cfg)
	return nil
}

// Explain returns raw per-category scores for a prompt (for debugging)
func (c *Client) Explain(prompt string) map[string]float64 {
	s := strings.ToLower(strings.TrimSpace(prompt))
	out := map[string]float64{}
	for cat, rule := range c.cfg.Categories {
		out[cat] = c.scoreCategory(cat, s, rule)
	}
	return out
}
