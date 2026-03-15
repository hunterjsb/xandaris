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

	players := context.GetPlayers()

	for _, player := range players {
		for _, planet := range player.OwnedPlanets {
			fps.processFactories(planet)
		}
	}
}

func (fps *FactoryProductionSystem) processFactories(planet *entities.Planet) {
	for _, buildingEntity := range planet.Buildings {
		if building, ok := buildingEntity.(*entities.Building); ok {
			if building.BuildingType == entities.BuildingFactory && building.IsOperational {
				fps.processFactory(planet, building)
			}
		}
	}
}

// processFactory processes a single factory.
// Base: 2 Rare Metals + 1 Iron → 2 Electronics per interval.
// Each level adds 30% throughput.
func (fps *FactoryProductionSystem) processFactory(planet *entities.Planet, factory *entities.Building) {
	baseRareMetals := 2
	baseIron := 1
	baseElectronics := 2

	levelMultiplier := 1.0 + float64(factory.Level-1)*0.3
	staffing := factory.GetStaffingRatio()
	rareMetalsNeeded := int(float64(baseRareMetals) * levelMultiplier * staffing)
	ironNeeded := int(float64(baseIron) * levelMultiplier * staffing)
	electronicsProduced := int(float64(baseElectronics) * levelMultiplier * staffing)

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

