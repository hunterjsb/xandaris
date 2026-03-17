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

	// Use system entity planets (authoritative) instead of player.OwnedPlanets (stale)
	game := context.GetGame()
	if game == nil {
		return
	}
	for _, sys := range game.GetSystems() {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}
			for _, resourceEntity := range planet.Resources {
				if resource, ok := resourceEntity.(*entities.Resource); ok {
					// Auto-fix resource ownership to match planet owner
					if resource.Owner != planet.Owner {
						resource.Owner = planet.Owner
					}

					extractionAmount := computeResourceExtraction(resource, planet)
					if extractionAmount <= 0 {
						continue
					}

					planet.AddStoredResource(resource.ResourceType, extractionAmount)

					// Depletion: lose 1 abundance per 10,000 ticks (~17 min at 1x).
					// Deposits bottom out at 10 (still produce, just slower).
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

	// Power scaling: mines run at 25% without power, scaling up to 100% at full power.
	// This prevents total shutdown while still making generators important.
	powerFactor := 0.25 + 0.75*planet.GetPowerRatio()

	amount := float64(8) * resource.ExtractionRate * multiplier * abundanceFactor * techBonus * powerFactor
	if amount < 1 && multiplier > 0 {
		amount = 1
	}

	return int(amount)
}
