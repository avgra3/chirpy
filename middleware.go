package main

import (
	"net/http"
	"sync/atomic"
)

// Types
type apiConfig struct {
	// Allows us to safely increment an integer across goroutines
	fileserverHits atomic.Int32
}

// Middleware
func (cfg *apiConfig) middlewareMetricsInt(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increments the filerserverHits counter ever time it's called.
		cfg.fileserverHits.Add(1)
		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
