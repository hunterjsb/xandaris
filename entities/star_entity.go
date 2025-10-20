package entities

import (
	"fmt"
	"image/color"
)

// Star represents a star entity in a star system
type Star struct {
	BaseEntity
	StarType    string  // Subtype like "Main Sequence", "Red Giant", etc.
	Temperature int     // Surface temperature in Kelvin
	Mass        float64 // Solar masses (1.0 = Sun's mass)
	Radius      int     // Visual radius in pixels
	Luminosity  float64 // Luminosity relative to Sun (1.0 = Sun's luminosity)
	Age         float64 // Age in billion years
	Metallicity float64 // Heavy element abundance (0.0-2.0, 1.0 = Sun)
	IsBinary    bool    // Whether this is part of a binary system
	Flares      bool    // Whether the star has solar flares
}

// NewStar creates a new star entity
func NewStar(id int, name, starType string, c color.RGBA) *Star {
	return &Star{
		BaseEntity: BaseEntity{
			ID:            id,
			Name:          name,
			Type:          EntityTypeStar,
			SubType:       starType,
			Color:         c,
			OrbitDistance: 0.0, // Stars are always at the center
			OrbitAngle:    0.0,
		},
		StarType:    starType,
		Temperature: 5778, // Default to Sun-like
		Mass:        1.0,
		Radius:      15,
		Luminosity:  1.0,
		Age:         4.6,
		Metallicity: 1.0,
		IsBinary:    false,
		Flares:      false,
	}
}

// GetDescription returns a brief description of the star
func (s *Star) GetDescription() string {
	if s.IsBinary {
		return fmt.Sprintf("%s Binary System", s.StarType)
	}
	return fmt.Sprintf("%s Star", s.StarType)
}

// GetClickRadius returns the click detection radius
func (s *Star) GetClickRadius(view string) float64 {
	return float64(s.Radius) + 5 // Star radius plus margin
}

// GetContextMenuTitle implements ContextMenuProvider
func (s *Star) GetContextMenuTitle() string {
	return s.Name
}

// GetContextMenuItems implements ContextMenuProvider
func (s *Star) GetContextMenuItems() []string {
	items := []string{}

	items = append(items, fmt.Sprintf("Type: %s", s.StarType))
	items = append(items, fmt.Sprintf("Temperature: %d K", s.Temperature))
	items = append(items, fmt.Sprintf("Mass: %.2f solar masses", s.Mass))
	items = append(items, fmt.Sprintf("Luminosity: %.2fx solar", s.Luminosity))
	items = append(items, fmt.Sprintf("Age: %.1f billion years", s.Age))

	if s.IsBinary {
		items = append(items, "Binary star system")
	}

	if s.Flares {
		items = append(items, "Solar flare activity detected")
	}

	items = append(items, "") // Empty line
	items = append(items, "Stellar Classification:")
	items = append(items, fmt.Sprintf("  Spectral class: %s", s.GetSpectralClass()))
	items = append(items, fmt.Sprintf("  Habitability zone: %.1f - %.1f AU", s.GetHabitableZoneInner(), s.GetHabitableZoneOuter()))

	return items
}

// GetSpectralClass returns the spectral classification of the star
func (s *Star) GetSpectralClass() string {
	switch s.StarType {
	case "Blue Giant", "Blue Supergiant":
		return "O-type"
	case "Blue Main Sequence":
		return "B-type"
	case "Hot Main Sequence":
		return "A-type"
	case "Main Sequence":
		if s.Temperature > 6000 {
			return "F-type"
		}
		return "G-type"
	case "Orange Dwarf":
		return "K-type"
	case "Red Dwarf":
		return "M-type"
	case "Red Giant":
		return "K-type giant"
	case "Red Supergiant":
		return "M-type supergiant"
	case "White Dwarf":
		return "White dwarf"
	case "Brown Dwarf":
		return "Brown dwarf"
	default:
		return "Unknown"
	}
}

// GetHabitableZoneInner returns the inner edge of the habitable zone in AU
func (s *Star) GetHabitableZoneInner() float64 {
	// Simplified calculation based on luminosity
	return 0.95 * (s.Luminosity / 1.1)
}

// GetHabitableZoneOuter returns the outer edge of the habitable zone in AU
func (s *Star) GetHabitableZoneOuter() float64 {
	// Simplified calculation based on luminosity
	return 1.37 * (s.Luminosity / 1.1)
}

// GetHabitabilityModifier returns a modifier for planetary habitability based on star type
func (s *Star) GetHabitabilityModifier() float64 {
	switch s.StarType {
	case "Main Sequence":
		return 1.0 // Perfect for life
	case "Orange Dwarf":
		return 0.9 // Very good for life
	case "Red Dwarf":
		return 0.7 // Tidal locking issues, but long-lived
	case "Hot Main Sequence":
		return 0.8 // A bit too energetic
	case "Red Giant":
		return 0.3 // Planets would be too hot or too cold
	case "White Dwarf":
		return 0.1 // Very small habitable zone
	case "Blue Giant", "Blue Supergiant":
		return 0.2 // Too hot and short-lived
	case "Brown Dwarf":
		return 0.1 // Too dim
	default:
		return 0.5
	}
}

// IsStable returns whether the star is stable enough for long-term life
func (s *Star) IsStable() bool {
	return s.StarType == "Main Sequence" ||
		s.StarType == "Orange Dwarf" ||
		s.StarType == "Red Dwarf"
}

// GetLifespan returns the approximate lifespan of the star in billion years
func (s *Star) GetLifespan() float64 {
	switch s.StarType {
	case "Blue Giant", "Blue Supergiant":
		return 0.01 // Very short-lived
	case "Hot Main Sequence":
		return 1.0
	case "Main Sequence":
		return 10.0
	case "Orange Dwarf":
		return 15.0
	case "Red Dwarf":
		return 100.0 // Extremely long-lived
	case "Red Giant":
		return s.Age + 1.0 // Near end of life
	case "White Dwarf":
		return 1000.0 // Very long cooling time
	case "Brown Dwarf":
		return 1000.0 // Never really dies
	default:
		return 5.0
	}
}

// GetDetailedInfo returns detailed information about the star
func (s *Star) GetDetailedInfo() map[string]string {
	return map[string]string{
		"Type":        s.StarType,
		"Temperature": fmt.Sprintf("%d K", s.Temperature),
		"Mass":        fmt.Sprintf("%.2f solar masses", s.Mass),
		"Luminosity":  fmt.Sprintf("%.2fx solar", s.Luminosity),
		"Age":         fmt.Sprintf("%.1f billion years", s.Age),
		"Lifespan":    fmt.Sprintf("%.1f billion years", s.GetLifespan()),
		"Spectral":    s.GetSpectralClass(),
	}
}
