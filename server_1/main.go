package main

import (
	"fmt"
	"log"
	"net/http"
)

var counter = 0

func healthHandler(w http.ResponseWriter, r *http.Request) {
	counter += 1
	if counter%2 == 0 {
		w.WriteHeader(http.StatusOK)
	} else {
		// simulate bad requests.
		w.WriteHeader(http.StatusBadGateway)
	}

}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello from the backend server 1!")
}

func main() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/health", healthHandler)

	port := ":8081" // Define the port to listen on
	fmt.Printf("Server starting at port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil)) // Start the server
}
