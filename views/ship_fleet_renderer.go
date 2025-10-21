package views

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/rendering"
	"github.com/hunterjsb/xandaris/utils"
)

// ShipFleetRenderer handles rendering of both individual ships and fleets
type ShipFleetRenderer struct {
	spriteRenderer *rendering.SpriteRenderer
	ctx            GameContext
}

// NewShipFleetRenderer creates a new ship/fleet renderer
func NewShipFleetRenderer(ctx GameContext, spriteRenderer *rendering.SpriteRenderer) *ShipFleetRenderer {
	return &ShipFleetRenderer{
		spriteRenderer: spriteRenderer,
		ctx:            ctx,
	}
}

// GetShipsAndFleetsInSystem returns individual ships and fleets orbiting the STAR (not planets)
func (sfr *ShipFleetRenderer) GetShipsAndFleetsInSystem(system *entities.System) ([]*entities.Ship, []*entities.Fleet) {
	if system == nil {
		return nil, nil
	}

	// First, find all planet orbits so we can filter them out
	planetOrbits := make(map[float64]bool)
	for _, entity := range system.Entities {
		if planet, ok := entity.(*entities.Planet); ok {
			planetOrbits[planet.GetOrbitDistance()] = true
		}
	}

	var ships []*entities.Ship
	var fleets []*entities.Fleet

	for _, entity := range system.Entities {
		switch e := entity.(type) {
		case *entities.Ship:
			// Only include ships NOT at planet orbits
			isAtPlanet := false
			for planetOrbit := range planetOrbits {
				if abs(e.GetOrbitDistance()-planetOrbit) < 1.0 {
					isAtPlanet = true
					break
				}
			}
			if !isAtPlanet {
				ships = append(ships, e)
			}
		case *entities.Fleet:
			// Only include fleets NOT at planet orbits
			if e.LeadShip != nil {
				isAtPlanet := false
				for planetOrbit := range planetOrbits {
					if abs(e.LeadShip.GetOrbitDistance()-planetOrbit) < 1.0 {
						isAtPlanet = true
						break
					}
				}
				if !isAtPlanet {
					fleets = append(fleets, e)
				}
			}
		}
	}

	return ships, fleets
}

// GetShipsAndFleetsAtPlanet returns ships and fleets orbiting a specific planet
func (sfr *ShipFleetRenderer) GetShipsAndFleetsAtPlanet(system *entities.System, planet *entities.Planet) ([]*entities.Ship, []*entities.Fleet) {
	if system == nil || planet == nil {
		return nil, nil
	}

	var ships []*entities.Ship
	var fleets []*entities.Fleet

	planetOrbit := planet.GetOrbitDistance()

	for _, entity := range system.Entities {
		switch e := entity.(type) {
		case *entities.Ship:
			// Check if ship is at this planet's orbit
			if abs(e.GetOrbitDistance()-planetOrbit) < 1.0 {
				ships = append(ships, e)
			}
		case *entities.Fleet:
			// Check if fleet is at this planet's orbit
			if e.LeadShip != nil && abs(e.LeadShip.GetOrbitDistance()-planetOrbit) < 1.0 {
				fleets = append(fleets, e)
			}
		}
	}

	return ships, fleets
}

// DrawShip renders an individual ship
func (sfr *ShipFleetRenderer) DrawShip(screen *ebiten.Image, ship *entities.Ship, size int) {
	if ship == nil {
		return
	}

	x, y := ship.GetAbsolutePosition()
	centerX := int(x)
	centerY := int(y)

	// Draw ownership ring if owned
	if ship.Owner != "" {
		if ownerColor, ok := sfr.getOwnerColor(ship.Owner); ok {
			DrawOwnershipRing(screen, centerX, centerY, float64(size+9), ownerColor)
		}
	}

	// Render ship sprite (using fleet renderer for single ship)
	sfr.spriteRenderer.RenderFleet(screen, centerX, centerY, size, ship.Color)

	// Draw ship name below
	DrawText(screen, ship.Name, centerX-30, centerY+size+5, utils.TextSecondary)
}

// DrawFleet renders a fleet
func (sfr *ShipFleetRenderer) DrawFleet(screen *ebiten.Image, fleet *entities.Fleet, size int) {
	if fleet == nil || len(fleet.Ships) == 0 {
		return
	}

	// Use lead ship's position
	x, y := fleet.GetAbsolutePosition()
	centerX := int(x)
	centerY := int(y)

	// Draw ownership ring
	if fleet.GetOwner() != "" {
		if ownerColor, ok := sfr.getOwnerColor(fleet.GetOwner()); ok {
			DrawOwnershipRing(screen, centerX, centerY, float64(size+9), ownerColor)
		}
	}

	// Render fleet with sprite renderer
	sfr.spriteRenderer.RenderFleet(screen, centerX, centerY, size, fleet.LeadShip.Color)

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

// DrawShipsAndFleets draws all ships and fleets from the provided lists
func (sfr *ShipFleetRenderer) DrawShipsAndFleets(screen *ebiten.Image, ships []*entities.Ship, fleets []*entities.Fleet, size int) {
	// Draw individual ships
	for _, ship := range ships {
		sfr.DrawShip(screen, ship, size)
	}

	// Draw fleets
	for _, fleet := range fleets {
		sfr.DrawFleet(screen, fleet, size)
	}
}

// GetShipOrFleetAtPosition finds a ship or fleet at a click position
func (sfr *ShipFleetRenderer) GetShipOrFleetAtPosition(ships []*entities.Ship, fleets []*entities.Fleet, x, y int, radius float64) (ship *entities.Ship, fleet *entities.Fleet) {
	// Check fleets first
	for _, f := range fleets {
		fx, fy := f.GetAbsolutePosition()
		dx := float64(x) - fx
		dy := float64(y) - fy
		distance := dx*dx + dy*dy

		if distance <= radius*radius {
			return nil, f
		}
	}

	// Check individual ships
	for _, s := range ships {
		sx, sy := s.GetAbsolutePosition()
		dx := float64(x) - sx
		dy := float64(y) - sy
		distance := dx*dx + dy*dy

		if distance <= radius*radius {
			return s, nil
		}
	}

	return nil, nil
}

// getOwnerColor gets the color for a player
func (sfr *ShipFleetRenderer) getOwnerColor(owner string) (c color.RGBA, ok bool) {
	if owner == "" {
		return color.RGBA{}, false
	}

	for _, player := range sfr.ctx.GetPlayers() {
		if player != nil && player.Name == owner {
			return player.Color, true
		}
	}

	return color.RGBA{}, false
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
