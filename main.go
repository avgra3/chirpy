package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/avgra3/chirpy/internal/database"
	"github.com/google/uuid"
	_ "github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Getting .env
	godotenv.Load()
	currentPlatform := os.Getenv("PLATFORM")
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	dbQuerries := database.New(db)

	// Setting up our server
	serverMux := http.NewServeMux()
	// Setting up our readiness endpoint
	serverMux.HandleFunc("GET /api/healthz", readiness)

	// Handle that adds a file server.
	// Fileserver uses the http.Dir to map the
	// current directory to http address.
	apiCfg := apiConfig{
		dbQuerries: dbQuerries,
		platform:   currentPlatform,
	}
	app := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	serverMux.Handle("/app/", apiCfg.middlewareMetricsInt(app))

	// Handle hits to the file server
	serverMux.HandleFunc("GET /admin/metrics", apiCfg.adminHandler)
	serverMux.HandleFunc("POST /admin/reset", apiCfg.resetCounter)
	// serverMux.HandleFunc("POST /api/validate_chirp", validateChirpLength)
	serverMux.HandleFunc("POST /api/users", apiCfg.newUserHandler)
	serverMux.HandleFunc("POST /api/chirps", apiCfg.newChirps)

	server := http.Server{
		Handler: serverMux,
		Addr:    ":8080",
	}
	log.Printf("Running server on port: %v", server.Addr)
	server.ListenAndServe()
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}
