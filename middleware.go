package main

import (
	"net/http"
	"sync/atomic"

	"github.com/avgra3/chirpy/internal/database"
)

// Types
type apiConfig struct {
	// Allows us to safely increment an integer across goroutines
	fileserverHits atomic.Int32
	// For connecting to our database
	dbQuerries *database.Queries
	// Platform
	platform string
	// JWT Secret
	jwtSecret string
	// Polka API Key
	polkaKey string
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
