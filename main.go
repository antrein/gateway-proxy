package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	infra_mode := os.Getenv("INFRA_MODE")
	token_secret := os.Getenv("TOKEN_SECRET")
	projectID := os.Getenv("PROJECT_ID")
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
		htmlURL := strings.Replace(html_base_url, "{project_id}", projectID, 1)
		html, err := fetchHTMLContent(htmlURL)
		if err != nil {
			serveErrorHTML(w, "Failed to fetch HTML content")
			return
		}
		htmlContent := addScriptHTML(html, projectID)
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

	http.ListenAndServe(":9080", nil)
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

func loadDefaultHTML() (string, error) {
	filePath := "template.html"
	htmlFile, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer htmlFile.Close()

	content, err := io.ReadAll(htmlFile)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func addScriptHTML(htmlContent, projectID string) string {
	script := `
    <script>
	const url = window.location.href;
	console.log(url);
	const cookies = document.cookie;
	const cookieMap = new Map(cookies.split('; ').map(cookie => cookie.split('=')));

	function hasCookie(name) {
		return cookieMap.has(name) && cookieMap.get(name) !== '';
	}

	async function registerQueue() {
		try {
			const response = await fetch('https://api.antrein.com/bc/queue/register?project_id={project_id}');
			if (!response.ok) {
				throw new Error('HTTP error! status: ' + response.status);
			}
			const data = await response.json();
			if (data.status === 200) {
				const tokens = data.data;
				if (tokens.main_room_token !== "") {
					document.cookie = 'antrein_authorization=' + tokens.main_room_token + '; path=/; SameSite=Lax';
				}
				if (tokens.waiting_room_token !== "") {
					document.cookie = 'antrein_waiting_room=' + tokens.waiting_room_token + '; path=/; SameSite=Lax';
				}
				console.log('Cookies updated:', document.cookie);
			}
		} catch (e) {
			console.error('Error during registration:', e);
		}
	}

	function startCountdown(duration) {
		let timer = duration, minutes, seconds;
		const countdownElement = document.getElementById('countdown');
		const intervalId = setInterval(function () {
			minutes = parseInt(timer / 60, 10);
			seconds = parseInt(timer % 60, 10);

			minutes = minutes < 10 ? "0" + minutes : minutes;
			seconds = seconds < 10 ? "0" + seconds : seconds;

			countdownElement.textContent = minutes + ":" + seconds;

			if (--timer < 0) {
				clearInterval(intervalId);
				countdownElement.textContent = '00:00';
				window.location.reload();
			}
		}, 1000);
	}

	if (!hasCookie('antrein_authorization') && !hasCookie('antrein_waiting_room')) {
		registerQueue();
	} else if (!hasCookie('antrein_authorization') && hasCookie('antrein_waiting_room')) {
		startCountdown(30);
	}
    </script>`
	return htmlContent + strings.Replace(script, "{project_id}", projectID, 1)
}

func fetchHTMLContent(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	if strings.Contains(string(body), "The specified key does not exist.") {
		return loadDefaultHTML()
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
