package storage

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

var (
	aliasesOnce sync.Once
	aliasesMap  map[string]string
)

func loadAliasesOnce() {
	aliasesMap = map[string]string{}
	b, err := os.ReadFile("configs/model_aliases.json")
	if err != nil {
		return // not fatal
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return
	}
	for k, v := range m {
		lk := strings.ToLower(strings.TrimSpace(k))
		aliasesMap[lk] = strings.TrimSpace(v)
	}
}

// ResolveAlias maps a user/slug/id → canonical model_id (as stored in DB).
// Falls back to the input if no mapping exists.
func ResolveAlias(id string) string {
	aliasesOnce.Do(loadAliasesOnce)
	key := strings.ToLower(strings.TrimSpace(id))
	if canon, ok := aliasesMap[key]; ok && canon != "" {
		return canon
	}
	return id
}

