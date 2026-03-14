//go:build js

package api

// StartServer is a no-op on WASM builds.
func StartServer(provider GameStateProvider) {}
