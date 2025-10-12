package entities

import (
	"fmt"
	"image/color"
)

// Building represents a building entity on a planet or resource
type Building struct {
	BaseEntity
	BuildingType       string  // "Mine", "Extractor", "Shipyard", "Barracks", "Habitat", etc.
	AttachedTo         string  // ID of planet or resource this is built on
	AttachmentType     string  // "Planet" or "Resource"
	ProductionBonus    float64 // Multiplier for resource production (e.g., 1.5 = +50%)
	PopulationCapacity int64   // Additional population capacity (for habitats)
	BuildCost          int     // Cost in credits to build
	UpkeepCost         int     // Cost per tick to maintain
	Level              int     // Building level (for upgrades)
	MaxLevel           int     // Maximum upgrade level
	IsOperational      bool    // Whether the building is functioning
	Size               int     // Visual size in pixels
	Owner              string  // Name of the player who owns this building
	Description        string  // Detailed description
}

// NewBuilding creates a new building entity
func NewBuilding(id int, name, buildingType string, orbitDistance, orbitAngle float64, c color.RGBA) *Building {
	return &Building{
		BaseEntity: BaseEntity{
			ID:            id,
			Name:          name,
			Type:          EntityTypeBuilding,
			SubType:       buildingType,
			Color:         c,
			OrbitDistance: orbitDistance,
			OrbitAngle:    orbitAngle,
		},
		BuildingType:       buildingType,
		AttachedTo:         "",
		AttachmentType:     "Planet",
		ProductionBonus:    1.0,
		PopulationCapacity: 0,
		BuildCost:          1000,
		UpkeepCost:         10,
		Level:              1,
		MaxLevel:           5,
		IsOperational:      true,
		Size:               4,
		Owner:              "",
		Description:        "",
	}
}

// GetDescription returns a brief description of the building
func (b *Building) GetDescription() string {
	if b.Level > 1 {
		return fmt.Sprintf("%s (Level %d)", b.BuildingType, b.Level)
	}
	return b.BuildingType
}

// GetClickRadius returns the click detection radius
func (b *Building) GetClickRadius() float64 {
	return float64(b.Size) + 1
}

// GetContextMenuTitle implements ContextMenuProvider
func (b *Building) GetContextMenuTitle() string {
	return b.Name
}

// GetContextMenuItems implements ContextMenuProvider
func (b *Building) GetContextMenuItems() []string {
	items := []string{}

	items = append(items, fmt.Sprintf("Type: %s", b.BuildingType))
	items = append(items, fmt.Sprintf("Level: %d/%d", b.Level, b.MaxLevel))

	if b.IsOperational {
		items = append(items, "Status: Operational")
	} else {
		items = append(items, "Status: Offline")
	}

	items = append(items, fmt.Sprintf("Upkeep: %d credits/sec", b.UpkeepCost))

	if b.ProductionBonus > 1.0 {
		items = append(items, fmt.Sprintf("Production: +%.0f%%", (b.ProductionBonus-1.0)*100))
	}

	if b.PopulationCapacity > 0 {
		items = append(items, fmt.Sprintf("Housing: %d capacity", b.PopulationCapacity))
	}

	if b.Owner != "" {
		items = append(items, "") // Empty line
		items = append(items, fmt.Sprintf("Owner: %s", b.Owner))
	}

	if b.Description != "" {
		items = append(items, "") // Empty line
		items = append(items, b.Description)
	}

	return items
}

// CanUpgrade returns whether the building can be upgraded
func (b *Building) CanUpgrade() bool {
	return b.Level < b.MaxLevel && b.IsOperational
}

// GetUpgradeCost returns the cost to upgrade to the next level
func (b *Building) GetUpgradeCost() int {
	if !b.CanUpgrade() {
		return 0
	}
	// Cost increases by 50% per level
	return int(float64(b.BuildCost) * float64(b.Level) * 1.5)
}

// Upgrade increases the building level and improves stats
func (b *Building) Upgrade() bool {
	if !b.CanUpgrade() {
		return false
	}

	b.Level++

	// Increase production bonus by 20% per level
	b.ProductionBonus += 0.2

	// Increase population capacity by 50% per level
	if b.PopulationCapacity > 0 {
		b.PopulationCapacity = int64(float64(b.PopulationCapacity) * 1.5)
	}

	// Increase upkeep by 30% per level
	b.UpkeepCost = int(float64(b.UpkeepCost) * 1.3)

	return true
}

// SetOperational sets the operational status
func (b *Building) SetOperational(operational bool) {
	b.IsOperational = operational
}

// GetEffectiveProductionBonus returns the production bonus (0 if not operational)
func (b *Building) GetEffectiveProductionBonus() float64 {
	if !b.IsOperational {
		return 0.0
	}
	return b.ProductionBonus
}

// GetEffectivePopulationCapacity returns the population capacity (0 if not operational)
func (b *Building) GetEffectivePopulationCapacity() int64 {
	if !b.IsOperational {
		return 0
	}
	return b.PopulationCapacity
}

// GetDetailedInfo returns detailed information about the building
func (b *Building) GetDetailedInfo() map[string]string {
	status := "Operational"
	if !b.IsOperational {
		status = "Offline"
	}

	info := map[string]string{
		"Type":       b.BuildingType,
		"Level":      fmt.Sprintf("%d/%d", b.Level, b.MaxLevel),
		"Status":     status,
		"Build Cost": fmt.Sprintf("%d credits", b.BuildCost),
		"Upkeep":     fmt.Sprintf("%d credits/sec", b.UpkeepCost),
		"Attachment": b.AttachmentType,
	}

	if b.ProductionBonus > 1.0 {
		info["Production Bonus"] = fmt.Sprintf("+%.0f%%", (b.ProductionBonus-1.0)*100)
	}

	if b.PopulationCapacity > 0 {
		info["Population Capacity"] = fmt.Sprintf("%d", b.PopulationCapacity)
	}

	if b.CanUpgrade() {
		info["Upgrade Cost"] = fmt.Sprintf("%d credits", b.GetUpgradeCost())
	}

	return info
}

// IsResourceBuilding returns whether this building is attached to a resource
func (b *Building) IsResourceBuilding() bool {
	return b.AttachmentType == "Resource"
}

// IsPlanetBuilding returns whether this building is attached to a planet
func (b *Building) IsPlanetBuilding() bool {
	return b.AttachmentType == "Planet"
}
