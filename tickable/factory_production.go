package tickable

import (
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&FactoryProductionSystem{
		BaseSystem: NewBaseSystem("FactoryProduction", 16),
	})
}

// FactoryProductionSystem handles factories converting Rare Metals + Iron into Electronics.
type FactoryProductionSystem struct {
	*BaseSystem
}

// OnTick processes factory production each tick.
func (fps *FactoryProductionSystem) OnTick(tick int64) {
	if tick%10 != 0 {
		return
	}

	context := fps.GetContext()
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
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				fps.processFactories(planet)
			}
		}
	}
}

func (fps *FactoryProductionSystem) processFactories(planet *entities.Planet) {
	for _, buildingEntity := range planet.Buildings {
		if building, ok := buildingEntity.(*entities.Building); ok {
			if building.BuildingType == entities.BuildingFactory && building.IsOperational && building.GetStaffingRatio() > 0 {
				fps.processFactory(planet, building)
			}
		}
	}
}

// processFactory processes a single factory.
// Base: 2 Rare Metals + 1 Iron → 2 Electronics per interval.
// Each level adds 30% throughput. Tech adds +3% per level.
func (fps *FactoryProductionSystem) processFactory(planet *entities.Planet, factory *entities.Building) {
	baseRareMetals := 2
	baseIron := 1
	baseElectronics := 2

	levelMultiplier := 1.0 + float64(factory.Level-1)*0.3
	staffing := factory.GetStaffingRatio()

	// Power scaling: factories need power. 25% output at 0% power, 100% at full.
	powerFactor := 0.25 + 0.75*planet.GetPowerRatio()

	// Tech bonus: +3% per tech level (same as mining)
	techBonus := 1.0 + planet.TechLevel*0.03

	combined := levelMultiplier * staffing * powerFactor * techBonus
	rareMetalsNeeded := int(float64(baseRareMetals) * combined)
	ironNeeded := int(float64(baseIron) * combined)
	electronicsProduced := int(float64(baseElectronics) * combined)

	if rareMetalsNeeded < 1 || ironNeeded < 1 || electronicsProduced < 1 {
		return
	}

	// Ensure Electronics storage exists
	if _, has := planet.StoredResources[entities.ResElectronics]; !has {
		planet.AddStoredResource(entities.ResElectronics, 0)
	}

	// Market-responsive: idle if Electronics storage >80% capacity
	elecStorage := planet.StoredResources[entities.ResElectronics]
	if elecStorage != nil && elecStorage.Capacity > 0 {
		ratio := float64(elecStorage.Amount) / float64(elecStorage.Capacity)
		if ratio > 0.8 {
			return
		}
	}

	// Check inputs
	storedRM, hasRM := planet.StoredResources[entities.ResRareMetals]
	storedIron, hasIron := planet.StoredResources[entities.ResIron]
	if !hasRM || storedRM.Amount < rareMetalsNeeded || !hasIron || storedIron.Amount < ironNeeded {
		return
	}

	// Consume inputs
	planet.RemoveStoredResource(entities.ResRareMetals, rareMetalsNeeded)
	planet.RemoveStoredResource(entities.ResIron, ironNeeded)

	// Produce Electronics
	actual := planet.AddStoredResource(entities.ResElectronics, electronicsProduced)
	if actual == 0 {
		// Storage full — return inputs
		planet.AddStoredResource(entities.ResRareMetals, rareMetalsNeeded)
		planet.AddStoredResource(entities.ResIron, ironNeeded)
	}
}
