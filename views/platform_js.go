//go:build js

package views

import "syscall/js"

const defaultServerURL = "https://api.xandaris.space"

// platformServerURL returns the game server URL.
// On WASM, checks for a ?server= URL param, then falls back to default.
func platformServerURL() string {
	url := js.Global().Get("URL").New(js.Global().Get("window").Get("location").Get("href"))
	server := url.Get("searchParams").Call("get", "server").String()
	if server != "" && server != "null" && server != "<null>" {
		return server
	}
	return defaultServerURL
}

// platformGetEnv is a no-op on WASM (no env vars in browser).
func platformGetEnv(key string) string {
	return ""
}

// platformOpenURL opens a URL in the browser (WASM: redirect the current page).
func platformOpenURL(url string) {
	js.Global().Get("window").Get("location").Call("assign", url)
}

// platformLoadSession loads stored credentials from localStorage.
func platformLoadSession() (name, apiKey string, playerID int) {
	storage := js.Global().Get("localStorage")
	name = storage.Call("getItem", "xn_name").String()
	apiKey = storage.Call("getItem", "xn_api_key").String()
	pidStr := storage.Call("getItem", "xn_player_id").String()

	if name == "null" || name == "<null>" {
		name = ""
	}
	if apiKey == "null" || apiKey == "<null>" {
		apiKey = ""
	}

	if pidStr != "null" && pidStr != "<null>" && pidStr != "" {
		for _, c := range pidStr {
			if c >= '0' && c <= '9' {
				playerID = playerID*10 + int(c-'0')
			}
		}
	}
	return
}

// platformSaveSession saves credentials to localStorage.
func platformSaveSession(name, apiKey string, playerID int) {
	storage := js.Global().Get("localStorage")
	storage.Call("setItem", "xn_name", name)
	storage.Call("setItem", "xn_api_key", apiKey)
	storage.Call("setItem", "xn_player_id", js.ValueOf(playerID).String())
}

// platformClearSession removes stored credentials.
func platformClearSession() {
	storage := js.Global().Get("localStorage")
	storage.Call("removeItem", "xn_name")
	storage.Call("removeItem", "xn_api_key")
	storage.Call("removeItem", "xn_player_id")
}

// platformStartOAuthListener redirects the browser to Discord OAuth on WASM.
func platformStartOAuthListener(onSuccess func(name, apiKey string, playerID int)) {
	platformOpenURL(platformServerURL() + "/api/auth/discord")
}
