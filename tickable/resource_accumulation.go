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

					extractionAmount := computeResourceExtraction(resource, planet)
					if extractionAmount <= 0 {
						continue
					}

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

func computeResourceExtraction(resource *entities.Resource, planet *entities.Planet) int {
	if resource == nil || planet == nil {
		return 0
	}

	resourceID := fmt.Sprintf("%d", resource.GetID())
	multiplier := 0.0

	for _, buildingEntity := range planet.Buildings {
		building, ok := buildingEntity.(*entities.Building)
		if !ok {
			continue
		}
		if building.BuildingType != "Mine" {
			continue
		}
		if !building.IsOperational {
			continue
		}
		if building.AttachmentType != "Resource" || building.AttachedTo != resourceID {
			continue
		}

		ratio := building.GetStaffingRatio()
		if ratio <= 0 {
			continue
		}

		multiplier += ratio * building.ProductionBonus
	}

	if multiplier <= 0 {
		return 0
	}

	amount := float64(10) * resource.ExtractionRate * multiplier
	if amount < 0 {
		return 0
	}

	return int(amount)
}

// calculateProduction calculates total resource production value for a planet (in credits/second)
func (ras *ResourceAccumulationSystem) calculateProduction(planetInterface interface{}) int64 {
	planet, ok := planetInterface.(*entities.Planet)
	if !ok {
		return 0
	}

	production := int64(0)

	// Base production: 1 credit per 100 population per interval
	if planet.Population > 0 {
		production += planet.Population / 100
	}

	// Calculate production from resource extraction
	for _, resourceEntity := range planet.Resources {
		if resource, ok := resourceEntity.(*entities.Resource); ok {
			if resource.Owner != planet.Owner {
				continue
			}

			extractionAmount := computeResourceExtraction(resource, planet)
			if extractionAmount <= 0 {
				continue
			}

			resourceValue := int64(float64(extractionAmount) * float64(resource.Value))
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
				if resource.Owner != planet.Owner {
					continue
				}

				extractionAmount := computeResourceExtraction(resource, planet)
				if extractionAmount <= 0 {
					continue
				}

				planetResources[resource.ResourceType] = extractionAmount
			}
		}

		if len(planetResources) > 0 {
			breakdown[planet.Name] = planetResources
		}
	}

	return breakdown
}
