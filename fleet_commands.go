package main

import (
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/views"
)

// GetFleetCommander returns the fleet command interface (Game implements it)
func (g *Game) GetFleetCommander() views.FleetCommandInterface {
	return g
}

// MoveFleetToSystem attempts to move all ships in a fleet to another system
// Returns (successCount, failCount) for how many ships successfully moved
func (g *Game) MoveFleetToSystem(fleet *entities.Fleet, targetSystemID int) (int, int) {
	if fleet == nil || len(fleet.Ships) == 0 {
		return 0, 0
	}

	// Create ship movement helper
	helper := tickable.NewShipMovementHelper(g.GetSystemsMap(), g.GetHyperlanes())

	successCount := 0
	failCount := 0

	// Attempt to move each ship in the fleet
	for _, ship := range fleet.Ships {
		if helper.StartJourney(ship, targetSystemID) {
			successCount++
		} else {
			failCount++
		}
	}

	return successCount, failCount
}

// MoveFleetToPlanet moves all ships in a fleet to orbit a specific planet
func (g *Game) MoveFleetToPlanet(fleet *entities.Fleet, targetPlanet *entities.Planet) (int, int) {
	if fleet == nil || len(fleet.Ships) == 0 || targetPlanet == nil {
		return 0, 0
	}

	successCount := 0
	failCount := 0

	// Find which system the planet is in
	planetSystemID := -1
	for _, system := range g.systems {
		for _, entity := range system.Entities {
			if planet, ok := entity.(*entities.Planet); ok && planet.GetID() == targetPlanet.GetID() {
				planetSystemID = system.ID
				break
			}
		}
		if planetSystemID != -1 {
			break
		}
	}

	// Move each ship to the planet's orbit
	for _, ship := range fleet.Ships {
		// Only move ships that are in the same system as the planet
		if ship.CurrentSystem == planetSystemID {
			ship.OrbitDistance = targetPlanet.GetOrbitDistance()
			ship.OrbitAngle = targetPlanet.GetOrbitAngle()
			successCount++
		} else {
			failCount++
		}
	}

	return successCount, failCount
}

// MoveFleetToStar moves all ships in a fleet to orbit the system's star
func (g *Game) MoveFleetToStar(fleet *entities.Fleet) (int, int) {
	if fleet == nil || len(fleet.Ships) == 0 {
		return 0, 0
	}

	successCount := 0
	failCount := 0

	// Get the system the fleet is in
	if len(fleet.Ships) > 0 {
		systemID := fleet.Ships[0].CurrentSystem
		system := g.GetSystemsMap()[systemID]

		if system != nil {
			// Move each ship to star orbit (small orbit distance)
			for _, ship := range fleet.Ships {
				if ship.CurrentSystem == systemID {
					ship.OrbitDistance = 50.0              // Small distance from star
					ship.OrbitAngle = ship.GetOrbitAngle() // Keep current angle
					successCount++
				} else {
					failCount++
				}
			}
		}
	}

	return successCount, failCount
}

// GetConnectedSystems returns system IDs connected to the given system via hyperlanes
func (g *Game) GetConnectedSystems(fromSystemID int) []int {
	connected := make([]int, 0)

	for _, hyperlane := range g.hyperlanes {
		if hyperlane.From == fromSystemID {
			connected = append(connected, hyperlane.To)
		} else if hyperlane.To == fromSystemID {
			connected = append(connected, hyperlane.From)
		}
	}

	return connected
}

// GetSystemByID returns a system by its ID
func (g *Game) GetSystemByID(systemID int) *entities.System {
	return g.GetSystemsMap()[systemID]
}
