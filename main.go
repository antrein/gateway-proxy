package main

import (
	"fmt"
	"io"
	"log"
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

	log.Printf("Starting reverse proxy from to %s", target)

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
			deleteCookieScript := `<script>
				function deleteAllCookies() {
					const cookies = document.cookie.split(";");
				
					for (let i = 0; i < cookies.length; i++) {
						const cookie = cookies[i];
						const eqPos = cookie.indexOf("=");
						const name = eqPos > -1 ? cookie.substr(0, eqPos) : cookie;
						document.cookie = name + "=;expires=Thu, 01 Jan 1970 00:00:00 GMT";
					}
				}

				deleteAllCookies()
			</script>
			`
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(htmlContent + deleteCookieScript))
			return
		}
	})

	http.ListenAndServe(":8080", nil)
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
	const cookies = document.cookie;
	const cookieMap = new Map(cookies.split('; ').map(cookie => cookie.split('=')));

	function hasCookie(name) {
		return cookieMap.has(name) && cookieMap.get(name) !== '';
	}

	function updateLastUpdated() {
		const now = new Date();
		const formatted = now.toLocaleTimeString('en-US', { hour: 'numeric', minute: 'numeric', second: 'numeric' });
		document.getElementById('lastUpdated').textContent = 'Last updated: ' + formatted;
	}

	function refreshFunction() {
		window.location.reload();
	}

    function formatDuration(minutes) {
        if (minutes < 0) {
            return "0 minutes"
        }

        const hours = Math.floor(minutes / 60);
        const remainingMinutes = minutes % 60;

        let result = "";

        if (hours > 0) {
            result += hours + " " + "hour" + (hours > 1 ? 's' : '')
            if (remainingMinutes > 0) {
                result += " ";
            }
        }

        if (remainingMinutes > 0) {
            result += remainingMinutes + " " + "minute" + (remainingMinutes > 1 ? 's' : '');
        }

        return result || "0 minutes";
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
				if (tokens.waiting_room_token !== "") {
					document.cookie = 'antrein_waiting_room=' + tokens.waiting_room_token + '; path=/; SameSite=Lax';
					setTimeout(() => {
						refreshFunction();
					}, 5000);
				}				
				if (tokens.main_room_token !== "") {
					document.cookie = 'antrein_authorization=' + tokens.main_room_token + '; path=/; SameSite=Lax';
					window.location.reload();
				}
			}
		} catch (e) {
			console.error('Error during registration:', e);
		}
	}

	if (!hasCookie('antrein_authorization') && !hasCookie('antrein_waiting_room')) {
		registerQueue();
	} else if (!hasCookie('antrein_authorization') && hasCookie('antrein_waiting_room')) {
        const token = cookieMap.get('antrein_waiting_room');
		const source = new EventSource("https://api.antrein.com/bc/queue/wr?token="+token);
        source.onmessage = function(event) {
            const data = JSON.parse(event.data);
            if (data){
                countdown.innerHTML = formatDuration(data.time_remaining)
                if (data.main_room_token && data.main_room_token != "" && data.is_finished) {
					document.cookie = 'antrein_authorization=' + data.main_room_token + '; path=/; SameSite=Lax';
                    window.location.reload();
                }
            }
        };
	}

	updateLastUpdated();

	window.onload = function() {
		setTimeout(() => {
			refreshFunction();
	
			setInterval(refreshFunction, 10000);
		}, 10000);
	};
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
