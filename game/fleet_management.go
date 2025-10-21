package game

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

// FleetManagementSystem handles explicit fleet creation and management
type FleetManagementSystem struct {
	state      *State
	nextFleetID int
}

// NewFleetManagementSystem creates a new fleet management system
func NewFleetManagementSystem(state *State) *FleetManagementSystem {
	return &FleetManagementSystem{
		state:      state,
		nextFleetID: 10000, // Start fleet IDs at 10000
	}
}

// CreateFleetFromShip promotes a ship to a fleet (fleet of 1)
func (fms *FleetManagementSystem) CreateFleetFromShip(ship *entities.Ship, owner *entities.Player) (*entities.Fleet, error) {
	if ship == nil {
		return nil, fmt.Errorf("ship cannot be nil")
	}
	if owner == nil {
		return nil, fmt.Errorf("owner cannot be nil")
	}
	if ship.Owner != owner.Name {
		return nil, fmt.Errorf("ship is not owned by this player")
	}

	// Find the system containing this ship
	var system *entities.System
	for _, sys := range fms.state.Systems {
		for _, entity := range sys.Entities {
			if s, ok := entity.(*entities.Ship); ok && s == ship {
				system = sys
				break
			}
		}
		if system != nil {
			break
		}
	}

	if system == nil {
		return nil, fmt.Errorf("ship not found in any system")
	}

	// Create the fleet
	fleet := entities.NewFleet(fms.nextFleetID, []*entities.Ship{ship})
	fms.nextFleetID++

	// Remove ship from system entities
	system.RemoveEntity(ship.ID)

	// Add fleet to system entities
	system.AddEntity(fleet)

	// Update player ownership
	owner.RemoveOwnedShip(ship)
	owner.AddOwnedFleet(fleet)

	fmt.Printf("[FleetManagement] Created fleet %d from ship %s\n", fleet.ID, ship.Name)
	return fleet, nil
}

// AddShipToFleet adds a ship to an existing fleet
func (fms *FleetManagementSystem) AddShipToFleet(ship *entities.Ship, fleet *entities.Fleet, owner *entities.Player) error {
	if ship == nil || fleet == nil || owner == nil {
		return fmt.Errorf("ship, fleet, and owner cannot be nil")
	}
	if ship.Owner != owner.Name {
		return fmt.Errorf("ship is not owned by this player")
	}
	if fleet.GetOwner() != owner.Name {
		return fmt.Errorf("fleet is not owned by this player")
	}

	// Verify ship and fleet are in the same system
	shipSystemID := ship.CurrentSystem
	fleetSystemID := fleet.GetSystemID()
	if shipSystemID != fleetSystemID {
		return fmt.Errorf("ship and fleet must be in the same system")
	}

	// Verify ship and fleet are at similar orbits (same planet or star)
	orbitDiff := ship.GetOrbitDistance() - fleet.LeadShip.GetOrbitDistance()
	if orbitDiff < -10.0 || orbitDiff > 10.0 {
		return fmt.Errorf("ship must be at a similar orbit to join the fleet")
	}

	// Find the system containing the ship
	var system *entities.System
	for _, sys := range fms.state.Systems {
		if sys.ID == shipSystemID {
			system = sys
			break
		}
	}

	if system == nil {
		return fmt.Errorf("system not found")
	}

	// Remove ship from system entities
	system.RemoveEntity(ship.ID)

	// Add ship to fleet
	fleet.AddShip(ship)

	// Update player ownership
	owner.RemoveOwnedShip(ship)

	fmt.Printf("[FleetManagement] Added ship %s to fleet %d\n", ship.Name, fleet.ID)
	return nil
}

// RemoveShipFromFleet removes a ship from a fleet
func (fms *FleetManagementSystem) RemoveShipFromFleet(ship *entities.Ship, fleet *entities.Fleet, owner *entities.Player) error {
	if ship == nil || fleet == nil || owner == nil {
		return fmt.Errorf("ship, fleet, and owner cannot be nil")
	}
	if ship.Owner != owner.Name {
		return fmt.Errorf("ship is not owned by this player")
	}
	if fleet.GetOwner() != owner.Name {
		return fmt.Errorf("fleet is not owned by this player")
	}

	// Verify ship is in the fleet
	found := false
	for _, s := range fleet.Ships {
		if s == ship {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("ship is not in this fleet")
	}

	// Find the system containing the fleet
	var system *entities.System
	for _, sys := range fms.state.Systems {
		for _, entity := range sys.Entities {
			if f, ok := entity.(*entities.Fleet); ok && f == fleet {
				system = sys
				break
			}
		}
		if system != nil {
			break
		}
	}

	if system == nil {
		return fmt.Errorf("fleet not found in any system")
	}

	// Remove ship from fleet
	fleet.RemoveShip(ship)

	// Add ship back to system as individual entity
	system.AddEntity(ship)

	// Update player ownership
	owner.AddOwnedShip(ship)

	fmt.Printf("[FleetManagement] Removed ship %s from fleet %d\n", ship.Name, fleet.ID)

	// If fleet is now empty or has only 1 ship, disband it
	if len(fleet.Ships) <= 1 {
		return fms.DisbandFleet(fleet, owner)
	}

	return nil
}

// DisbandFleet breaks up a fleet back into individual ships
func (fms *FleetManagementSystem) DisbandFleet(fleet *entities.Fleet, owner *entities.Player) error {
	if fleet == nil || owner == nil {
		return fmt.Errorf("fleet and owner cannot be nil")
	}
	if fleet.GetOwner() != owner.Name {
		return fmt.Errorf("fleet is not owned by this player")
	}

	// Find the system containing the fleet
	var system *entities.System
	for _, sys := range fms.state.Systems {
		for _, entity := range sys.Entities {
			if f, ok := entity.(*entities.Fleet); ok && f == fleet {
				system = sys
				break
			}
		}
		if system != nil {
			break
		}
	}

	if system == nil {
		return fmt.Errorf("fleet not found in any system")
	}

	// Remove fleet from system
	system.RemoveEntity(fleet.ID)

	// Add all ships back to system as individual entities
	for _, ship := range fleet.Ships {
		system.AddEntity(ship)
		owner.AddOwnedShip(ship)
	}

	// Remove fleet from player ownership
	owner.RemoveOwnedFleet(fleet)

	fmt.Printf("[FleetManagement] Disbanded fleet %d into %d ships\n", fleet.ID, len(fleet.Ships))
	return nil
}

// GetNearbyFleets finds fleets near a ship (within same orbit range)
func (fms *FleetManagementSystem) GetNearbyFleets(ship *entities.Ship, owner *entities.Player) []*entities.Fleet {
	if ship == nil || owner == nil {
		return nil
	}

	// Find fleets in the same system and at similar orbit
	var nearbyFleets []*entities.Fleet
	for _, fleet := range owner.OwnedFleets {
		if fleet.GetSystemID() == ship.CurrentSystem {
			orbitDiff := ship.GetOrbitDistance() - fleet.LeadShip.GetOrbitDistance()
			if orbitDiff >= -10.0 && orbitDiff <= 10.0 {
				nearbyFleets = append(nearbyFleets, fleet)
			}
		}
	}

	return nearbyFleets
}
