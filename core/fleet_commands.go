package core

import (
	"github.com/hunterjsb/xandaris/entities"
)

// Fleet command interface implementation - delegates to FleetCommandExecutor

// MoveFleetToSystem attempts to move all ships in a fleet to another system
func (a *App) MoveFleetToSystem(fleet *entities.Fleet, targetSystemID int) (int, int) {
	return a.fleetCmdExecutor.MoveFleetToSystem(fleet, targetSystemID)
}

// MoveFleetToPlanet moves all ships in a fleet to orbit a specific planet
func (a *App) MoveFleetToPlanet(fleet *entities.Fleet, targetPlanet *entities.Planet) (int, int) {
	return a.fleetCmdExecutor.MoveFleetToPlanet(fleet, targetPlanet)
}

// MoveFleetToStar moves all ships in a fleet to orbit the system's star
func (a *App) MoveFleetToStar(fleet *entities.Fleet) (int, int) {
	return a.fleetCmdExecutor.MoveFleetToStar(fleet)
}

// GetConnectedSystems returns system IDs connected to the given system via hyperlanes
func (a *App) GetConnectedSystems(fromSystemID int) []int {
	return a.fleetCmdExecutor.GetConnectedSystems(fromSystemID)
}

// GetSystemByID returns a system by its ID
func (a *App) GetSystemByID(systemID int) *entities.System {
	return a.fleetCmdExecutor.GetSystemByID(systemID)
}
