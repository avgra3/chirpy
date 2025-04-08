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
		Password string `json:"password"`
		Email    string `json:"email"`
		//	ExpiresInSeconds int    `json:"expires_in_secods"`
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
	// if userByEmail.ExpiresInSeconds <= 0 || userByEmail.ExpiresInSeconds > 60*60 {
	// 	// If not provided or greater than 1 hour, will default to 1 hour, in seconds
	// 	userByEmail.ExpiresInSeconds = 60 * 60
	// }

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
	jwtDuration := time.Duration(60*60) * time.Second
	jwt, err := auth.MakeJWT(user.ID, cfg.jwtSecret, jwtDuration)
	if err != nil {
		respondWithError(respWriter, 500, "Unable to create token at this time")
		return
	}

	refreshToken, _ := auth.MakeRefreshToken()

	authUser := User{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        jwt,
		RefreshToken: refreshToken,
		IsChirpyRed:  user.IsChirpyRed.Bool,
	}

	// Makes new Refresh token in the database
	ctx = context.Background()
	_, err = cfg.dbQuerries.NewRereshToken(ctx, database.NewRereshTokenParams{Token: refreshToken, UserID: authUser.ID})
	if err != nil {
		respondWithError(respWriter, 401, "Unable to refresh token")
		return
	}

	respondWithJSON(respWriter, 200, authUser)
	return

}

func (cfg *apiConfig) refreshToken(respWriter http.ResponseWriter, req *http.Request) {
	// Requires no body
	// Requires header: "Authorization: Bearer <token>"
	head := req.Header
	refreshToken, err := auth.GetBearerToken(head)
	// If token does not exists or expired in the DB:
	//	Respond with 401
	if err != nil {
		respondWithError(respWriter, 401, "Does not exist")
		return
	}
	ctx := context.Background()
	// We have refresh token, we want to check if it is still valid
	userByToken, err := cfg.dbQuerries.GetUserFromRefreshToken(ctx, refreshToken)
	if err != nil {
		respondWithError(respWriter, 401, "Does not exist")
		return

	}
	// If the token has a revoked at date (meaning now invalid)
	if userByToken.RevokedAt.Valid {
		respondWithError(respWriter, 401, "Refresh token revoked")
		return
	}
	// Make new token
	jwtDuration, _ := time.ParseDuration("1h")
	newToken, err := auth.MakeJWT(userByToken.UserID, cfg.jwtSecret, jwtDuration)

	// We are good to go!
	type responseValue struct {
		Token string `json:"token"`
	}
	respondWithJSON(respWriter, 200, responseValue{Token: newToken})
}

func (cfg *apiConfig) updateEmailPassword(respWriter http.ResponseWriter, req *http.Request) {
	// Expect header to contain an access token
	accessToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(respWriter, 401, "Bad access token")
		return
	}
	// Validate accessToken
	userID, err := auth.ValidateJWT(accessToken, cfg.jwtSecret)
	if err != nil {
		respondWithError(respWriter, 401, "Bad access token")
		return
	}

	type emailPassRequestBody struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	defer req.Body.Close()
	data, err := io.ReadAll(req.Body)
	if err != nil {
		respondWithError(respWriter, 500, "couldn't read request")
		return
	}
	params := emailPassRequestBody{}
	err = json.Unmarshal(data, &params)
	if err != nil {
		respondWithError(respWriter, 500, "couldn't unmarshal request body")
		return
	}
	// Need to hash the password
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(respWriter, 500, "couldn't has password")
		return
	}
	// Need to update the users table with new email and password
	updates := database.UpdateUserEmailPasswordParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
		ID:             userID,
	}
	// Make the update
	ctx := context.Background()
	updatedUser, err := cfg.dbQuerries.UpdateUserEmailPassword(ctx, updates)
	if err != nil {
		respondWithError(respWriter, 500, "Unable to complete request")
		return
	}

	respondWithJSON(respWriter, 200, updatedUser)

}

func (cfg *apiConfig) revokeToken(respWriter http.ResponseWriter, req *http.Request) {
	// No body accepted
	// Requires a refreshToken in header: "Athorization: Bearer <token>" fromat
	refreshToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(respWriter, 500, "Invalid header")
		return
	}
	// Need to update revoked_at to the current timestamp (which also updates the updated_at field)
	ctx := context.Background()
	err = cfg.dbQuerries.RevokeRefreshToken(ctx, refreshToken)
	if err != nil {
		respondWithError(respWriter, 500, "An error occured")
		return
	}
	// Respond with a 204 status code -- no body returned
	respondWithJSON(respWriter, 204, "")
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
		return
	}
	// Need to get a JWT
	// Need to get refresh token

	ourUser := User{
		ID:             newUser.ID,
		CreatedAt:      newUser.CreatedAt,
		UpdatedAt:      newUser.UpdatedAt,
		Email:          newUser.Email,
		HashedPassword: hashedPasword,
		IsChirpyRed:    newUser.IsChirpyRed.Bool,
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
	// See if the user wants to filter by user id
	authorIDParam := r.URL.Query().Get("author_id")
	ctx := context.Background()

	if authorIDParam == "" {
		chirps, err := cfg.dbQuerries.GetChirps(ctx)
		if err != nil {
			errMessage := fmt.Sprintf("ERROR: %v", err)
			respondWithError(w, 500, errMessage)
		}

		respondWithJSON(w, 200, chirps)
		return
	}
	authorUUID, err := uuid.Parse(authorIDParam)
	if err != nil {
		respondWithError(w, 500, "Unable to parse author_id")
		return
	}

	chirps, err := cfg.dbQuerries.GetChirpsByAuthorID(ctx, authorUUID)
	if err != nil {
		respondWithError(w, 500, "There was a problem trying to get chirps.")
		return
	}
	respondWithJSON(w, 200, chirps)
}

func (cfg *apiConfig) getChirp(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	chirpIDStr := r.PathValue("chirpID")
	if chirpIDStr == "" {
		respondWithError(w, 400, "Missing chirp ID")
		return
	}
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		errMessage := fmt.Sprintf("ERROR: %v", err)
		respondWithError(w, 500, errMessage)

	}
	chirps, err := cfg.dbQuerries.GetChirpByChirpID(ctx, chirpID)
	if err != nil {
		errMessage := fmt.Sprintf("ERROR: %v", err)
		respondWithError(w, 404, errMessage)
	}

	respondWithJSON(w, 200, chirps)
	return
}

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, r *http.Request) {
	// Check token in header
	log.Printf("HEADER: %v", r.Header)
	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Bad access token")
		return
	}
	// The user ID is giving the chirp ID
	userID, err := auth.ValidateJWT(accessToken, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, 401, "Bad access token")
		return
	}
	// Chirp ID to delete
	chirpIDStr := r.PathValue("chirpID")
	if chirpIDStr == "" {
		respondWithError(w, 400, "Missing chirp id")
		return
	}
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		respondWithError(w, 401, "Bad chirp ID")
		return
	}

	// Only allow deletion if the user is the owner of the chirp
	ctx := context.Background()
	deleteChirpParams := database.DeleteChirpParams{ID: chirpID, UserID: userID}
	affectedRows, err := cfg.dbQuerries.DeleteChirp(ctx, deleteChirpParams)
	if err != nil {
		log.Printf("ERROR: %v", err)
		respondWithError(w, 500, "Server error!")
		return
	}
	// Check no rows affected
	if affectedRows == 0 {
		// Try to get the chirp, regardless of owner
		chirp, err := cfg.dbQuerries.GetChirp(ctx, userID)
		if err != nil || chirp == (database.Chirp{}) {
			respondWithError(w, 403, "Chirp not found")
		} else if chirp.UserID != userID {
			respondWithError(w, 403, "You don't own this chirp!")

		} else {
			// log.Printf("ERROR: chirp.ID => %v", chirp.ID)
			// log.Printf("ERROR: chirp.UserID => %v", chirp.UserID)
			// log.Printf("ERROR: chirp id we have => %v", chirpID)
			// log.Printf("ERROR: user id we have => %v", userID)
			// log.Printf("ERROR: chirp.ID == chirpID: %v", chirp.ID == chirpID)
			// log.Printf("ERROR: chirp.UserID == userID: %v", chirp.UserID == userID)
			// log.Printf("ERROR: len(chirp.ID) = %v", len(chirp.UserID))
			// log.Printf("ERROR: len(userID) = %v", len(userID))
			// log.Printf("ERROR: chirpIDStr value = %v", chirpIDStr)
			// log.Printf("ERROR: Requested Path: %s", r.URL.Path)
			// log.Printf("ERROR: Requested Raw Path: %s", r.URL.RawPath)
			//
			// log.Printf("ERROR: %v", err)
			respondWithError(w, 403, "We dont know what the error is!")
		}
		return
	}

	message := fmt.Sprintf("Successfully (hopefully) deleted CHIRP ID: %v \n", chirpID)
	log.Println(message)

	// We successfully deleted!
	w.WriteHeader(204)
}

func (cfg *apiConfig) upgradeUserToRed(w http.ResponseWriter, r *http.Request) {
	_, err := auth.GetAPIKey(r.Header)
	if err != nil {
		w.WriteHeader(401)
		return
	}

	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}
	defer r.Body.Close()

	data, err := io.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, 404, "Unable to read body")
		return
	}
	params := parameters{}
	err = json.Unmarshal(data, &params)
	if err != nil {
		respondWithError(w, 500, "couldn't unmarshal parameters")
		return
	}
	if params.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}
	// Need to update user in the database
	ctx := context.Background()
	// Double check user exists
	userCheck, err := cfg.dbQuerries.GetUserById(ctx, params.Data.UserID)
	if userCheck.ID != params.Data.UserID {
		respondWithError(w, 404, "User does not exist")
	}

	err = cfg.dbQuerries.UpgradeUserToChirpyRed(ctx, params.Data.UserID)
	if err != nil {
		respondWithError(w, 500, "Unable to complete upgrade")
		return
	}
	// Should have been successful, return 204
	w.WriteHeader(204)
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
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		errorMessage := fmt.Sprintf("Error: %v", err)
		respondWithError(w, 401, errorMessage)
		return
	}
	// The token we have is a refresh token -- we need to get the access token
	// We have the userID, so we need to get the access token
	userID, err := auth.ValidateJWT(refreshToken, cfg.jwtSecret)
	if err != nil {
		errorMessage := fmt.Sprintf("TOKEN: %v", refreshToken)
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
		ID:        newChirp.ID,
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
