//go:build js

package core

// getRemoteConstructionItems is a no-op on WASM (no remote play support).
func (a *App) getRemoteConstructionItems(_ string) []CachedConstructionItem {
	return nil
}
