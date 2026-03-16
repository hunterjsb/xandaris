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

type ResourceAccumulationSystem struct {
	*BaseSystem
}

func (ras *ResourceAccumulationSystem) OnTick(tick int64) {
	if tick%10 != 0 {
		return
	}

	context := ras.GetContext()
	if context == nil {
		return
	}

	players := context.GetPlayers()

	for _, player := range players {
		for _, planet := range player.OwnedPlanets {
			if len(planet.Resources) > 0 && tick%1000 == 0 {
				fmt.Printf("[ResAccum] %s owns %s: %d resources, %d buildings\n",
					player.Name, planet.Name, len(planet.Resources), len(planet.Buildings))
			}
			for _, resourceEntity := range planet.Resources {
				if resource, ok := resourceEntity.(*entities.Resource); ok {
					// Auto-fix resource ownership to match planet owner
					if resource.Owner != player.Name {
						fmt.Printf("[ResAccum] Fixed ownership: %s %s -> %s\n",
							resource.ResourceType, resource.Owner, player.Name)
						resource.Owner = player.Name
					}

					extractionAmount := computeResourceExtraction(resource, planet)
					if extractionAmount <= 0 {
						if tick%1000 == 0 {
							fmt.Printf("[ResAccum] %s on %s: extraction=0 (abund=%d)\n",
								resource.ResourceType, planet.Name, resource.Abundance)
						}
						continue
					}

					planet.AddStoredResource(resource.ResourceType, extractionAmount)

					// Depletion: lose 1 abundance per 10,000 ticks (~17 min at 1x).
					// Deposits bottom out at 10 (still produce, just slower).
					// This means a deposit lasts 60×17 = ~1000 minutes from 70→10.
					if resource.Abundance > 10 && tick%10000 == 0 {
						resource.Abundance--
					}
				}
			}
		}
	}
}

func computeResourceExtraction(resource *entities.Resource, planet *entities.Planet) int {
	if resource == nil || planet == nil || resource.Abundance <= 0 {
		return 0
	}

	resourceID := fmt.Sprintf("%d", resource.GetID())
	multiplier := 0.0

	for _, buildingEntity := range planet.Buildings {
		building, ok := buildingEntity.(*entities.Building)
		if !ok || building.BuildingType != entities.BuildingMine || !building.IsOperational {
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

	// Scale by abundance: full rate at 70+, ~14% rate at 10
	abundanceFactor := float64(resource.Abundance) / 70.0
	if abundanceFactor > 1.0 {
		abundanceFactor = 1.0
	}
	if abundanceFactor < 0.1 {
		abundanceFactor = 0.1
	}

	// Tech bonus: +3% extraction per tech level
	techBonus := 1.0 + planet.TechLevel*0.03

	amount := float64(8) * resource.ExtractionRate * multiplier * abundanceFactor * techBonus
	if amount < 1 && multiplier > 0 {
		amount = 1
	}

	return int(amount)
}
