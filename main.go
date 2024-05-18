package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

func main() {
	infra_mode := os.Getenv("INFRA_MODE")

	var target string

	if infra_mode == "multi" {
		target = os.Getenv("SERVICE_URL")
	} else {
		target = os.Getenv("SERVICE_URL") // ubah jadi get url data dari db
	}
	proxyUrl, err := url.Parse(target)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if isValidToken(r.Header.Get("X-Access-Token")) {
			// Masuk ke reverse proxy
			proxy := httputil.NewSingleHostReverseProxy(proxyUrl)
			proxy.ServeHTTP(w, r)
		} else {
			// Kelempar
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
		}
	})

	http.ListenAndServe(":8080", nil)
}

// isValidToken checks if the provided access token is valid
func isValidToken(token string) bool {
	// Logic checker di sini
	if token == "valid-token" {
		return true
	} else {
		return false
	}
}
