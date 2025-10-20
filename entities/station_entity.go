package entities

import (
	"fmt"
	"image/color"
)

// Station represents a space station entity in a star system
type Station struct {
	BaseEntity
	StationType  string   // Type like "Trading", "Military", "Research", etc.
	Capacity     int      // Maximum population capacity
	CurrentPop   int      // Current population
	Services     []string // Available services
	Owner        string   // Owner player name or NPC faction/organization
	TradeGoods   []string // Available trade goods
	DefenseLevel int      // Defense rating 0-10
}

// NewStation creates a new station entity
func NewStation(id int, name, stationType string, orbitDistance, orbitAngle float64, c color.RGBA) *Station {
	return &Station{
		BaseEntity: BaseEntity{
			ID:            id,
			Name:          name,
			Type:          EntityTypeStation,
			SubType:       stationType,
			Color:         c,
			OrbitDistance: orbitDistance,
			OrbitAngle:    orbitAngle,
		},
		StationType:  stationType,
		Capacity:     1000,
		CurrentPop:   500,
		Services:     []string{"Docking", "Fuel", "Repairs"},
		Owner:        "Independent",
		TradeGoods:   []string{},
		DefenseLevel: 3,
	}
}

// GetDescription returns a brief description of the station
func (s *Station) GetDescription() string {
	return fmt.Sprintf("%s Station", s.StationType)
}

// GetClickRadius returns the click detection radius
func (s *Station) GetClickRadius(view string) float64 {
	return 6.0 // Fixed radius for station click detection
}

// GetOwner returns the owner of this station
func (s *Station) GetOwner() string {
	return s.Owner
}

// IsPlayerOwned checks if this station is owned by a player (vs NPC faction)
func (s *Station) IsPlayerOwned() bool {
	// Player-owned stations won't have faction names like "Independent", "Trade Union", etc.
	// This is a simple heuristic - can be improved later
	npcFactions := []string{"Independent", "Trade Union", "Commerce Guild", "Merchant Alliance",
		"Military Corp", "Defense Coalition", "Fleet Command", "Sector Defense Force",
		"Research Guild", "Scientific Consortium", "Academy of Sciences", "Tech Institute",
		"Mining Consortium", "Independent Miners", "Resource Corp", "Industrial Alliance"}

	for _, faction := range npcFactions {
		if s.Owner == faction {
			return false
		}
	}
	return s.Owner != ""
}

// GetContextMenuTitle implements ContextMenuProvider
func (s *Station) GetContextMenuTitle() string {
	return s.Name
}

// GetContextMenuItems implements ContextMenuProvider
func (s *Station) GetContextMenuItems() []string {
	items := []string{}

	items = append(items, fmt.Sprintf("Type: %s Station", s.StationType))
	items = append(items, fmt.Sprintf("Owner: %s", s.Owner))
	items = append(items, fmt.Sprintf("Capacity: %d", s.Capacity))
	items = append(items, fmt.Sprintf("Population: %d/%d", s.CurrentPop, s.Capacity))
	items = append(items, fmt.Sprintf("Defense Level: %d", s.DefenseLevel))
	items = append(items, fmt.Sprintf("Docking Fee: %d credits", s.GetDockingFee()))

	if s.CanDock() {
		items = append(items, "Status: Accepting docking")
	} else if s.IsHostile() {
		items = append(items, "Status: Hostile")
	} else {
		items = append(items, "Status: At capacity")
	}

	if len(s.Services) > 0 {
		items = append(items, "") // Empty line
		items = append(items, "Services:")
		for _, service := range s.Services {
			items = append(items, fmt.Sprintf("  - %s", service))
		}
	}

	if len(s.TradeGoods) > 0 {
		items = append(items, "") // Empty line
		items = append(items, "Trade Goods:")
		for _, good := range s.TradeGoods {
			items = append(items, fmt.Sprintf("  - %s", good))
		}
	}

	return items
}

// IsHostile returns whether the station is hostile
func (s *Station) IsHostile() bool {
	return s.StationType == "Military" && s.Owner == "Military Corp"
}

// CanDock returns whether a player can dock at this station
func (s *Station) CanDock() bool {
	return s.CurrentPop < s.Capacity && !s.IsHostile()
}

// GetDockingFee returns the fee for docking at this station
func (s *Station) GetDockingFee() int {
	baseFee := 100

	switch s.StationType {
	case "Trading":
		return baseFee + 25
	case "Military":
		return baseFee * 2 // Military stations charge more
	case "Research":
		return baseFee / 2 // Research stations are cheaper
	case "Shipyard":
		return baseFee + 50
	default:
		return baseFee
	}
}

// GetDetailedInfo returns detailed information about the station
func (s *Station) GetDetailedInfo() map[string]string {
	return map[string]string{
		"Type":       s.StationType,
		"Capacity":   fmt.Sprintf("%d", s.Capacity),
		"Population": fmt.Sprintf("%d/%d", s.CurrentPop, s.Capacity),
		"Owner":      s.Owner,
		"Defense":    fmt.Sprintf("Level %d", s.DefenseLevel),
		"Occupancy":  fmt.Sprintf("%.1f%%", float64(s.CurrentPop)/float64(s.Capacity)*100),
	}
}
