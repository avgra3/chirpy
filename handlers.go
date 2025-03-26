package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

// Handlers
// Adding a new user
func (cfg *apiConfig) newUserHandler(respWriter http.ResponseWriter, req *http.Request) {
	// Takes in JSON request like:
	// {
	//   "email": "user@example.com"
	// }
	type parameters struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respWriter.WriteHeader(500)
		return
	}
	if strings.Trim(params.Email, " ") == "" {
		log.Printf("Error: Empty email field\n")
		log.Printf("Error: email field => %v\n", params.Email)
		respWriter.WriteHeader(400)
		return
	}

	// Responds with HTTP 201 Created
	// {
	//   "id": "50746277-23c6-4d85-a890-564c0044c2fb",
	//   "created_at": "2021-07-07T00:00:00Z",
	//   "updated_at": "2021-07-07T00:00:00Z",
	//   "email": "user@example.com"
	// }

	// Need to actually make the user:
	ctx := context.Background()
	newUser, err := cfg.dbQuerries.CreateUser(ctx, params.Email)
	if err != nil {
		respWriter.WriteHeader(500)
		log.Printf("Error: %v\n", err)
		return
	}
	ourUser := User{
		ID:        newUser.ID,
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
		Email:     newUser.Email,
	}
	dat, err := json.Marshal(ourUser)
	if err != nil {
		respWriter.WriteHeader(500)
		log.Printf("Error: %v\n", err)
		return
	}
	respWriter.WriteHeader(201)
	respWriter.Header().Set("Content-Type", "application/json")
	respWriter.Write(dat)
	return
}

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
func (cfg *apiConfig) resetCounter(respWriter http.ResponseWriter, req *http.Request) {
	// Check platform
	if cfg.platform != "dev" {
		respWriter.WriteHeader(403)
		return
	}
	// Delete all users in the database
	ctx := context.Background()
	err := cfg.dbQuerries.DeleteAllUsers(ctx)
	if err != nil {
		respWriter.WriteHeader(500)
		return
	}

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

	// type returnValue struct {
	// 	Valid bool   `json:"valid"`
	// 	Error string `json:"error"`
	// }
	type returnValue struct {
		CleanedBody string `json:"cleaned_body"`
		Error       string `json:"error"`
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
	// returnVal.Valid = true
	returnVal.CleanedBody = cleanWords(params.Body)
	dat, err := json.Marshal(returnVal)
	if err != nil {
		w.WriteHeader(500)
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	w.Write(dat)
	return

}

// Helper function(s)
func cleanWords(input string) string {
	splitInput := strings.Split(input, " ")
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	cleanedInput := []string{}
	for _, word := range splitInput {
		cleanedWord := word
		for _, badWord := range badWords {
			if strings.ToUpper(cleanedWord) == strings.ToUpper(badWord) {
				cleanedWord = "****"
			}
		}
		cleanedInput = append(cleanedInput, cleanedWord)

	}
	return strings.Join(cleanedInput, " ")
}
