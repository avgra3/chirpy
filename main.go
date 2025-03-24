package main

import (
	"log"
	"net/http"
)

func main() {
	serverMux := http.NewServeMux()
	// Handle that adds a file server.
	// Fileserver uses the http.Dir to map the
	// current directory to http address.
	serverMux.Handle("/", http.FileServer(http.Dir(".")))
	server := http.Server{
		Handler: serverMux,
		Addr:    ":8080",
	}
	log.Printf("Running server on port: %v", server.Addr)
	server.ListenAndServe()

}
