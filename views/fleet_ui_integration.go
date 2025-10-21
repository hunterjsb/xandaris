package views

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// fleetUIManager handles fleet display and interaction in planet view
type fleetUIManager struct {
	planetView *PlanetView
}

// newFleetUIManager creates a new fleet UI manager
func newFleetUIManager(pv *PlanetView) *fleetUIManager {
	return &fleetUIManager{
		planetView: pv,
	}
}

// updateFleets is no longer needed - ships and fleets are already in the system
func (f *fleetUIManager) updateFleets() {
	// No-op: fleets are now persistent entities in the system
}

// drawFleets draws all ships and fleets orbiting this planet
func (f *fleetUIManager) drawFleets(screen *ebiten.Image) {
	pv := f.planetView
	if pv.system == nil || pv.planet == nil || pv.shipFleetRenderer == nil {
		return
	}

	// Get ships and fleets at this planet
	ships, fleets := pv.shipFleetRenderer.GetShipsAndFleetsAtPlanet(pv.system, pv.planet)

	// Draw them using centralized renderer
	pv.shipFleetRenderer.DrawShipsAndFleets(screen, ships, fleets, 6)
}

// handleFleetClick checks if a ship or fleet was clicked and shows the info UI
func (f *fleetUIManager) handleFleetClick(x, y int) bool {
	pv := f.planetView
	if pv.system == nil || pv.planet == nil || pv.shipFleetRenderer == nil {
		return false
	}

	// Get ships and fleets at this planet
	ships, fleets := pv.shipFleetRenderer.GetShipsAndFleetsAtPlanet(pv.system, pv.planet)

	// Check if clicking on a ship or fleet
	ship, fleet := pv.shipFleetRenderer.GetShipOrFleetAtPosition(ships, fleets, x, y, 15.0)

	if fleet != nil {
		if pv.fleetInfoUI != nil {
			pv.fleetInfoUI.ShowFleet(fleet)
		}
		return true
	}

	if ship != nil {
		if pv.fleetInfoUI != nil {
			pv.fleetInfoUI.ShowShip(ship)
		}
		return true
	}

	return false
}

// updateFleetInfoUI updates the fleet info UI if visible
func (f *fleetUIManager) updateFleetInfoUI() {
	pv := f.planetView

	if pv.fleetInfoUI != nil && pv.fleetInfoUI.IsVisible() {
		pv.fleetInfoUI.Update()
	}
}

// drawFleetInfoUI draws the fleet info UI if visible
func (f *fleetUIManager) drawFleetInfoUI(screen *ebiten.Image) {
	pv := f.planetView

	if pv.fleetInfoUI != nil && pv.fleetInfoUI.IsVisible() {
		pv.fleetInfoUI.Draw(screen)
	}
}

// closeFleetInfoUI closes the fleet info UI
func (f *fleetUIManager) closeFleetInfoUI() bool {
	pv := f.planetView

	if pv.fleetInfoUI != nil && pv.fleetInfoUI.IsVisible() {
		pv.fleetInfoUI.Hide()
		return true
	}

	return false
}
