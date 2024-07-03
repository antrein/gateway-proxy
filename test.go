package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var (
	originHost = "http://localhost:80" // Origin server
	targetHost = "https://gdplabs.id"  // Target server
)

func handleReverseProxy(target *url.URL) http.HandlerFunc {
	proxy := httputil.NewSingleHostReverseProxy(target)

	return func(w http.ResponseWriter, r *http.Request) {
		// Modify the request URL to the target
		r.URL.Host = target.Host
		r.URL.Scheme = target.Scheme
		r.Header.Set("X-Forwarded-Host", r.Host)
		r.Host = target.Host

		// Log the request
		log.Printf("Proxying request: %s to %s", originHost, targetHost)

		// Serve the request
		proxy.ServeHTTP(w, r)
	}
}

func main() {
	// Parse the target URL
	targetURL, err := url.Parse(targetHost)
	if err != nil {
		log.Fatalf("Failed to parse target URL: %v", err)
	}

	// Handle all requests with the reverse proxy handler
	http.HandleFunc("/", handleReverseProxy(targetURL))

	log.Printf("Starting reverse proxy server on %s, proxying to %s", originHost, targetHost)

	// Start the server
	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
