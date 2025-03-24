package main

import (
	"log"
	"net/http"
)

func main() {
	serverMux := http.NewServeMux()
	// Setting up our readiness endpoint
	serverMux.HandleFunc("/healthz", readiness)

	// Handle that adds a file server.
	// Fileserver uses the http.Dir to map the
	// current directory to http address.
	serverMux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	server := http.Server{
		Handler: serverMux,
		Addr:    ":8080",
	}
	log.Printf("Running server on port: %v", server.Addr)
	server.ListenAndServe()

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
