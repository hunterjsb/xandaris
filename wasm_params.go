//go:build js

package main

import "syscall/js"

// getWASMConnectParams reads ?server=&player=&key= from the browser URL.
func getWASMConnectParams() (serverURL, playerName, apiKey string) {
	url := js.Global().Get("URL").New(js.Global().Get("window").Get("location").Get("href"))
	params := url.Get("searchParams")
	serverURL = params.Call("get", "server").String()
	playerName = params.Call("get", "player").String()
	apiKey = params.Call("get", "key").String()

	if serverURL == "null" || serverURL == "<null>" {
		serverURL = ""
	}
	if playerName == "null" || playerName == "<null>" {
		playerName = ""
	}
	if apiKey == "null" || apiKey == "<null>" {
		apiKey = ""
	}
	return
}
