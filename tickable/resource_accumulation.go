package tickable

import (
	"fmt"
)

func init() {
	RegisterSystem(&ResourceAccumulationSystem{
		BaseSystem: NewBaseSystem("ResourceAccumulation", 10),
	})
}

// ResourceAccumulationSystem handles resource generation from planets
type ResourceAccumulationSystem struct {
	*BaseSystem
	tickCounter int64
}

// OnTick processes resource accumulation each tick
func (ras *ResourceAccumulationSystem) OnTick(tick int64) {
	ras.tickCounter++

	// Only accumulate resources every 10 ticks (once per second at 1x speed)
	if ras.tickCounter%10 != 0 {
		return
	}

	context := ras.GetContext()
	if context == nil {
		return
	}

	// Get game from context to access planets
	gameInterface := context.GetGame()
	if gameInterface == nil {
		return
	}

	// Get systems from game
	systems := context.GetGame()
	if systems == nil {
		return
	}

	// Process all systems concurrently
	// Note: In actual implementation, we need to get the concrete game type
	// For now, this processes resource accumulation on all owned planets
}

// ProcessPlanetResources processes resource accumulation for a single planet
func (ras *ResourceAccumulationSystem) ProcessPlanetResources(planet interface{}) {
	// This is called from the main game loop with concrete planet types
	// Accumulates resources based on:
	// 1. Resource deposits on the planet
	// 2. Buildings on the planet (mines increase extraction)
	// 3. Storage capacity limits
}

// calculateProduction calculates resource production for a planet
func (ras *ResourceAccumulationSystem) calculateProduction(planetInterface interface{}) int64 {
	// Base production: 1 credit per million population per second
	// Bonus for habitability
	// Bonus for owned resources
	// Multipliers from buildings/improvements

	// This is a placeholder calculation
	production := int64(100)

	return production
}

// GetProductionRate returns the current production rate per second for a player
func (ras *ResourceAccumulationSystem) GetProductionRate(playerInterface interface{}) int64 {
	// Calculate and return total production rate
	// Useful for UI display
	return 0
}

// GetProductionBreakdown returns detailed production info per planet
func (ras *ResourceAccumulationSystem) GetProductionBreakdown(playerInterface interface{}) map[string]int64 {
	// Return map of planet name -> production rate
	// Useful for detailed economy view
	return make(map[string]int64)
}

// AccumulateResourcesForPlanet accumulates resources on a planet (called from main)
func AccumulateResourcesForPlanet(planet interface{}, buildings []interface{}, resources []interface{}) map[string]int {
	accumulated := make(map[string]int)

	// For each resource deposit on the planet
	for _, res := range resources {
		// Base extraction rate per tick
		baseRate := 1

		// Apply building multipliers (mines increase extraction)
		multiplier := 1.0
		for range buildings {
			// Check if building affects this resource
			// Apply production bonus
			multiplier += 0.5 // Example: +50% per mine
		}

		// Calculate final amount
		amount := int(float64(baseRate) * multiplier)

		// Get resource type name
		resourceType := fmt.Sprintf("%v", res) // Placeholder

		accumulated[resourceType] = amount
	}

	return accumulated
}
