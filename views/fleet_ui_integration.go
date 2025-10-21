package views

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/utils"
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

// updateFleets aggregates ships into fleets at this planet
func (f *fleetUIManager) updateFleets() {
	pv := f.planetView
	if pv.system == nil || pv.planet == nil {
		return
	}
	// Only aggregate ships that are actually at this planet's orbital distance
	fm := pv.ctx.GetFleetManager()
	pv.fleets = fm.AggregateFleetsAtPlanet(pv.system, pv.planet)
}

// drawFleets draws all fleets orbiting this planet
func (f *fleetUIManager) drawFleets(screen *ebiten.Image) {
	for _, fleet := range f.planetView.fleets {
		f.drawFleet(screen, fleet)
	}
}

// drawFleet draws a fleet of ships
func (f *fleetUIManager) drawFleet(screen *ebiten.Image, fleet *Fleet) {
	pv := f.planetView

	if fleet == nil || len(fleet.Ships) == 0 {
		return
	}

	// Use lead ship's position
	x, y := fleet.GetPosition()
	centerX := int(x)
	centerY := int(y)
	size := 6

	if fleet.Owner != "" {
		if ownerColor, ok := pv.getOwnerColor(fleet.Owner); ok {
			DrawOwnershipRing(screen, centerX, centerY, float64(size+9), ownerColor)
		}
	}

	// Try to load ship sprite
	shipType := string(fleet.LeadShip.ShipType)
	sprite, err := pv.spriteRenderer.GetAssetLoader().LoadShipSprite(shipType)
	if err == nil && sprite != nil {
		// Render sprite
		opts := &ebiten.DrawImageOptions{}

		// Scale to match desired size
		bounds := sprite.Bounds()
		spriteWidth := float64(bounds.Dx())
		spriteHeight := float64(bounds.Dy())
		scale := float64(size*2) / spriteWidth

		// Center and scale
		opts.GeoM.Translate(-spriteWidth/2, -spriteHeight/2)
		opts.GeoM.Scale(scale, scale)
		opts.GeoM.Translate(float64(centerX), float64(centerY))

		screen.DrawImage(sprite, opts)
	} else {
		// Fallback to triangle
		pv.spriteRenderer.RenderFleet(screen, centerX, centerY, size, fleet.LeadShip.Color)
	}

	// If multiple ships, draw count badge
	if fleet.Size() > 1 {
		badge := fmt.Sprintf("%d", fleet.Size())
		badgeX := centerX + size - 2
		badgeY := centerY - size - 8

		// Badge background
		badgePanel := &UIPanel{
			X:           badgeX - 2,
			Y:           badgeY - 2,
			Width:       12,
			Height:      12,
			BgColor:     utils.PanelBg,
			BorderColor: utils.PanelBorder,
		}
		badgePanel.Draw(screen)

		DrawText(screen, badge, badgeX, badgeY, utils.Highlight)
	}

	// Draw fleet info
	if fleet.Size() == 1 {
		DrawText(screen, fleet.Ships[0].Name, centerX-30, centerY+size+5, utils.TextSecondary)
	} else {
		fleetText := fmt.Sprintf("Fleet (%d ships)", fleet.Size())
		DrawText(screen, fleetText, centerX-40, centerY+size+5, utils.TextSecondary)
	}
}

// handleFleetClick checks if a fleet was clicked and shows the fleet info UI
func (f *fleetUIManager) handleFleetClick(x, y int) bool {
	pv := f.planetView

	// Check if clicking on a fleet
	fm := pv.ctx.GetFleetManager()
	clickedFleet := fm.GetFleetAtPosition(pv.fleets, x, y, 15.0)
	if clickedFleet != nil {
		if pv.fleetInfoUI != nil {
			pv.fleetInfoUI.ShowFleet(clickedFleet)
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
