package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
)

func main() {
	infra_mode := os.Getenv("INFRA_MODE")
	token_secret := os.Getenv("TOKEN_SECRET")
	htmlContent := `
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Unauthorized</title>
    </head>
    <body>
        <h1>Unauthorized Access</h1>
        <p>You do not have permission to access this page.</p>
    </body>
    </html>
    `
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
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(htmlContent))

		}
		auth, err := r.Cookie("antrein_authorization")
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(htmlContent))
		}

		if isValidToken(auth.Value, token_secret, projectID) {
			// Masuk ke reverse proxy
			proxy := httputil.NewSingleHostReverseProxy(proxyUrl)
			proxy.ServeHTTP(w, r)
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(htmlContent))
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
	// token, err := jwt.Parse(authToken, func(token *jwt.Token) (interface{}, error) {
	// 	return []byte(secret), nil
	// })

	// if err != nil || !token.Valid {
	// 	return false
	// }

	// claims, ok := token.Claims.(jwt.MapClaims)
	// if !ok {
	// 	return false
	// }

	// if exp, ok := claims["exp"].(float64); ok {
	// 	if time.Unix(int64(exp), 0).Before(time.Now()) {
	// 		return false
	// 	}
	// } else {
	// 	return false
	// }

	// tokenProjectID, ok := claims["project_id"].(string)
	// if !ok {
	// 	return false
	// }

	// if tokenProjectID != projectID {
	// 	return false
	// }

	return authToken == "valid-token"
}

func isValidToken(token, secret, projectID string) bool {
	if authorizationCheck(token, secret, projectID) {
		return true
	} else {
		return false
	}
}
