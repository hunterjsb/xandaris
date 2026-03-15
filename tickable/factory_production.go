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
			fps.processFactories(planet)
		}
	}
}

func (fps *FactoryProductionSystem) processFactories(planet *entities.Planet) {
	for _, buildingEntity := range planet.Buildings {
		if building, ok := buildingEntity.(*entities.Building); ok {
			if building.BuildingType == "Factory" && building.IsOperational {
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
	if _, has := planet.StoredResources["Electronics"]; !has {
		planet.AddStoredResource("Electronics", 0)
	}

	// Market-responsive: idle if Electronics storage >80% capacity
	elecStorage := planet.StoredResources["Electronics"]
	if elecStorage != nil && elecStorage.Capacity > 0 {
		ratio := float64(elecStorage.Amount) / float64(elecStorage.Capacity)
		if ratio > 0.8 {
			return
		}
	}

	// Check inputs
	storedRM, hasRM := planet.StoredResources["Rare Metals"]
	storedIron, hasIron := planet.StoredResources["Iron"]
	if !hasRM || storedRM.Amount < rareMetalsNeeded || !hasIron || storedIron.Amount < ironNeeded {
		return
	}

	// Consume inputs
	planet.RemoveStoredResource("Rare Metals", rareMetalsNeeded)
	planet.RemoveStoredResource("Iron", ironNeeded)

	// Produce Electronics
	actual := planet.AddStoredResource("Electronics", electronicsProduced)
	if actual == 0 {
		// Storage full — return inputs
		planet.AddStoredResource("Rare Metals", rareMetalsNeeded)
		planet.AddStoredResource("Iron", ironNeeded)
	}
}

// GetFactoryInfo returns production info for factories on a planet.
func GetFactoryInfo(planet *entities.Planet) (count int, rareMetalsPerInterval int, ironPerInterval int, electronicsPerInterval int) {
	for _, buildingEntity := range planet.Buildings {
		if building, ok := buildingEntity.(*entities.Building); ok {
			if building.BuildingType == "Factory" && building.IsOperational {
				count++
				levelMultiplier := 1.0 + float64(building.Level-1)*0.3
				rareMetalsPerInterval += int(2.0 * levelMultiplier)
				ironPerInterval += int(1.0 * levelMultiplier)
				electronicsPerInterval += int(2.0 * levelMultiplier)
			}
		}
	}
	return
}
