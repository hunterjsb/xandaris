package systems

import (
	"math"

	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/views"
)

// GameDataProvider defines the interface for accessing game data needed by fleet manager
type GameDataProvider interface {
	GetSystemsMap() map[int]*entities.System
	GetHyperlanes() []entities.Hyperlane
}

// FleetManager handles fleet aggregation and management
type FleetManager struct {
	gameData GameDataProvider
}

// NewFleetManager creates a new fleet manager
func NewFleetManager(gameData GameDataProvider) *FleetManager {
	return &FleetManager{
		gameData: gameData,
	}
}

// AggregateFleets groups ships into fleets based on location
// Only shows ships that are orbiting the STAR (not orbiting planets)
func (fm *FleetManager) AggregateFleets(system *entities.System) []*views.Fleet {
	if system == nil {
		return nil
	}

	// Group ships by location (within a threshold)
	const distanceThreshold = 5.0 // Ships within 5 units are considered same location

	// Find planets in this system to filter out planet-orbiting ships
	planetOrbits := make(map[float64]bool)
	for _, entity := range system.Entities {
		if planet, ok := entity.(*entities.Planet); ok {
			planetOrbits[planet.GetOrbitDistance()] = true
		}
	}

	var ships []*entities.Ship
	for _, entity := range system.Entities {
		if ship, ok := entity.(*entities.Ship); ok {
			// Only include ships that are NOT orbiting a planet
			// Ships orbiting planets have OrbitDistance matching a planet's orbit
			isOrbitingPlanet := false
			for planetOrbit := range planetOrbits {
				if math.Abs(ship.GetOrbitDistance()-planetOrbit) < 1.0 {
					isOrbitingPlanet = true
					break
				}
			}

			// Only add ships orbiting the star (not planets)
			if !isOrbitingPlanet {
				ships = append(ships, ship)
			}
		}
	}

	// Group ships into fleets
	var fleets []*views.Fleet
	used := make(map[*entities.Ship]bool)

	for _, ship := range ships {
		if used[ship] {
			continue
		}

		// Start a new fleet with this ship
		fleetShips := []*entities.Ship{ship}
		used[ship] = true

		// Find other ships at the same location
		for _, otherShip := range ships {
			if used[otherShip] || ship == otherShip {
				continue
			}

			// Check if ships are at similar positions
			dx := ship.GetOrbitDistance() - otherShip.GetOrbitDistance()
			dy := ship.GetOrbitAngle() - otherShip.GetOrbitAngle()
			distance := math.Sqrt(dx*dx + dy*dy)

			if distance < distanceThreshold {
				fleetShips = append(fleetShips, otherShip)
				used[otherShip] = true
			}
		}

		// Create fleet (even for single ships)
		fleet := views.NewFleet(fleetShips)
		if fleet != nil {
			fleets = append(fleets, fleet)
		}
	}

	return fleets
}

// AggregateFleetsAtPlanet groups ships orbiting a specific planet
func (fm *FleetManager) AggregateFleetsAtPlanet(system *entities.System, planet *entities.Planet) []*views.Fleet {
	if system == nil || planet == nil {
		return nil
	}

	// Find ships orbiting THIS specific planet
	var nearbyShips []*entities.Ship
	for _, entity := range system.Entities {
		if ship, ok := entity.(*entities.Ship); ok {
			// Check if ship is orbiting this specific planet
			// Ships orbit planets if their OrbitDistance matches exactly
			planetOrbit := planet.GetOrbitDistance()
			shipOrbit := ship.GetOrbitDistance()

			if math.Abs(planetOrbit-shipOrbit) < 1.0 {
				nearbyShips = append(nearbyShips, ship)
			}
		}
	}

	// Group ships by their relative angle to planet
	const angleThreshold = 0.3 // Radians

	var fleets []*views.Fleet
	used := make(map[*entities.Ship]bool)

	for _, ship := range nearbyShips {
		if used[ship] {
			continue
		}

		// Start a new fleet
		fleetShips := []*entities.Ship{ship}
		used[ship] = true
		shipAngle := ship.GetOrbitAngle() - planet.GetOrbitAngle()

		// Find other ships at similar angles
		for _, otherShip := range nearbyShips {
			if used[otherShip] {
				continue
			}

			otherAngle := otherShip.GetOrbitAngle() - planet.GetOrbitAngle()
			angleDiff := math.Abs(shipAngle - otherAngle)

			// Handle angle wrap-around
			if angleDiff > math.Pi {
				angleDiff = 2*math.Pi - angleDiff
			}

			if angleDiff < angleThreshold {
				fleetShips = append(fleetShips, otherShip)
				used[otherShip] = true
			}
		}

		fleet := views.NewFleet(fleetShips)
		if fleet != nil {
			fleets = append(fleets, fleet)
		}
	}

	return fleets
}

// GetFleetAtPosition finds a fleet at a specific position
func (fm *FleetManager) GetFleetAtPosition(fleets []*views.Fleet, x, y int, radius float64) *views.Fleet {
	for _, fleet := range fleets {
		fx, fy := fleet.GetPosition()
		dx := float64(x) - fx
		dy := float64(y) - fy
		distance := math.Sqrt(dx*dx + dy*dy)

		if distance <= radius {
			return fleet
		}
	}
	return nil
}

// MoveFleet orders all ships in a fleet to move to a target system
func (fm *FleetManager) MoveFleet(fleet *views.Fleet, targetSystemID int) (successCount int, failCount int) {
	if fleet == nil || len(fleet.Ships) == 0 {
		return 0, 0
	}

	// Create ship movement helper
	helper := tickable.NewShipMovementHelper(fm.gameData.GetSystemsMap(), fm.gameData.GetHyperlanes())

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

// CanFleetMove checks if the entire fleet can move to a target system
func (fm *FleetManager) CanFleetMove(fleet *views.Fleet, targetSystemID int) bool {
	if fleet == nil || len(fleet.Ships) == 0 {
		return false
	}

	// Check if any ship in the fleet can make the jump
	for _, ship := range fleet.Ships {
		if ship.CanJump() {
			return true
		}
	}

	return false
}

// GetFleetMovementStatus returns a summary of fleet movement capability
func (fm *FleetManager) GetFleetMovementStatus(fleet *views.Fleet) (canMove int, lowFuel int, noFuel int) {
	if fleet == nil {
		return 0, 0, 0
	}

	for _, ship := range fleet.Ships {
		if ship.CanJump() {
			canMove++
		} else if ship.CurrentFuel > 0 {
			lowFuel++
		} else {
			noFuel++
		}
	}

	return canMove, lowFuel, noFuel
}

// GetAllFleetsInSystem returns all fleets in a specific system
func (fm *FleetManager) GetAllFleetsInSystem(systemID int) []*views.Fleet {
	system := fm.gameData.GetSystemsMap()[systemID]
	if system == nil {
		return nil
	}
	return fm.AggregateFleets(system)
}
