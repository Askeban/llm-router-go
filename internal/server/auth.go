package server

import (
	"net/http"
	"os"
	"strings"
)

func APIKeyAuth(next http.Handler) http.Handler {
	keysCSV := os.Getenv("ROUTER_API_KEYS")
	keys := map[string]bool{}
	for _, k := range strings.Split(keysCSV, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			keys[k] = true
		}
	}
	if len(keys) == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-API-Key")
		if key == "" || !keys[key] {
			http.Error(w, `{"error":"unauthorized","message":"missing or invalid API key"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
