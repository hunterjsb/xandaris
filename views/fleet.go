package views

import (
	"github.com/hunterjsb/xandaris/entities"
)

// FleetCommandInterface defines commands that can be issued to fleets
// This decouples the UI from game logic - UI issues commands, game executes them
type FleetCommandInterface interface {
	// MoveFleetToSystem attempts to move a fleet to another system
	// Returns (successCount, failCount) for how many ships moved
	MoveFleetToSystem(fleet *entities.Fleet, targetSystemID int) (int, int)

	// MoveFleetToPlanet moves fleet to orbit a specific planet
	MoveFleetToPlanet(fleet *entities.Fleet, targetPlanet *entities.Planet) (int, int)

	// MoveFleetToStar moves fleet to orbit the system's star
	MoveFleetToStar(fleet *entities.Fleet) (int, int)

	// GetConnectedSystems returns system IDs connected via hyperlanes
	GetConnectedSystems(fromSystemID int) []int

	// GetSystemByID returns a system by its ID
	GetSystemByID(systemID int) *entities.System
}
