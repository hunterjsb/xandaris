package entities

import (
	"fmt"
	"image/color"
)

// Resource represents a resource node entity on a planet
type Resource struct {
	BaseEntity
	ResourceType   string  // Type like "Iron", "Water", "Helium-3", etc.
	Abundance      int     // Amount of resource available (0-100)
	ExtractionRate float64 // How easily it can be extracted (0.0-1.0)
	Value          int     // Economic value per unit
	Rarity         string  // "Common", "Uncommon", "Rare", "Very Rare"
	Size           int     // Visual size in pixels
	Quality        int     // Quality grade 0-100
	Owner          string  // Name of the player/faction who owns/controls this resource
	NodePosition   float64 // Fixed angle position on planet surface (0 to 2π radians)
}

// NewResource creates a new resource entity
func NewResource(id int, name, resourceType string, orbitDistance, orbitAngle float64, c color.RGBA) *Resource {
	return &Resource{
		BaseEntity: BaseEntity{
			ID:            id,
			Name:          name,
			Type:          EntityTypeResource,
			SubType:       resourceType,
			Color:         c,
			OrbitDistance: orbitDistance,
			OrbitAngle:    orbitAngle,
		},
		ResourceType:   resourceType,
		Abundance:      50,
		ExtractionRate: 0.5,
		Value:          100,
		Rarity:         "Common",
		Size:           3,
		Quality:        50,
		Owner:          "",         // Unowned by default
		NodePosition:   orbitAngle, // Use orbit angle as node position by default
	}
}

// GetDescription returns a brief description of the resource
func (r *Resource) GetDescription() string {
	return fmt.Sprintf("%s (%s)", r.ResourceType, r.Rarity)
}

// GetClickRadius returns the click detection radius
func (r *Resource) GetClickRadius() float64 {
	return float64(r.Size) + 1
}

// GetContextMenuTitle implements ContextMenuProvider
func (r *Resource) GetContextMenuTitle() string {
	return r.Name
}

// GetContextMenuItems implements ContextMenuProvider
func (r *Resource) GetContextMenuItems() []string {
	items := []string{}

	items = append(items, fmt.Sprintf("Type: %s", r.ResourceType))
	items = append(items, fmt.Sprintf("Rarity: %s", r.Rarity))
	items = append(items, fmt.Sprintf("Abundance: %d%%", r.Abundance))
	items = append(items, fmt.Sprintf("Quality: %d%%", r.Quality))
	items = append(items, fmt.Sprintf("Extraction Rate: %.1f%%", r.ExtractionRate*100))
	items = append(items, fmt.Sprintf("Value: %d credits/unit", r.Value))

	if r.Owner != "" {
		items = append(items, "") // Empty line
		items = append(items, fmt.Sprintf("Owner: %s", r.Owner))
	}

	items = append(items, "") // Empty line
	items = append(items, "Status: Ready for extraction")

	return items
}

// GetTotalValue returns the total potential value of this resource node
func (r *Resource) GetTotalValue() int {
	return r.Value * r.Abundance * r.Quality / 100
}

// GetExtractionDifficulty returns a text description of extraction difficulty
func (r *Resource) GetExtractionDifficulty() string {
	if r.ExtractionRate > 0.8 {
		return "Very Easy"
	} else if r.ExtractionRate > 0.6 {
		return "Easy"
	} else if r.ExtractionRate > 0.4 {
		return "Moderate"
	} else if r.ExtractionRate > 0.2 {
		return "Difficult"
	}
	return "Very Difficult"
}

// GetDetailedInfo returns detailed information about the resource
func (r *Resource) GetDetailedInfo() map[string]string {
	return map[string]string{
		"Type":                  r.ResourceType,
		"Rarity":                r.Rarity,
		"Abundance":             fmt.Sprintf("%d%%", r.Abundance),
		"Quality":               fmt.Sprintf("%d%%", r.Quality),
		"Extraction Rate":       fmt.Sprintf("%.1f%%", r.ExtractionRate*100),
		"Value":                 fmt.Sprintf("%d credits/unit", r.Value),
		"Total Value":           fmt.Sprintf("%d credits", r.GetTotalValue()),
		"Extraction Difficulty": r.GetExtractionDifficulty(),
	}
}
