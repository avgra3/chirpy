package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

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

// Want a handler to get the admin page
func (cfg *apiConfig) adminHandler(respWriter http.ResponseWriter, req *http.Request) {
	// Set content-type
	req.Header.Add("Content-Type", "text/html")
	// Set the body
	file, err := os.ReadFile("./admin/metrics.html")
	if err != nil {
		log.Fatal(err)
	}
	fileString := string(file)
	//cfg.fileserverHits.Load()
	updatedFileString := fmt.Sprintf(fileString, cfg.fileserverHits.Load())
	// req.Body.Read([]byte(updatedFileString))
	// Write status code
	respWriter.WriteHeader(200)
	// Write the body
	respWriter.Write([]byte(updatedFileString))
}

// Handler to encode JSON response
func validateChirpLength(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	type returnValue struct {
		Valid bool   `json:"valid"`
		Error string `json:"error"`
	}

	returnVal := returnValue{}

	if len(params.Body) > 120 {
		errorMessage := "Chirp is too long"
		returnVal.Error = errorMessage
		dat, err := json.Marshal(returnVal)
		if err != nil {
			w.WriteHeader(500)
		}
		w.WriteHeader(400)
		w.Header().Set("Content-Type", "application/json")
		w.Write(dat)
		return
	}
	returnVal.Valid = true
	dat, err := json.Marshal(returnVal)
	if err != nil {
		w.WriteHeader(500)
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	w.Write(dat)
	return

}
