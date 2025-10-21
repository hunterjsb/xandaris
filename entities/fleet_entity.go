package entities

import (
	"fmt"
	"image/color"
)

// Fleet represents a group of ships at the same location
// Fleet is a proper Entity that can be rendered and interacted with
type Fleet struct {
	BaseEntity
	Ships    []*Ship
	LeadShip *Ship // First ship in fleet, used for positioning
}

// NewFleet creates a new fleet from a list of ships
func NewFleet(id int, ships []*Ship) *Fleet {
	if len(ships) == 0 {
		return nil
	}

	leadShip := ships[0]

	fleet := &Fleet{
		BaseEntity: BaseEntity{
			ID:            id,
			Name:          fmt.Sprintf("Fleet (%d ships)", len(ships)),
			Type:          "Fleet", // Using string for now, can add EntityTypeFleet later
			SubType:       leadShip.GetSubType(),
			Color:         leadShip.Color,
			OrbitDistance: leadShip.GetOrbitDistance(),
			OrbitAngle:    leadShip.GetOrbitAngle(),
		},
		Ships:    ships,
		LeadShip: leadShip,
	}

	return fleet
}

// GetOwner returns the owner of the fleet (from lead ship)
func (f *Fleet) GetOwner() string {
	if f.LeadShip != nil {
		return f.LeadShip.Owner
	}
	return ""
}

// GetHP returns aggregate hull points for the fleet
func (f *Fleet) GetHP() (int, int) {
	currentTotal := 0
	maxTotal := 0
	for _, ship := range f.Ships {
		currentTotal += ship.CurrentHealth
		maxTotal += ship.MaxHealth
	}
	return currentTotal, maxTotal
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

// GetClickRadius returns the click detection radius for the fleet
func (f *Fleet) GetClickRadius(view string) float64 {
	// Fleets are represented by their lead ship
	if f.LeadShip != nil {
		return f.LeadShip.GetClickRadius(view)
	}
	return 10.0 // Default if no ships
}

// Size returns the number of ships in the fleet
func (f *Fleet) Size() int {
	return len(f.Ships)
}

// GetAbsolutePosition returns the fleet's position (from lead ship)
func (f *Fleet) GetAbsolutePosition() (float64, float64) {
	if f.LeadShip != nil {
		return f.LeadShip.GetAbsolutePosition()
	}
	return f.AbsoluteX, f.AbsoluteY
}

// SetAbsolutePosition sets position for all ships in the fleet
func (f *Fleet) SetAbsolutePosition(x, y float64) {
	f.AbsoluteX = x
	f.AbsoluteY = y
	// Update lead ship position
	if f.LeadShip != nil {
		f.LeadShip.SetAbsolutePosition(x, y)
	}
}

// AddShip adds a ship to the fleet
func (f *Fleet) AddShip(ship *Ship) {
	f.Ships = append(f.Ships, ship)
	f.Name = fmt.Sprintf("Fleet (%d ships)", len(f.Ships))
}

// RemoveShip removes a ship from the fleet
func (f *Fleet) RemoveShip(ship *Ship) {
	for i, s := range f.Ships {
		if s == ship {
			f.Ships = append(f.Ships[:i], f.Ships[i+1:]...)
			break
		}
	}

	// Update lead ship if necessary
	if f.LeadShip == ship && len(f.Ships) > 0 {
		f.LeadShip = f.Ships[0]
		f.Color = f.LeadShip.Color
		f.OrbitDistance = f.LeadShip.GetOrbitDistance()
		f.OrbitAngle = f.LeadShip.GetOrbitAngle()
	}

	f.Name = fmt.Sprintf("Fleet (%d ships)", len(f.Ships))
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
func (f *Fleet) GetShipTypeCounts() map[ShipType]int {
	counts := make(map[ShipType]int)
	for _, ship := range f.Ships {
		counts[ship.ShipType]++
	}
	return counts
}

// GetSystemID returns the current system ID for this fleet
func (f *Fleet) GetSystemID() int {
	if f.LeadShip != nil {
		return f.LeadShip.CurrentSystem
	}
	return 0
}

// GetColor returns the fleet color (from lead ship)
func (f *Fleet) GetColor() color.RGBA {
	if f.LeadShip != nil {
		return f.LeadShip.Color
	}
	return f.Color
}

// GetMovementStatus returns a summary of fleet movement capability
// Returns (canMove, lowFuel, noFuel) ship counts
func (f *Fleet) GetMovementStatus() (canMove int, lowFuel int, noFuel int) {
	for _, ship := range f.Ships {
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

// GetContextMenuTitle implements ContextMenuProvider
func (f *Fleet) GetContextMenuTitle() string {
	if f.Size() == 1 {
		return f.Ships[0].Name
	}
	return fmt.Sprintf("Fleet (%d ships)", f.Size())
}

// GetContextMenuItems implements ContextMenuProvider
func (f *Fleet) GetContextMenuItems() []string {
	items := []string{
		fmt.Sprintf("Ships: %d", f.Size()),
		fmt.Sprintf("Owner: %s", f.GetOwner()),
		"",
	}

	// Show ship type breakdown
	typeCounts := f.GetShipTypeCounts()
	for shipType, count := range typeCounts {
		items = append(items, fmt.Sprintf("  %dx %s", count, shipType))
	}

	items = append(items, "")

	// Show aggregate fuel
	avgFuel := f.GetAverageFuelPercent()
	items = append(items, fmt.Sprintf("Avg Fuel: %.0f%%", avgFuel))

	// Show aggregate health
	currentHP, maxHP := f.GetHP()
	if maxHP > 0 {
		hpPercent := float64(currentHP) / float64(maxHP) * 100
		items = append(items, fmt.Sprintf("Fleet HP: %d/%d (%.0f%%)", currentHP, maxHP, hpPercent))
	}

	return items
}
