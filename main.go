package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

func main() {
	serverMux := http.NewServeMux()
	// Setting up our readiness endpoint
	serverMux.HandleFunc("GET /api/healthz", readiness)

	// Handle that adds a file server.
	// Fileserver uses the http.Dir to map the
	// current directory to http address.
	apiCfg := apiConfig{}
	app := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	serverMux.Handle("/app/", apiCfg.middlewareMetricsInt(app))

	// Handle hits to the file server
	serverMux.HandleFunc("GET /api/metrics", apiCfg.hitCounter)
	serverMux.HandleFunc("POST /api/reset", apiCfg.restCounter)

	server := http.Server{
		Handler: serverMux,
		Addr:    ":8080",
	}
	log.Printf("Running server on port: %v", server.Addr)
	server.ListenAndServe()

}

// Handlers
// Want the readiness endpoint to be accessible at /healthz using any HTTP method
// Should return: 200 OK status code
// Header => Content-Type: text/plain; charset=utf-8
// Body => OK
// Later the endpoint can be enhanced to return 403 Service Unavailable status code
// if the server is not ready
func readiness(respWriter http.ResponseWriter, req *http.Request) {
	// Set content-type
	req.Header.Add("Content-Type", "text/plain; charset=utf-8")
	// Write status code
	respWriter.WriteHeader(200)
	// Write the body
	respWriter.Write([]byte("OK"))
}

// Want a handler that writes the number of requests that have been counted as plain text in this format to the HTTP response:
// Hits: x
func (cfg *apiConfig) hitCounter(respWriter http.ResponseWriter, req *http.Request) {
	// Set content-type
	req.Header.Add("Content-Type", "text/plain; charset=utf-8")
	// Write status code
	respWriter.WriteHeader(200)
	// Write the body
	hitCounter := fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())
	respWriter.Write([]byte(hitCounter))
}

// Want a handler to reset the counter
func (cfg *apiConfig) restCounter(respWriter http.ResponseWriter, req *http.Request) {
	// Set content-type
	req.Header.Add("Content-Type", "text/plain; charset=utf-8")
	// Write status code
	respWriter.WriteHeader(200)
	// Write the body
	hitCounterBefore := fmt.Sprintf("Hits (before reset): %v", cfg.fileserverHits.Load())
	// Resets back to zero
	cfg.fileserverHits.Store(0)
	hitCounterAfter := fmt.Sprintf("Hits (before reset): %v", cfg.fileserverHits.Load())
	hitCounter := hitCounterBefore + "\n" + hitCounterAfter
	respWriter.Write([]byte(hitCounter))
}

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
