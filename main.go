package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	infra_mode := os.Getenv("INFRA_MODE")
	token_secret := os.Getenv("TOKEN_SECRET")
	html_base_url := "https://storage.googleapis.com/antrein-ta/html_templates/{project_id}.html"
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
		host := r.Referer()
		projectID, err := extractProjectID(host)
		if err != nil {
			serveErrorHTML(w, "URL not registered")
			return
		}

		htmlURL := strings.Replace(html_base_url, "{project_id}", projectID, 1)
		fmt.Println(htmlURL)
		htmlContent, err := fetchHTMLContent(htmlURL)
		if err != nil {
			serveErrorHTML(w, "Failed to fetch HTML content")
			return
		}

		auth, err := r.Cookie("antrein_authorization")
		if err != nil || auth == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(htmlContent))
			return
		}

		if isValidToken(auth.Value, token_secret, projectID) {
			// Masuk ke reverse proxy
			proxy := httputil.NewSingleHostReverseProxy(proxyUrl)
			proxy.ServeHTTP(w, r)
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(htmlContent))
			return
		}
	})

	http.ListenAndServe(":8080", nil)
}

func extractProjectID(url string) (string, error) {
	re := regexp.MustCompile(`https?://([^.]+)\.antrein\.com`)
	matches := re.FindStringSubmatch(url)
	if len(matches) < 2 {
		return "", fmt.Errorf("URL not registered")
	}
	return matches[1], nil
}

func authorizationCheck(authToken, secret, projectID string) bool {
	token, err := jwt.Parse(authToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}

	if exp, ok := claims["exp"].(float64); ok {
		if time.Unix(int64(exp), 0).Before(time.Now()) {
			return false
		}
	} else {
		return false
	}

	tokenProjectID, ok := claims["project_id"].(string)
	if !ok {
		return false
	}

	if tokenProjectID != projectID {
		return false
	}

	return true
}

func isValidToken(token, secret, projectID string) bool {
	if authorizationCheck(token, secret, projectID) {
		return true
	} else {
		return false
	}
}

func fetchHTMLContent(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func serveErrorHTML(w http.ResponseWriter, message string) {
	htmlContent := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Unauthorized</title>
	</head>
	<body>
		<h1>%s</h1>
		<p>You do not have permission to access this page.</p>
	</body>
	</html>
	`, message)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(htmlContent))
}
