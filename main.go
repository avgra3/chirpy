package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/avgra3/chirpy/internal/database"
	_ "github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Getting .env
	godotenv.Load()
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
	}
	app := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	serverMux.Handle("/app/", apiCfg.middlewareMetricsInt(app))

	// Handle hits to the file server
	serverMux.HandleFunc("GET /admin/metrics", apiCfg.adminHandler)
	serverMux.HandleFunc("POST /admin/reset", apiCfg.restCounter)
	serverMux.HandleFunc("POST /api/validate_chirp", validateChirpLength)

	server := http.Server{
		Handler: serverMux,
		Addr:    ":8080",
	}
	log.Printf("Running server on port: %v", server.Addr)
	server.ListenAndServe()

}
