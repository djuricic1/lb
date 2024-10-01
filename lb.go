package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

type Server struct {
	URL     *url.URL
	Healthy bool
}

var (
	// List of backend servers
	servers = []*Server{
		{URL: parseURL("http://localhost:8081"), Healthy: true},
		{URL: parseURL("http://localhost:8082"), Healthy: true},
	}

	// Counter for round-robin selection
	currentServer int32
)

// parseURL is a helper function to convert string to *url.URL
func parseURL(serverURL string) *url.URL {
	url, err := url.Parse(serverURL)
	if err != nil {
		log.Fatalf("Failed to parse URL %s: %v", serverURL, err)
	}
	return url
}

// getNextServer selects the next server in round-robin fashion
func getNextServer() *Server {
	for {
		serverIndex := atomic.AddInt32(&currentServer, 1)
		server := servers[int(serverIndex)%len(servers)]

		// Check if the selected server is healthy
		if server.Healthy {
			return server
		}
	}
}

// healthCheck pings the server's health check endpoint
func healthCheck(server *Server) {
	resp, err := http.Get(server.URL.String() + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		server.Healthy = false
	} else {
		server.Healthy = true
	}
}

// runHealthChecks starts a goroutine to periodically check the health of servers
func runHealthChecks() {
	for {
		for _, server := range servers {
			go healthCheck(server)
		}
		time.Sleep(1 * time.Second) // Check health every second
	}
}

func main() {
	// Start health checks in a goroutine
	go runHealthChecks()
	// Define the handler for the load balancer
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Log the request details
		logRequest(r)

		// Get the next server using round-robin
		backendServer := getNextServer()

		// Forward the request in a goroutine
		forwardRequest(w, r, backendServer.URL)
	}

	// Start the load balancer server
	http.HandleFunc("/", handler)
	port := ":8080"
	log.Printf("Load balancer listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func logRequest(r *http.Request) {
	remoteAddress := r.RemoteAddr
	method, request_uri, protocol := r.Method, r.RequestURI, r.Proto
	host := r.Host
	userAgent := r.UserAgent()
	acceptHeader := r.Header.Get("Accept")

	log.Printf("Received request from %s\n", remoteAddress)
	log.Println(method, request_uri, protocol)
	log.Printf("Host: %s\n", host)
	log.Printf("User-Agent: %s\n", userAgent)
	log.Printf("Accept: %s\n", acceptHeader)
}

// forwardRequest forwards an incoming HTTP request to a backend server.
// It takes the original request and response writer, along with the backend URL,
// and proxies the request to the backend server.
func forwardRequest(w http.ResponseWriter, r *http.Request, backendURL *url.URL) {
	// Create a new HTTP request based on the incoming request
	req, err := http.NewRequest(r.Method, backendURL.String()+r.RequestURI, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers from the original request to the new one
	req.Header = r.Header

	// Create a new HTTP client to send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to forward request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy the headers from the backend response
	for key, value := range resp.Header {
		w.Header()[key] = value
	}

	// Write the status code from the backend response
	w.WriteHeader(resp.StatusCode)

	// Copy the response body from the backend to the client
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, "Failed to copy response body", http.StatusInternalServerError)
	}
}
