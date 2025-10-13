package views

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
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

// FleetManagerInterface defines operations for managing fleets
type FleetManagerInterface interface {
	AggregateFleets(system *entities.System) []*Fleet
	AggregateFleetsAtPlanet(system *entities.System, planet *entities.Planet) []*Fleet
	GetAllFleetsInSystem(systemID int) []*Fleet
	GetFleetAtPosition(fleets []*Fleet, x, y int, radius float64) *Fleet
}
