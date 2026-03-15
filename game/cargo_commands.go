package game

import (
	"fmt"
	"math"

	"github.com/hunterjsb/xandaris/entities"
)

// orbitTolerance is the distance threshold for considering a ship "at" a planet.
// Matches the tolerance used in fleet_info_ui.go isFleetAtPlanet().
const orbitTolerance = 1.0

// CargoCommandExecutor handles loading and unloading cargo between ships and planets.
type CargoCommandExecutor struct {
	systems []*entities.System
}

// NewCargoCommandExecutor creates a new cargo command executor.
func NewCargoCommandExecutor(systems []*entities.System) *CargoCommandExecutor {
	return &CargoCommandExecutor{systems: systems}
}

// LoadCargo transfers resources from a planet's storage to a ship's cargo hold.
// The ship must be orbiting the planet and the planet must be owned by the ship's owner.
// Returns the quantity actually loaded and any error.
func (cce *CargoCommandExecutor) LoadCargo(ship *entities.Ship, planet *entities.Planet, resource string, qty int) (int, error) {
	if ship == nil {
		return 0, fmt.Errorf("no ship specified")
	}
	if planet == nil {
		return 0, fmt.Errorf("no planet specified")
	}
	if qty <= 0 {
		return 0, fmt.Errorf("invalid quantity")
	}

	// Verify ship is orbiting this planet
	if !cce.isShipAtPlanet(ship, planet) {
		return 0, fmt.Errorf("ship %s is not orbiting %s", ship.Name, planet.Name)
	}

	// Verify ownership
	if planet.Owner != ship.Owner {
		return 0, fmt.Errorf("planet %s is not owned by %s", planet.Name, ship.Owner)
	}

	// Check planet has the resource
	stored := planet.StoredResources[resource]
	if stored == nil || stored.Amount <= 0 {
		return 0, fmt.Errorf("no %s available on %s", resource, planet.Name)
	}

	// Clamp to available stock
	actual := qty
	if actual > stored.Amount {
		actual = stored.Amount
	}

	// Load onto ship (AddCargo clamps to available space)
	loaded := ship.AddCargo(resource, actual)
	if loaded <= 0 {
		return 0, fmt.Errorf("ship cargo hold is full")
	}

	// Remove from planet
	planet.RemoveStoredResource(resource, loaded)

	fmt.Printf("[Cargo] Loaded %d %s onto %s from %s\n", loaded, resource, ship.Name, planet.Name)
	return loaded, nil
}

// UnloadCargo transfers resources from a ship's cargo hold to a planet's storage.
// The ship must be orbiting the planet and the planet must be owned by the ship's owner.
// Returns the quantity actually unloaded and any error.
func (cce *CargoCommandExecutor) UnloadCargo(ship *entities.Ship, planet *entities.Planet, resource string, qty int) (int, error) {
	if ship == nil {
		return 0, fmt.Errorf("no ship specified")
	}
	if planet == nil {
		return 0, fmt.Errorf("no planet specified")
	}
	if qty <= 0 {
		return 0, fmt.Errorf("invalid quantity")
	}

	// Verify ship is orbiting this planet
	if !cce.isShipAtPlanet(ship, planet) {
		return 0, fmt.Errorf("ship %s is not orbiting %s", ship.Name, planet.Name)
	}

	// Allow unloading at owned planets OR any planet with a Trading Post (foreign trade)
	if planet.Owner != ship.Owner {
		hasTradingPost := false
		for _, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok && b.BuildingType == "Trading Post" && b.IsOperational {
				hasTradingPost = true
				break
			}
		}
		if !hasTradingPost {
			return 0, fmt.Errorf("planet %s has no Trading Post for foreign trade", planet.Name)
		}
	}

	// Check ship has the resource
	if ship.CargoHold == nil || ship.CargoHold[resource] <= 0 {
		return 0, fmt.Errorf("no %s in ship cargo", resource)
	}

	// Clamp to available cargo
	actual := qty
	if actual > ship.CargoHold[resource] {
		actual = ship.CargoHold[resource]
	}

	// Remove from ship
	removed := ship.RemoveCargo(resource, actual)
	if removed <= 0 {
		return 0, fmt.Errorf("failed to remove cargo")
	}

	// Add to planet
	planet.AddStoredResource(resource, removed)

	fmt.Printf("[Cargo] Unloaded %d %s from %s to %s\n", removed, resource, ship.Name, planet.Name)
	return removed, nil
}

// isShipAtPlanet checks if a ship is orbiting a given planet (same system, matching orbit distance).
func (cce *CargoCommandExecutor) isShipAtPlanet(ship *entities.Ship, planet *entities.Planet) bool {
	// Find which system the planet is in
	for _, system := range cce.systems {
		for _, entity := range system.Entities {
			if p, ok := entity.(*entities.Planet); ok && p.GetID() == planet.GetID() {
				// Planet found in this system — check ship is here too
				if ship.CurrentSystem != system.ID {
					return false
				}
				return math.Abs(ship.GetOrbitDistance()-planet.GetOrbitDistance()) < orbitTolerance
			}
		}
	}
	return false
}

// FindPlanetByID looks up a planet by ID across all systems.
func (cce *CargoCommandExecutor) FindPlanetByID(planetID int) *entities.Planet {
	for _, system := range cce.systems {
		for _, entity := range system.Entities {
			if planet, ok := entity.(*entities.Planet); ok && planet.GetID() == planetID {
				return planet
			}
		}
	}
	return nil
}

// FindShipByID looks up a ship by ID across all players.
func FindShipByID(players []*entities.Player, shipID int) *entities.Ship {
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship != nil && ship.GetID() == shipID {
				return ship
			}
		}
	}
	return nil
}

// FindFleetByID looks up a fleet by ID across all players.
func FindFleetByID(players []*entities.Player, fleetID int) (*entities.Fleet, *entities.Player) {
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, fleet := range player.OwnedFleets {
			if fleet != nil && fleet.ID == fleetID {
				return fleet, player
			}
		}
	}
	return nil, nil
}

// GetSystemForPlanet returns the system ID that contains the given planet.
func (cce *CargoCommandExecutor) GetSystemForPlanet(planet *entities.Planet) int {
	for _, system := range cce.systems {
		for _, entity := range system.Entities {
			if p, ok := entity.(*entities.Planet); ok && p.GetID() == planet.GetID() {
				return system.ID
			}
		}
	}
	return -1
}
