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
	// Process refineries every tick (10 ticks per second)
	context := rps.GetContext()
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
	// Base conversion rate: 10 Oil per second → 5 Fuel per second
	// At 10 ticks per second: 1 Oil per tick → 0.5 Fuel per tick
	baseOilConsumption := 1
	baseFuelProduction := 0.5

	// Apply building level bonus (each level adds 20% efficiency)
	levelMultiplier := 1.0 + float64(refinery.Level-1)*0.2
	oilNeeded := int(float64(baseOilConsumption) * levelMultiplier)
	fuelProduced := int(baseFuelProduction * levelMultiplier)
	if fuelProduced < 1 {
		fuelProduced = 1 // Always produce at least 1 fuel per tick
	}

	// Ensure Fuel storage exists
	if _, hasFuel := planet.StoredResources["Fuel"]; !hasFuel {
		planet.AddStoredResource("Fuel", 0)
	}

	// Check if planet has enough oil
	storedOil, hasOil := planet.StoredResources["Oil"]
	if !hasOil || storedOil.Amount < oilNeeded {
		// Not enough oil - refinery is idle
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

// GetRefineryInfo returns information about refinery production
func (rps *RefineryProductionSystem) GetRefineryInfo(planet *entities.Planet) (count int, oilPerSec int, fuelPerSec int) {
	count = 0
	totalOil := 0
	totalFuel := 0

	for _, buildingEntity := range planet.Buildings {
		if building, ok := buildingEntity.(*entities.Building); ok {
			if building.BuildingType == "Refinery" && building.IsOperational {
				count++
				// Calculate rate per second (10 ticks per second)
				levelMultiplier := 1.0 + float64(building.Level-1)*0.2
				oilRate := int(10.0 * levelMultiplier) // 10 oil/sec base
				fuelRate := int(5.0 * levelMultiplier) // 5 fuel/sec base
				totalOil += oilRate
				totalFuel += fuelRate
			}
		}
	}

	return count, totalOil, totalFuel
}
