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
}

// OnTick processes resource accumulation each tick
func (ras *ResourceAccumulationSystem) OnTick(tick int64) {
	// Only accumulate resources every 10 ticks (once per second at 1x speed)
	if tick%10 != 0 {
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

					// Base extraction rate (10 units per second, scaled by extraction rate)
					extractionAmount := int(float64(10) * resource.ExtractionRate)

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
					actualAmount := planet.AddStoredResource(resource.ResourceType, extractionAmount)

					// Update abundance based on extraction
					if actualAmount > 0 && resource.Abundance > 0 {
						// Reduce abundance by a small amount (resources deplete slowly)
						depletionRate := 0.001 * float64(actualAmount)
						if depletionRate > 0 {
							resource.Abundance = int(float64(resource.Abundance) - depletionRate)
							if resource.Abundance < 0 {
								resource.Abundance = 0
							}
						}
					}
				}
			}
		}
	}
}

// calculateProduction calculates total resource production value for a planet (in credits/second)
func (ras *ResourceAccumulationSystem) calculateProduction(planetInterface interface{}) int64 {
	planet, ok := planetInterface.(*entities.Planet)
	if !ok {
		return 0
	}

	production := int64(0)

	// Base production: 1 credit per million population per second
	if planet.Population > 0 {
		production += planet.Population / 1000000
	}

	// Habitability bonus (0-100% increases base by up to 50%)
	habitabilityBonus := float64(planet.Habitability) / 200.0
	production = int64(float64(production) * (1.0 + habitabilityBonus))

	// Calculate production from resource extraction
	for _, resourceEntity := range planet.Resources {
		if resource, ok := resourceEntity.(*entities.Resource); ok {
			// Only count owned resources
			if resource.Owner != planet.Owner {
				continue
			}

			// Base extraction rate (10 units per second, scaled by extraction rate)
			extractionAmount := int(float64(10) * resource.ExtractionRate)

			// Apply mine bonuses
			mineBonus := 1.0
			for _, buildingEntity := range planet.Buildings {
				if building, ok := buildingEntity.(*entities.Building); ok {
					if building.BuildingType == "Mine" &&
						building.IsOperational &&
						building.AttachedTo == fmt.Sprintf("%d", resource.GetID()) {
						mineBonus += building.ProductionBonus - 1.0
					}
				}
			}

			// Calculate value of extracted resources
			resourceValue := int64(float64(extractionAmount) * mineBonus * float64(resource.Value))
			production += resourceValue
		}
	}

	// Building production bonuses
	for _, buildingEntity := range planet.Buildings {
		if building, ok := buildingEntity.(*entities.Building); ok {
			// Some buildings might provide direct production bonuses
			if building.BuildingType == "Factory" && building.IsOperational {
				production = int64(float64(production) * building.ProductionBonus)
			}
		}
	}

	return production
}

// GetProductionRate returns the current production rate per second for a player (in credits/second)
func (ras *ResourceAccumulationSystem) GetProductionRate(playerInterface interface{}) int64 {
	player, ok := playerInterface.(*entities.Player)
	if !ok {
		return 0
	}

	totalProduction := int64(0)

	// Sum production from all owned planets
	for _, planet := range player.OwnedPlanets {
		totalProduction += ras.calculateProduction(planet)
	}

	return totalProduction
}

// GetProductionBreakdown returns detailed production info per planet (planet name -> credits/second)
func (ras *ResourceAccumulationSystem) GetProductionBreakdown(playerInterface interface{}) map[string]int64 {
	player, ok := playerInterface.(*entities.Player)
	if !ok {
		return make(map[string]int64)
	}

	breakdown := make(map[string]int64)

	// Calculate production for each owned planet
	for _, planet := range player.OwnedPlanets {
		production := ras.calculateProduction(planet)
		breakdown[planet.Name] = production
	}

	return breakdown
}

// GetResourceBreakdown returns detailed resource extraction info per planet
func (ras *ResourceAccumulationSystem) GetResourceBreakdown(playerInterface interface{}) map[string]map[string]int {
	player, ok := playerInterface.(*entities.Player)
	if !ok {
		return make(map[string]map[string]int)
	}

	breakdown := make(map[string]map[string]int)

	// Calculate resource extraction for each owned planet
	for _, planet := range player.OwnedPlanets {
		planetResources := make(map[string]int)

		for _, resourceEntity := range planet.Resources {
			if resource, ok := resourceEntity.(*entities.Resource); ok {
				// Only count owned resources
				if resource.Owner != planet.Owner {
					continue
				}

				// Base extraction rate (10 units per second, scaled by extraction rate)
				extractionAmount := int(float64(10) * resource.ExtractionRate)

				// Apply mine bonuses
				mineBonus := 1.0
				for _, buildingEntity := range planet.Buildings {
					if building, ok := buildingEntity.(*entities.Building); ok {
						if building.BuildingType == "Mine" &&
							building.IsOperational &&
							building.AttachedTo == fmt.Sprintf("%d", resource.GetID()) {
							mineBonus += building.ProductionBonus - 1.0
						}
					}
				}

				extractionAmount = int(float64(extractionAmount) * mineBonus)
				planetResources[resource.ResourceType] = extractionAmount
			}
		}

		if len(planetResources) > 0 {
			breakdown[planet.Name] = planetResources
		}
	}

	return breakdown
}
