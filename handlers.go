package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	defer req.Body.Close()

	data, err := io.ReadAll(req.Body)
	if err != nil {
		respondWithError(respWriter, 500, "couldn't read request")
		return
	}
	params := parameters{}
	err = json.Unmarshal(data, &params)
	if err != nil {
		respondWithError(respWriter, 500, "couldn't unmarshal parameters")
		return
	}
	if strings.Trim(params.Email, " ") == "" {
		respondWithError(respWriter, 400, "entered email was invalid (empty string)")
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
		message := fmt.Sprintf("Error: %v\n", err)
		respondWithError(respWriter, 500, message)
		return
	}
	ourUser := User{
		ID:        newUser.ID,
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
		Email:     newUser.Email,
	}
	respondWithJSON(respWriter, 201, ourUser)
	return
}

// Want the readiness endpoint to be accessible at /healthz using any HTTP method
// Should return: 200 OK status code
// Header => Content-Type: text/plain; charset=utf-8
// Body => OK
// Later the endpoint can be enhanced to return 403 Service Unavailable status code
// if the server is not ready
func readiness(respWriter http.ResponseWriter, req *http.Request) {
	respondWithText(respWriter, 200, []byte("OK"))
}

// Want a handler that writes the number of requests that have been counted as plain text in this format to the HTTP response:
// Hits: x
func (cfg *apiConfig) hitCounter(respWriter http.ResponseWriter, req *http.Request) {
	hitCounter := fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())
	respondWithText(respWriter, 200, []byte(hitCounter))
}

// Want a handler to reset the counter
func (cfg *apiConfig) resetCounter(respWriter http.ResponseWriter, req *http.Request) {
	// Check platform
	if cfg.platform != "dev" {
		respondWithError(respWriter, 403, "platform not authenticated")
		return
	}
	// Delete all users in the database
	ctx := context.Background()
	err := cfg.dbQuerries.DeleteAllUsers(ctx)
	if err != nil {
		respondWithError(respWriter, 500, "error making change")
		return
	}
	hitCounterBefore := fmt.Sprintf("Hits (before reset): %v", cfg.fileserverHits.Load())
	// Resets back to zero
	cfg.fileserverHits.Store(0)
	hitCounterAfter := fmt.Sprintf("Hits (after reset): %v", cfg.fileserverHits.Load())
	hitCounter := hitCounterBefore + "\n" + hitCounterAfter
	// respWriter.Write([]byte(hitCounter))
	respondWithText(respWriter, 200, []byte(hitCounter))
}

// Want a handler to get the admin page
func (cfg *apiConfig) adminHandler(respWriter http.ResponseWriter, req *http.Request) {
	// Set content-type
	// Set the body
	file, err := os.ReadFile("./admin/metrics.html")
	if err != nil {
		log.Fatal(err)
	}
	fileString := string(file)
	updatedFileString := fmt.Sprintf(fileString, cfg.fileserverHits.Load())
	respondWithHTML(respWriter, 200, []byte(updatedFileString))
}

// Handler to encode JSON response
func validateChirpLength(w http.ResponseWriter, r *http.Request) {
	// Request parameters
	type parameters struct {
		Body string `json:"body"`
	}

	// type returnValue struct {
	// 	Valid bool   `json:"valid"`
	// 	Error string `json:"error"`
	// }
	type returnValue struct {
		CleanedBody string `json:"cleaned_body"`
		Error       string `json:"error"`
	}
	defer r.Body.Close()

	// Read in data
	data, err := io.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, 500, "couldn't read request")
		return
	}
	// Now get the data
	params := parameters{}
	err = json.Unmarshal(data, &params)
	if err != nil {
		respondWithError(w, 500, "couldn't unmarshal parameters")
		return
	}
	if len(params.Body) > 120 {
		errorMessage := "Chirp is too long"
		respondWithError(w, 400, errorMessage)
		return
	}

	newBodyResponse := returnValue{
		CleanedBody: cleanWords(params.Body),
	}

	respondWithJSON(w, 200, newBodyResponse)
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

func respondWithJSON(respWriter http.ResponseWriter, code int, payload interface{}) error {
	response, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	respWriter.Header().Set("Content-Type", "application/json")
	respWriter.Header().Set("Access-Control-Allow-Origin", "*")
	respWriter.WriteHeader(code)
	respWriter.Write(response)
	return nil
}

func respondWithError(respWriter http.ResponseWriter, code int, msg string) error {
	return respondWithJSON(respWriter, code, map[string]string{"error": msg})
}

func respondWithText(respWriter http.ResponseWriter, code int, payload []byte) error {
	respWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
	respWriter.Header().Set("Access-Control-Allow-Origin", "*")
	respWriter.WriteHeader(code)
	respWriter.Write(payload)
	return nil
}

func respondWithHTML(respWriter http.ResponseWriter, code int, payload []byte) error {
	respWriter.Header().Set("Content-Type", "text/html")
	respWriter.Header().Set("Access-Control-Allow-Origin", "*")
	respWriter.WriteHeader(code)
	respWriter.Write(payload)
	return nil
}
