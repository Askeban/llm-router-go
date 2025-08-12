package policy

import (
	"llm-router-go/internal/api"
	"strings"
)

type Decision struct {
	Top          string
	Alternatives []string
	Confidence   float32
	Rationale    string
	Flags        []string
}

func Score(c Catalog, req *api.RecommendRequest) Decision {
	long := req.Context != nil && (req.Context.SelectionBytes > 48000 || sumBytes(req) > 120000)
	lang := ""
	if req.Context != nil {
		lang = req.Context.Language
	}

	limit := map[string]bool{}
	if len(req.Catalog) > 0 {
		for _, m := range req.Catalog {
			limit[m] = true
		}
	}

	cands := []string{}
	for _, m := range c.Models {
		if len(limit) > 0 && !limit[m.Name] {
			continue
		}
		if lang != "" && !contains(m.Languages, lang) {
			continue
		}
		cands = append(cands, m.Name)
	}
	if len(cands) == 0 {
		for _, m := range c.Models {
			cands = append(cands, m.Name)
		}
	}

	var top string
	var conf float32 = 0.6
	var why []string
	flags := []string{}

	if long {
		for _, n := range []string{"claude-3.5-sonnet", "gemini-1.5-pro", "gpt-4o"} {
			if has(cands, n) {
				top = n
				break
			}
		}
		why = append(why, "Detected large context; prefer long-context models.")
		flags = append(flags, "large_context")
		conf = 0.72
	} else {
		for _, n := range []string{"gpt-4o-mini", "gpt-4o", "claude-3.5-sonnet"} {
			if has(cands, n) {
				top = n
				break
			}
		}
		why = append(why, "Moderate prompt; prefer fast code-generation.")
	}

	alts := []string{}
	for _, n := range cands {
		if n != top {
			alts = append(alts, n)
		}
	}
	return Decision{Top: top, Alternatives: alts, Confidence: conf, Rationale: strings.Join(why, " "), Flags: flags}
}

func sumBytes(r *api.RecommendRequest) int {
	if r.Context == nil {
		return 0
	}
	t := 0
	for _, s := range r.Context.Snippets {
		t += s.Bytes
	}
	return t
}
func contains(a []string, x string) bool {
	for _, v := range a {
		if v == x {
			return true
		}
	}
	return false
}
func has(a []string, x string) bool { return contains(a, x) }
