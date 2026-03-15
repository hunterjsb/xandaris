//go:build !js

package views

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

type sessionData struct {
	Name     string `json:"name"`
	APIKey   string `json:"api_key"`
	PlayerID int    `json:"player_id"`
}

const defaultServerURL = "https://api.xandaris.space"

// platformServerURL returns the game server URL (from BASE_URL env var, or default).
func platformServerURL() string {
	if env := os.Getenv("BASE_URL"); env != "" {
		return env
	}
	return defaultServerURL
}

// platformGetEnv reads an environment variable.
func platformGetEnv(key string) string {
	return os.Getenv(key)
}

func sessionPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".xandaris", "session.json")
}

// platformOpenURL opens a URL in the user's default browser.
func platformOpenURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}

// platformLoadSession loads stored credentials from ~/.xandaris/session.json.
func platformLoadSession() (name, apiKey string, playerID int) {
	data, err := os.ReadFile(sessionPath())
	if err != nil {
		return
	}
	var s sessionData
	if json.Unmarshal(data, &s) == nil {
		name = s.Name
		apiKey = s.APIKey
		playerID = s.PlayerID
	}
	return
}

// platformSaveSession saves credentials to ~/.xandaris/session.json.
func platformSaveSession(name, apiKey string, playerID int) {
	s := sessionData{Name: name, APIKey: apiKey, PlayerID: playerID}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(sessionPath()), 0700)
	os.WriteFile(sessionPath(), data, 0600)
}

// platformClearSession removes the stored session file.
func platformClearSession() {
	os.Remove(sessionPath())
}

// platformStartOAuthListener starts a temporary HTTP server to capture the OAuth callback.
// It opens the browser to the Discord OAuth URL and waits for the redirect.
func platformStartOAuthListener(onSuccess func(name, apiKey string, playerID int)) {
	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("[Auth] Failed to start OAuth listener: %v\n", err)
		return
	}
	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://localhost:%d/callback", port)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Read credentials from query params (server redirects here with ?key=&name=&player_id=)
		key := r.URL.Query().Get("key")
		name := r.URL.Query().Get("name")
		pidStr := r.URL.Query().Get("player_id")
		pid, _ := strconv.Atoi(pidStr)

		if key != "" && name != "" {
			platformSaveSession(name, key, pid)

			// Show success page
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<!DOCTYPE html><html><head><style>
				body{background:#0a0c14;color:#c0c8d8;font-family:'Courier New',monospace;display:flex;justify-content:center;align-items:center;height:100vh}
				.box{text-align:center}.ok{color:#7fdbca;font-size:2em}p{color:#667}
			</style></head><body><div class="box"><div class="ok">Signed in as %s</div><p>You can close this tab and return to the game.</p></div></body></html>`, name)

			onSuccess(name, key, pid)
		} else {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<!DOCTYPE html><html><head><style>
				body{background:#0a0c14;color:#c44;font-family:'Courier New',monospace;display:flex;justify-content:center;align-items:center;height:100vh}
			</style></head><body>Sign-in failed. Please try again.</body></html>`)
		}

		// Shut down after handling
		go srv.Close()
	})

	go func() {
		srv.Serve(listener)
	}()

	oauthURL := fmt.Sprintf("%s/api/auth/discord?local_callback=%s", platformServerURL(), callbackURL)
	fmt.Printf("[Auth] Opening browser for Discord sign-in (callback on port %d)...\n", port)
	platformOpenURL(oauthURL)
}
