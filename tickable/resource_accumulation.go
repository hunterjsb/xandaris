package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
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

	// Get players from context
	playersInterface := context.GetPlayers()
	if playersInterface == nil {
		return
	}

	players, ok := playersInterface.([]*entities.Player)
	if !ok {
		return
	}

	for _, player := range players {
		for _, planet := range player.OwnedPlanets {
			// Process each resource deposit on the planet
			for _, resourceEntity := range planet.Resources {
				if resource, ok := resourceEntity.(*entities.Resource); ok {
					// Only accumulate from owned resources
					if resource.Owner != player.Name {
						continue
					}

					// Base extraction rate (1 unit per second)
					extractionAmount := int(float64(1) * resource.ExtractionRate)

					// Check for mines on this resource
					mineBonus := 1.0
					for _, buildingEntity := range planet.Buildings {
						if building, ok := buildingEntity.(*entities.Building); ok {
							if building.BuildingType == "Mine" && building.IsOperational && building.AttachedTo == fmt.Sprintf("%d", resource.GetID()) {
								mineBonus += building.ProductionBonus - 1.0 // Add the bonus portion
							}
						}
					}

					// Apply mine bonus
					extractionAmount = int(float64(extractionAmount) * mineBonus)

					// Try to add to planet storage
					planet.AddStoredResource(resource.ResourceType, extractionAmount)
				}
			}
		}
	}
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
