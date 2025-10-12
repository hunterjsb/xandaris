package entities

import (
	"fmt"
	"image/color"
)

// Planet represents a planet entity in a star system
type Planet struct {
	BaseEntity
	Size         int      // Radius in pixels
	PlanetType   string   // Subtype like "Terrestrial", "Gas Giant", etc.
	Population   int64    // Number of inhabitants
	Resources    []Entity // Resource entities on this planet
	Temperature  int      // Temperature in Celsius
	Atmosphere   string   // Type of atmosphere
	HasRings     bool     // Whether the planet has rings
	Habitability int      // Habitability score 0-100
}

// NewPlanet creates a new planet entity
func NewPlanet(id int, name, planetType string, orbitDistance, orbitAngle float64, c color.RGBA) *Planet {
	return &Planet{
		BaseEntity: BaseEntity{
			ID:            id,
			Name:          name,
			Type:          EntityTypePlanet,
			SubType:       planetType,
			Color:         c,
			OrbitDistance: orbitDistance,
			OrbitAngle:    orbitAngle,
		},
		PlanetType:   planetType,
		Size:         5,
		Temperature:  20,
		Atmosphere:   "Thin",
		Population:   0,
		Resources:    []Entity{},
		HasRings:     false,
		Habitability: 50,
	}
}

// GetDescription returns a brief description of the planet
func (p *Planet) GetDescription() string {
	return fmt.Sprintf("%s (%s)", p.Name, p.PlanetType)
}

// GetClickRadius returns the click detection radius
func (p *Planet) GetClickRadius() float64 {
	return float64(p.Size) + 3 // Small margin for accurate clicking
}

// GetContextMenuTitle implements ContextMenuProvider
func (p *Planet) GetContextMenuTitle() string {
	return p.Name
}

// GetContextMenuItems implements ContextMenuProvider
func (p *Planet) GetContextMenuItems() []string {
	items := []string{}

	items = append(items, fmt.Sprintf("Type: %s", p.PlanetType))
	items = append(items, fmt.Sprintf("Temperature: %d°C", p.Temperature))
	items = append(items, fmt.Sprintf("Atmosphere: %s", p.Atmosphere))
	items = append(items, fmt.Sprintf("Population: %d", p.Population))
	items = append(items, fmt.Sprintf("Habitability: %d%%", p.Habitability))

	if p.HasRings {
		items = append(items, "Has planetary rings")
	}

	if len(p.Resources) > 0 {
		items = append(items, "") // Empty line
		items = append(items, fmt.Sprintf("Resources: %d deposits", len(p.Resources)))
		items = append(items, "View planet for details")
	}

	return items
}

// IsHabitable returns whether the planet can support life
func (p *Planet) IsHabitable() bool {
	return p.Temperature > -50 && p.Temperature < 60 &&
		p.Atmosphere != "None" && p.Atmosphere != "Corrosive" &&
		p.PlanetType != "Lava"
}

// GetDetailedInfo returns detailed information about the planet
func (p *Planet) GetDetailedInfo() map[string]string {
	return map[string]string{
		"Type":         p.PlanetType,
		"Population":   fmt.Sprintf("%d", p.Population),
		"Temperature":  fmt.Sprintf("%d°C", p.Temperature),
		"Atmosphere":   p.Atmosphere,
		"Size":         fmt.Sprintf("%d km radius", p.Size*1000),
		"Habitability": fmt.Sprintf("%d%%", p.Habitability),
	}
}
