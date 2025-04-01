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
	"time"

	auth "github.com/avgra3/chirpy/internal/auth"
	"github.com/avgra3/chirpy/internal/database"
	"github.com/google/uuid"
)

// Handlers
// Adding a new user
func (cfg *apiConfig) userLogin(respWriter http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Password         string `json:"password"`
		Email            string `json:"email"`
		ExpiresInSeconds int    `json:"expires_in_secods"`
	}
	defer req.Body.Close()
	data, err := io.ReadAll(req.Body)
	if err != nil {
		respondWithError(respWriter, 500, "couldn't read request")
		return
	}
	userByEmail := parameters{}
	err = json.Unmarshal(data, &userByEmail)
	if err != nil {
		respondWithError(respWriter, 500, "couldn't unmarshal parameters")
		return
	}
	if userByEmail.ExpiresInSeconds <= 0 || userByEmail.ExpiresInSeconds > 60*60 {
		// If not provided or greater than 1 hour, will default to 1 hour, in seconds
		userByEmail.ExpiresInSeconds = 60 * 60
	}
	jwtDuration := time.Duration(userByEmail.ExpiresInSeconds) * time.Second

	ctx := context.Background()
	user, err := cfg.dbQuerries.UserLogin(ctx, userByEmail.Email)
	if err != nil {
		respondWithError(respWriter, 401, "Incorrect email or password")
		return
	}
	err = auth.CheckPasswordHash(user.HashedPassword, userByEmail.Password)
	if err != nil {
		respondWithError(respWriter, 401, "Incorrect email or password")
		return
	}
	// Once we are sure the user can log in, we create the JWT
	jwt, err := auth.MakeJWT(user.ID, cfg.jwtSecret, jwtDuration)
	if err != nil {
		respondWithError(respWriter, 500, "Unable to create token at this time")
	}

	authUser := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		Token:     jwt,
	}

	respondWithJSON(respWriter, 200, authUser)
	return

}

func (cfg *apiConfig) newUserHandler(respWriter http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
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
	if strings.Trim(params.Password, " ") == "" {
		respondWithError(respWriter, 400, "entered password was invalid (empty string)")
		return
	}

	// Need to actually make the user:
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(respWriter, 500, "unable to hash password")
		return
	}
	paramsHashed := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	}
	ctx := context.Background()
	newUser, err := cfg.dbQuerries.CreateUser(ctx, paramsHashed)
	if err != nil {
		message := fmt.Sprintf("Error: %v\n", err)
		respondWithError(respWriter, 500, message)
		return
	}
	hashedPasword, err := auth.HashPassword(newUser.HashedPassword)
	if err != nil {
		message := fmt.Sprintf("Error hashing password")
		respondWithError(respWriter, 500, message)
	}
	ourUser := User{
		ID:             newUser.ID,
		CreatedAt:      newUser.CreatedAt,
		UpdatedAt:      newUser.UpdatedAt,
		Email:          newUser.Email,
		HashedPassword: hashedPasword,
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
func (cfg *apiConfig) getChirps(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	chirps, err := cfg.dbQuerries.GetChirps(ctx)
	if err != nil {
		errMessage := fmt.Sprintf("ERROR: %v", err)
		respondWithError(w, 500, errMessage)
	}

	respondWithJSON(w, 200, chirps)
	return
}

func (cfg *apiConfig) getChirp(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	userID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		errMessage := fmt.Sprintf("ERROR: %v", err)
		respondWithError(w, 500, errMessage)

	}
	chirps, err := cfg.dbQuerries.GetChirp(ctx, userID)
	if err != nil {
		errMessage := fmt.Sprintf("ERROR: %v", err)
		respondWithError(w, 404, errMessage)
	}

	respondWithJSON(w, 200, chirps)
	return
}

func (cfg *apiConfig) newChirps(w http.ResponseWriter, r *http.Request) {
	// Request parameters
	type parameters struct {
		Body   string    `json:"body"`
		UserId uuid.UUID `json:"user_id"`
		Token  string    `json:"token"`
	}

	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, 500, "couldn't read request")
		return
	}
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
	// Check valid token
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		errorMessage := fmt.Sprintf("Error: %v", err)
		respondWithError(w, 401, errorMessage)
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		errorMessage := fmt.Sprintf("ERROR: %v\n", err)
		respondWithError(w, 401, errorMessage)
		return
	}

	validChirp := database.PostChirpParams{
		Body:   cleanWords(params.Body),
		UserID: userID,
	}
	ctx := context.Background()
	newChirp, err := cfg.dbQuerries.PostChirp(ctx, validChirp)
	if err != nil {
		errMessage := fmt.Sprintf("ERROR: %v", err)
		respondWithError(w, 500, errMessage)
	}
	actualChirp := Chirp{
		ID:        newChirp.UserID,
		CreatedAt: newChirp.CreatedAt,
		UpdatedAt: newChirp.UpdatedAt,
		Body:      newChirp.Body,
		UserID:    newChirp.UserID,
	}

	respondWithJSON(w, 201, actualChirp)
	return
}

func validateChirpLength(w http.ResponseWriter, r *http.Request) {
	// Request parameters
	type parameters struct {
		Body string `json:"body"`
	}

	type returnValue struct {
		CleanedBody string `json:"cleaned_body"`
		Error       string `json:"error,omitempty"`
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
