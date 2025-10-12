package main

import (
	"fmt"
	"math"

	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/tickable"
)

// Fleet represents a group of ships at the same location
type Fleet struct {
	Ships         []*entities.Ship
	LeadShip      *entities.Ship // First ship in fleet, used for positioning
	SystemID      int
	OrbitDistance float64
	OrbitAngle    float64
	Owner         string
	IsSelected    bool
}

// NewFleet creates a new fleet from a list of ships
func NewFleet(ships []*entities.Ship) *Fleet {
	if len(ships) == 0 {
		return nil
	}

	fleet := &Fleet{
		Ships:         ships,
		LeadShip:      ships[0],
		SystemID:      ships[0].CurrentSystem,
		OrbitDistance: ships[0].GetOrbitDistance(),
		OrbitAngle:    ships[0].GetOrbitAngle(),
		Owner:         ships[0].Owner,
		IsSelected:    false,
	}

	return fleet
}

// AddShip adds a ship to the fleet
func (f *Fleet) AddShip(ship *entities.Ship) {
	f.Ships = append(f.Ships, ship)
}

// RemoveShip removes a ship from the fleet
func (f *Fleet) RemoveShip(ship *entities.Ship) {
	for i, s := range f.Ships {
		if s == ship {
			f.Ships = append(f.Ships[:i], f.Ships[i+1:]...)
			break
		}
	}

	// Update lead ship if necessary
	if f.LeadShip == ship && len(f.Ships) > 0 {
		f.LeadShip = f.Ships[0]
	}
}

// Size returns the number of ships in the fleet
func (f *Fleet) Size() int {
	return len(f.Ships)
}

// GetPosition returns the fleet's position (from lead ship)
func (f *Fleet) GetPosition() (float64, float64) {
	if f.LeadShip != nil {
		return f.LeadShip.GetAbsolutePosition()
	}
	return 0, 0
}

// GetTotalFuel returns the total fuel across all ships
func (f *Fleet) GetTotalFuel() int {
	total := 0
	for _, ship := range f.Ships {
		total += ship.CurrentFuel
	}
	return total
}

// GetTotalMaxFuel returns the total max fuel capacity
func (f *Fleet) GetTotalMaxFuel() int {
	total := 0
	for _, ship := range f.Ships {
		total += ship.MaxFuel
	}
	return total
}

// GetAverageFuelPercent returns the average fuel percentage
func (f *Fleet) GetAverageFuelPercent() float64 {
	if len(f.Ships) == 0 {
		return 0
	}
	total := 0.0
	for _, ship := range f.Ships {
		total += ship.GetFuelPercentage()
	}
	return total / float64(len(f.Ships))
}

// GetShipTypeCounts returns a map of ship type to count
func (f *Fleet) GetShipTypeCounts() map[entities.ShipType]int {
	counts := make(map[entities.ShipType]int)
	for _, ship := range f.Ships {
		counts[ship.ShipType]++
	}
	return counts
}

// GetDescription returns a description of the fleet
func (f *Fleet) GetDescription() string {
	if len(f.Ships) == 1 {
		return f.Ships[0].GetDescription()
	}

	typeCounts := f.GetShipTypeCounts()
	desc := fmt.Sprintf("Fleet (%d ships)", len(f.Ships))
	for shipType, count := range typeCounts {
		desc += fmt.Sprintf("\n  %dx %s", count, shipType)
	}
	return desc
}

// FleetManager handles fleet aggregation and management
type FleetManager struct {
	game *Game
}

// NewFleetManager creates a new fleet manager
func NewFleetManager(game *Game) *FleetManager {
	return &FleetManager{
		game: game,
	}
}

// AggregateFleets groups ships into fleets based on location
// Only shows ships that are orbiting the STAR (not orbiting planets)
func (fm *FleetManager) AggregateFleets(system *entities.System) []*Fleet {
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
	var fleets []*Fleet
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
		fleet := NewFleet(fleetShips)
		if fleet != nil {
			fleets = append(fleets, fleet)
		}
	}

	return fleets
}

// AggregateFleetsAtPlanet groups ships orbiting a specific planet
func (fm *FleetManager) AggregateFleetsAtPlanet(system *entities.System, planet *entities.Planet) []*Fleet {
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

	var fleets []*Fleet
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

		fleet := NewFleet(fleetShips)
		if fleet != nil {
			fleets = append(fleets, fleet)
		}
	}

	return fleets
}

// GetFleetAtPosition finds a fleet at a specific position
func (fm *FleetManager) GetFleetAtPosition(fleets []*Fleet, x, y int, radius float64) *Fleet {
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
func (fm *FleetManager) MoveFleet(fleet *Fleet, targetSystemID int) (successCount int, failCount int) {
	if fleet == nil || len(fleet.Ships) == 0 {
		return 0, 0
	}

	// Create ship movement helper
	helper := tickable.NewShipMovementHelper(fm.game.GetSystems(), fm.game.GetHyperlanes())

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
func (fm *FleetManager) CanFleetMove(fleet *Fleet, targetSystemID int) bool {
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
func (fm *FleetManager) GetFleetMovementStatus(fleet *Fleet) (canMove int, lowFuel int, noFuel int) {
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
