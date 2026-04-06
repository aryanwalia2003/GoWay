package middleware

import (
	"net/http"
	"strings"
)

// Auth is the API Key authentication middleware.
func Auth(apiKeys string) func(http.Handler) http.Handler {

	// TODO :  Not wired to the env's yet
	keys := make(map[string]bool) // string to bool hashmap
	for _, k := range strings.Split(apiKeys, ",") {
		if k != "" {
			keys[k] = true
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !keys[r.Header.Get("X-API-Key")] {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
