package systems

import (
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/tickable"
)

// FleetCommander provides methods for commanding fleets
type FleetCommander struct {
	gameData GameDataProvider
}

// NewFleetCommander creates a new fleet commander
func NewFleetCommander(gameData GameDataProvider) *FleetCommander {
	return &FleetCommander{
		gameData: gameData,
	}
}

// MoveFleetToSystem attempts to move all ships in a fleet to another system
// Returns (successCount, failCount) for how many ships successfully moved
func (fc *FleetCommander) MoveFleetToSystem(ships []*entities.Ship, targetSystemID int) (int, int) {
	if len(ships) == 0 {
		return 0, 0
	}

	// Create ship movement helper
	helper := tickable.NewShipMovementHelper(fc.gameData.GetSystemsMap(), fc.gameData.GetHyperlanes())

	successCount := 0
	failCount := 0

	// Attempt to move each ship
	for _, ship := range ships {
		if helper.StartJourney(ship, targetSystemID) {
			successCount++
		} else {
			failCount++
		}
	}

	return successCount, failCount
}

// MoveFleetToPlanet moves all ships to orbit a specific planet
func (fc *FleetCommander) MoveFleetToPlanet(ships []*entities.Ship, targetPlanet *entities.Planet, systems []*entities.System) (int, int) {
	if len(ships) == 0 || targetPlanet == nil {
		return 0, 0
	}

	successCount := 0
	failCount := 0

	// Find which system the planet is in
	planetSystemID := -1
	for _, system := range systems {
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
	for _, ship := range ships {
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

// MoveFleetToStar moves all ships to orbit the system's star
func (fc *FleetCommander) MoveFleetToStar(ships []*entities.Ship) (int, int) {
	if len(ships) == 0 {
		return 0, 0
	}

	successCount := 0
	failCount := 0

	// Get the system the fleet is in
	if len(ships) > 0 {
		systemID := ships[0].CurrentSystem
		systemsMap := fc.gameData.GetSystemsMap()
		system := systemsMap[systemID]

		if system != nil {
			// Move each ship to star orbit (small orbit distance)
			for _, ship := range ships {
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
func GetConnectedSystems(fromSystemID int, hyperlanes []entities.Hyperlane) []int {
	connected := make([]int, 0)

	for _, hyperlane := range hyperlanes {
		if hyperlane.From == fromSystemID {
			connected = append(connected, hyperlane.To)
		} else if hyperlane.To == fromSystemID {
			connected = append(connected, hyperlane.From)
		}
	}

	return connected
}
