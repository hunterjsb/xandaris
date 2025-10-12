package main

import (
	"math"

	"github.com/hunterjsb/xandaris/entities"
)

// ViewScale represents different scaling contexts
type ViewScale struct {
	Name            string
	BasePixelSize   float64 // Base size for a "unit" in this scale
	OrbitMultiplier float64 // Multiplier for orbital distances
}

var (
	// GalaxyScale - for rendering systems in galaxy view
	GalaxyScale = ViewScale{
		Name:            "Galaxy",
		BasePixelSize:   8.0, // Systems are small dots
		OrbitMultiplier: 0.0, // No orbits in galaxy view
	}

	// SystemScale - for rendering entities within a system
	SystemScale = ViewScale{
		Name:            "System",
		BasePixelSize:   1.0, // 1:1 pixel ratio
		OrbitMultiplier: 1.0, // Direct orbital distances
	}

	// DetailScale - for close-up views (future use)
	DetailScale = ViewScale{
		Name:            "Detail",
		BasePixelSize:   3.0, // Enlarged for detail
		OrbitMultiplier: 2.0, // More spread out
	}
)

// ScaleSize scales a size value according to the view scale
func (vs *ViewScale) ScaleSize(baseSize float64) int {
	return int(baseSize * vs.BasePixelSize)
}

// ScaleOrbitDistance scales an orbital distance according to the view scale
func (vs *ViewScale) ScaleOrbitDistance(distance float64) float64 {
	return distance * vs.OrbitMultiplier
}

// ScaleClickRadius scales a click radius according to the view scale
func (vs *ViewScale) ScaleClickRadius(radius float64) float64 {
	return radius * vs.BasePixelSize
}

// FitToScreen adjusts orbital distances to fit within screen bounds
func (vs *ViewScale) FitToScreen(maxDistance float64, screenWidth, screenHeight int) float64 {
	// Calculate available screen space (leave margins)
	availableWidth := float64(screenWidth) * 0.8   // 80% of width
	availableHeight := float64(screenHeight) * 0.8 // 80% of height

	// Use the smaller dimension to ensure everything fits
	maxScreenDistance := math.Min(availableWidth, availableHeight) / 2.0

	// Calculate scale factor needed
	scaleFactor := maxScreenDistance / maxDistance

	// Don't scale up beyond reasonable limits
	if scaleFactor > 2.0 {
		scaleFactor = 2.0
	}

	// Don't scale down too much (minimum 0.3x)
	if scaleFactor < 0.3 {
		scaleFactor = 0.3
	}

	return scaleFactor
}

// AutoScale automatically determines the best scale for a system
func AutoScale(maxOrbitDistance float64, screenWidth, screenHeight int) *ViewScale {
	// Create a dynamic scale based on the maximum orbital distance
	scale := SystemScale // Start with system scale as base

	// Calculate fit factor
	fitFactor := scale.FitToScreen(maxOrbitDistance, screenWidth, screenHeight)

	// Create new scale with adjusted multipliers
	return &ViewScale{
		Name:            "Auto",
		BasePixelSize:   scale.BasePixelSize,
		OrbitMultiplier: fitFactor,
	}
}

// GetSystemMaxOrbitDistance returns the maximum orbital distance in a system
func GetSystemMaxOrbitDistance(system *entities.System) float64 {
	maxDistance := 0.0

	for _, entity := range system.Entities {
		distance := entity.GetOrbitDistance()
		if distance > maxDistance {
			maxDistance = distance
		}
	}

	// Add some padding
	return maxDistance + 50.0
}

// ScalePosition scales a position from world coordinates to screen coordinates
func (vs *ViewScale) ScalePosition(worldX, worldY, centerX, centerY float64) (float64, float64) {
	scaledX := centerX + (worldX-centerX)*vs.OrbitMultiplier
	scaledY := centerY + (worldY-centerY)*vs.OrbitMultiplier
	return scaledX, scaledY
}

// ScaleEntityForRendering returns the appropriate size for rendering an entity
func (vs *ViewScale) ScaleEntityForRendering(entity interface{}) int {
	switch entity.(type) {
	case *entities.System:
		return vs.ScaleSize(float64(circleRadius))
	default:
		// For other entities, use a base size
		return vs.ScaleSize(8.0)
	}
}
