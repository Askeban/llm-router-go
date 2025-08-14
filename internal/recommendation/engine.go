package recommendation

import (
	"github.com/Askeban/llm-router-go/internal/models"
	"math"
	"sort"
	"strings"
)

type Ranked struct {
	models.ModelProfile
	Score float64
	Why   string
}

func norm(x, min, max float64) float64 {
	if max <= min {
		return 0
	}
	v := (x - min) / (max - min)
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return v
}
func BlendScore(category, difficulty string, m models.ModelProfile, perfBoost float64, minC, maxC, minL, maxL float64) (float64, string) {
	wPerf, wCost, wLat := 0.72, 0.18, 0.10
	dm := map[string]float64{"easy": 1.0, "medium": 1.05, "hard": 1.12}[strings.ToLower(difficulty)]
	if dm == 0 {
		dm = 1
	}
	base := m.Capabilities[strings.ToLower(category)]
	perf := (base + perfBoost) * dm
	nc := 1 - norm(m.CostInPer1K, minC, maxC)
	nl := 1 - norm(float64(m.AvgLatencyMs), minL, maxL)
	s := wPerf*perf + wCost*nc + wLat*nl
	return s, "perf(caps+metrics)/cost/latency blend"
}
func Rank(category, difficulty string, all []models.ModelProfile, perfBoost map[string]float64) (top models.ModelProfile, ranked []Ranked) {
	if len(all) == 0 {
		return models.ModelProfile{}, []Ranked{}
	}
	minC, maxC := math.MaxFloat64, 0.0
	minL, maxL := math.MaxFloat64, 0.0
	for _, m := range all {
		if m.CostInPer1K < minC {
			minC = m.CostInPer1K
		}
		if m.CostInPer1K > maxC {
			maxC = m.CostInPer1K
		}
		if float64(m.AvgLatencyMs) < minL {
			minL = float64(m.AvgLatencyMs)
		}
		if float64(m.AvgLatencyMs) > maxL {
			maxL = float64(m.AvgLatencyMs)
		}
	}
	for _, m := range all {
		boost := perfBoost[m.ID]
		s, why := BlendScore(category, difficulty, m, boost, minC, maxC, minL, maxL)
		ranked = append(ranked, Ranked{ModelProfile: m, Score: s, Why: why})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].Score > ranked[j].Score })
	return ranked[0].ModelProfile, ranked
}
