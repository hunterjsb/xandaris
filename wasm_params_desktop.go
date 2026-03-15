//go:build !js

package main

// getWASMConnectParams is a no-op on desktop (use --connect and --key flags).
func getWASMConnectParams() (serverURL, playerName, apiKey string) {
	return "", "", ""
}
