package tickable

import (
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&RefineryProductionSystem{
		BaseSystem: NewBaseSystem("RefineryProduction", 15),
	})
}

// RefineryProductionSystem handles refineries converting oil to fuel
type RefineryProductionSystem struct {
	*BaseSystem
}

// OnTick processes refinery production each tick
func (rps *RefineryProductionSystem) OnTick(tick int64) {
	// Process refineries every 10 ticks (once per second) to match resource accumulation rate
	if tick%10 != 0 {
		return
	}

	context := rps.GetContext()
	if context == nil {
		return
	}

	players := context.GetPlayers()

	for _, player := range players {
		for _, planet := range player.OwnedPlanets {
			rps.processRefineries(planet)
		}
	}
}

// processRefineries processes all refineries on a planet
func (rps *RefineryProductionSystem) processRefineries(planet *entities.Planet) {
	// Find all operational refineries on this planet
	for _, buildingEntity := range planet.Buildings {
		if building, ok := buildingEntity.(*entities.Building); ok {
			if building.BuildingType == "Refinery" && building.IsOperational {
				rps.processRefinery(planet, building)
			}
		}
	}
}

// processRefinery processes a single refinery
func (rps *RefineryProductionSystem) processRefinery(planet *entities.Planet, refinery *entities.Building) {
	// Base: consumes 2 Oil, produces 3 Fuel per interval (10-tick cycle).
	// More efficient conversion that can run with less Oil stockpile.
	baseOilConsumption := 2
	baseFuelProduction := 3

	// Each level adds 30% throughput
	levelMultiplier := 1.0 + float64(refinery.Level-1)*0.3
	oilNeeded := int(float64(baseOilConsumption) * levelMultiplier)
	fuelProduced := int(float64(baseFuelProduction) * levelMultiplier)

	// Ensure Fuel storage exists
	if _, hasFuel := planet.StoredResources["Fuel"]; !hasFuel {
		planet.AddStoredResource("Fuel", 0)
	}

	// Market-responsive: if Fuel storage is over 80% capacity, idle the refinery.
	// This prevents overproduction and conserves Oil for other uses.
	// The refinery restarts when Fuel drops below 60%.
	fuelStorage := planet.StoredResources["Fuel"]
	if fuelStorage != nil && fuelStorage.Capacity > 0 {
		fuelRatio := float64(fuelStorage.Amount) / float64(fuelStorage.Capacity)
		if fuelRatio > 0.8 {
			return // Storage nearly full — idle to conserve Oil
		}
	}

	// Check if planet has enough oil
	storedOil, hasOil := planet.StoredResources["Oil"]
	if !hasOil || storedOil.Amount < oilNeeded {
		return
	}

	// Consume oil
	planet.RemoveStoredResource("Oil", oilNeeded)

	// Produce fuel
	actualFuel := planet.AddStoredResource("Fuel", fuelProduced)

	// If we couldn't add fuel (storage full), put the oil back
	if actualFuel == 0 {
		planet.AddStoredResource("Oil", oilNeeded)
	}
}

// GetRefineryInfo returns information about refinery production per interval.
// Values match processRefinery: base 2 Oil → 3 Fuel, +30% per level.
func (rps *RefineryProductionSystem) GetRefineryInfo(planet *entities.Planet) (count int, oilPerInterval int, fuelPerInterval int) {
	count = 0
	totalOil := 0
	totalFuel := 0

	for _, buildingEntity := range planet.Buildings {
		if building, ok := buildingEntity.(*entities.Building); ok {
			if building.BuildingType == "Refinery" && building.IsOperational {
				count++
				levelMultiplier := 1.0 + float64(building.Level-1)*0.3
				totalOil += int(2.0 * levelMultiplier)
				totalFuel += int(3.0 * levelMultiplier)
			}
		}
	}

	return count, totalOil, totalFuel
}
