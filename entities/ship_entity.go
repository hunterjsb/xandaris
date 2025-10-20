package entities

import (
	"fmt"
	"image/color"
)

// ShipType represents different types of ships
type ShipType string

const (
	ShipTypeScout     ShipType = "Scout"
	ShipTypeColony    ShipType = "Colony"
	ShipTypeCargo     ShipType = "Cargo"
	ShipTypeFrigate   ShipType = "Frigate"
	ShipTypeDestroyer ShipType = "Destroyer"
	ShipTypeCruiser   ShipType = "Cruiser"
)

// ShipStatus represents the current status of a ship
type ShipStatus string

const (
	ShipStatusDocked   ShipStatus = "Docked"
	ShipStatusOrbiting ShipStatus = "Orbiting"
	ShipStatusMoving   ShipStatus = "Moving"
	ShipStatusIdle     ShipStatus = "Idle"
)

// Ship represents a spacecraft that can move between systems
type Ship struct {
	BaseEntity
	ShipType       ShipType   // Type of ship
	Status         ShipStatus // Current status
	Owner          string     // Player who owns this ship
	CurrentSystem  int        // ID of system the ship is in
	TargetSystem   int        // ID of system moving to (-1 if not moving)
	TravelProgress float64    // 0.0 to 1.0, progress along hyperlane

	// Fuel system
	MaxFuel     int     // Maximum fuel capacity
	CurrentFuel int     // Current fuel amount
	FuelPerJump int     // Fuel consumed per hyperlane jump
	FuelPerTick float64 // Fuel consumed per tick while moving

	// Combat stats
	MaxHealth     int // Maximum hull points
	CurrentHealth int // Current hull points
	AttackPower   int // Damage per attack
	DefenseRating int // Damage reduction

	// Cargo system
	MaxCargo  int            // Maximum cargo capacity
	CargoHold map[string]int // Resources being transported
	Colonists int            // Number of colonists (for colony ships)

	// Movement
	Speed      float64 // Movement speed multiplier (1.0 = normal)
	IsSelected bool    // Whether this ship is selected in UI
}

// NewShip creates a new ship entity
func NewShip(id int, name string, shipType ShipType, systemID int, owner string, c color.RGBA) *Ship {
	ship := &Ship{
		BaseEntity: BaseEntity{
			ID:      id,
			Name:    name,
			Type:    EntityTypeShip,
			SubType: string(shipType),
			Color:   c,
		},
		ShipType:      shipType,
		Status:        ShipStatusDocked,
		Owner:         owner,
		CurrentSystem: systemID,
		TargetSystem:  -1,
		CargoHold:     make(map[string]int),
		Speed:         1.0,
	}

	// Set ship stats based on type
	switch shipType {
	case ShipTypeScout:
		ship.MaxFuel = 200
		ship.FuelPerJump = 20
		ship.FuelPerTick = 0.5
		ship.MaxHealth = 50
		ship.AttackPower = 5
		ship.DefenseRating = 2
		ship.MaxCargo = 50
		ship.Speed = 1.5

	case ShipTypeColony:
		ship.MaxFuel = 300
		ship.FuelPerJump = 40
		ship.FuelPerTick = 0.8
		ship.MaxHealth = 100
		ship.AttackPower = 0
		ship.DefenseRating = 5
		ship.MaxCargo = 100
		ship.Colonists = 1000000 // 1 million colonists
		ship.Speed = 0.8

	case ShipTypeCargo:
		ship.MaxFuel = 250
		ship.FuelPerJump = 30
		ship.FuelPerTick = 0.6
		ship.MaxHealth = 80
		ship.AttackPower = 2
		ship.DefenseRating = 3
		ship.MaxCargo = 500
		ship.Speed = 1.0

	case ShipTypeFrigate:
		ship.MaxFuel = 180
		ship.FuelPerJump = 25
		ship.FuelPerTick = 0.7
		ship.MaxHealth = 120
		ship.AttackPower = 20
		ship.DefenseRating = 8
		ship.MaxCargo = 100
		ship.Speed = 1.2

	case ShipTypeDestroyer:
		ship.MaxFuel = 220
		ship.FuelPerJump = 35
		ship.FuelPerTick = 0.9
		ship.MaxHealth = 200
		ship.AttackPower = 40
		ship.DefenseRating = 12
		ship.MaxCargo = 150
		ship.Speed = 1.0

	case ShipTypeCruiser:
		ship.MaxFuel = 300
		ship.FuelPerJump = 50
		ship.FuelPerTick = 1.2
		ship.MaxHealth = 350
		ship.AttackPower = 60
		ship.DefenseRating = 18
		ship.MaxCargo = 200
		ship.Speed = 0.9
	}

	// Start with full fuel and health
	ship.CurrentFuel = ship.MaxFuel
	ship.CurrentHealth = ship.MaxHealth

	return ship
}

// GetDescription returns a description of the ship
func (s *Ship) GetDescription() string {
	return fmt.Sprintf("%s (%s) - Fuel: %d/%d", s.Name, s.ShipType, s.CurrentFuel, s.MaxFuel)
}

// GetClickRadius returns the click detection radius
func (s *Ship) GetClickRadius(view string) float64 {
	return 8.0
}

// GetOwner returns the owner of this ship
func (s *Ship) GetOwner() string {
	return s.Owner
}

// GetHP returns current and max hull points for this ship
func (s *Ship) GetHP() (int, int) {
	return s.CurrentHealth, s.MaxHealth
}

// CanJump checks if the ship has enough fuel to make a jump
func (s *Ship) CanJump() bool {
	return s.CurrentFuel >= s.FuelPerJump && s.Status != ShipStatusMoving
}

// ConsumeFuel removes fuel from the ship
func (s *Ship) ConsumeFuel(amount int) {
	s.CurrentFuel -= amount
	if s.CurrentFuel < 0 {
		s.CurrentFuel = 0
	}
}

// Refuel adds fuel to the ship
func (s *Ship) Refuel(amount int) int {
	available := s.MaxFuel - s.CurrentFuel
	actualAmount := amount
	if actualAmount > available {
		actualAmount = available
	}
	s.CurrentFuel += actualAmount
	return actualAmount
}

// GetFuelPercentage returns fuel as a percentage
func (s *Ship) GetFuelPercentage() float64 {
	if s.MaxFuel == 0 {
		return 0
	}
	return float64(s.CurrentFuel) / float64(s.MaxFuel) * 100.0
}

// GetHealthPercentage returns health as a percentage
func (s *Ship) GetHealthPercentage() float64 {
	if s.MaxHealth == 0 {
		return 0
	}
	return float64(s.CurrentHealth) / float64(s.MaxHealth) * 100.0
}

// TakeDamage applies damage to the ship
func (s *Ship) TakeDamage(damage int) {
	actualDamage := damage - s.DefenseRating
	if actualDamage < 0 {
		actualDamage = 0
	}
	s.CurrentHealth -= actualDamage
	if s.CurrentHealth < 0 {
		s.CurrentHealth = 0
	}
}

// Repair repairs the ship
func (s *Ship) Repair(amount int) {
	s.CurrentHealth += amount
	if s.CurrentHealth > s.MaxHealth {
		s.CurrentHealth = s.MaxHealth
	}
}

// IsDestroyed returns whether the ship is destroyed
func (s *Ship) IsDestroyed() bool {
	return s.CurrentHealth <= 0
}

// AddCargo adds resources to the ship's cargo hold
func (s *Ship) AddCargo(resourceType string, amount int) int {
	currentLoad := s.GetTotalCargo()
	availableSpace := s.MaxCargo - currentLoad

	actualAmount := amount
	if actualAmount > availableSpace {
		actualAmount = availableSpace
	}

	if s.CargoHold == nil {
		s.CargoHold = make(map[string]int)
	}

	s.CargoHold[resourceType] += actualAmount
	return actualAmount
}

// RemoveCargo removes resources from the ship's cargo hold
func (s *Ship) RemoveCargo(resourceType string, amount int) int {
	if s.CargoHold == nil {
		return 0
	}

	current, exists := s.CargoHold[resourceType]
	if !exists {
		return 0
	}

	actualAmount := amount
	if actualAmount > current {
		actualAmount = current
	}

	s.CargoHold[resourceType] -= actualAmount
	if s.CargoHold[resourceType] <= 0 {
		delete(s.CargoHold, resourceType)
	}

	return actualAmount
}

// GetTotalCargo returns the total cargo currently in the hold
func (s *Ship) GetTotalCargo() int {
	total := 0
	for _, amount := range s.CargoHold {
		total += amount
	}
	return total
}

// GetCargoPercentage returns cargo load as a percentage
func (s *Ship) GetCargoPercentage() float64 {
	if s.MaxCargo == 0 {
		return 0
	}
	return float64(s.GetTotalCargo()) / float64(s.MaxCargo) * 100.0
}

// CanColonize returns whether this ship can colonize a planet
func (s *Ship) CanColonize() bool {
	return s.ShipType == ShipTypeColony && s.Colonists > 0
}

// GetContextMenuTitle implements ContextMenuProvider
func (s *Ship) GetContextMenuTitle() string {
	return s.Name
}

// GetContextMenuItems implements ContextMenuProvider
func (s *Ship) GetContextMenuItems() []string {
	items := []string{
		fmt.Sprintf("Type: %s", s.ShipType),
		fmt.Sprintf("Owner: %s", s.Owner),
		fmt.Sprintf("Status: %s", s.Status),
		"",
		fmt.Sprintf("Fuel: %d/%d (%.0f%%)", s.CurrentFuel, s.MaxFuel, s.GetFuelPercentage()),
		fmt.Sprintf("Health: %d/%d (%.0f%%)", s.CurrentHealth, s.MaxHealth, s.GetHealthPercentage()),
		"",
	}

	if s.ShipType == ShipTypeColony && s.Colonists > 0 {
		items = append(items, fmt.Sprintf("Colonists: %d", s.Colonists))
	}

	if s.MaxCargo > 0 {
		items = append(items, fmt.Sprintf("Cargo: %d/%d", s.GetTotalCargo(), s.MaxCargo))
		if len(s.CargoHold) > 0 {
			for resourceType, amount := range s.CargoHold {
				items = append(items, fmt.Sprintf("  %s: %d", resourceType, amount))
			}
		}
	}

	return items
}

// GetBuildCost returns the cost to build this ship type
func GetShipBuildCost(shipType ShipType) int {
	switch shipType {
	case ShipTypeScout:
		return 500
	case ShipTypeColony:
		return 2000
	case ShipTypeCargo:
		return 1000
	case ShipTypeFrigate:
		return 1500
	case ShipTypeDestroyer:
		return 3000
	case ShipTypeCruiser:
		return 5000
	default:
		return 1000
	}
}

// GetBuildTime returns the ticks required to build this ship type
func GetShipBuildTime(shipType ShipType) int {
	switch shipType {
	case ShipTypeScout:
		return 100 // 10 seconds at 1x speed
	case ShipTypeColony:
		return 300 // 30 seconds
	case ShipTypeCargo:
		return 200 // 20 seconds
	case ShipTypeFrigate:
		return 250 // 25 seconds
	case ShipTypeDestroyer:
		return 400 // 40 seconds
	case ShipTypeCruiser:
		return 600 // 60 seconds
	default:
		return 200
	}
}

// GetShipResourceRequirements returns the resources needed to build a ship
func GetShipResourceRequirements(shipType ShipType) map[string]int {
	requirements := make(map[string]int)

	switch shipType {
	case ShipTypeScout:
		requirements["Iron"] = 50
		requirements["Fuel"] = 20

	case ShipTypeColony:
		requirements["Iron"] = 100
		requirements["Fuel"] = 80
		requirements["Rare Metals"] = 20

	case ShipTypeCargo:
		requirements["Iron"] = 80
		requirements["Fuel"] = 40

	case ShipTypeFrigate:
		requirements["Iron"] = 120
		requirements["Rare Metals"] = 40
		requirements["Fuel"] = 50

	case ShipTypeDestroyer:
		requirements["Iron"] = 200
		requirements["Rare Metals"] = 80
		requirements["Fuel"] = 100

	case ShipTypeCruiser:
		requirements["Iron"] = 300
		requirements["Rare Metals"] = 150
		requirements["Fuel"] = 150
		requirements["Helium-3"] = 50
	}

	return requirements
}
