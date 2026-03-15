//go:build js

package ui

import "github.com/hunterjsb/xandaris/entities"

// updateRemote is a no-op on WASM (no net/http available).
func (p *PlanetDataProvider) updateRemote() {}

// populateRemoteEntities is a no-op on WASM.
func (p *PlanetDataProvider) populateRemoteEntities(_ *entities.Planet) bool { return false }
