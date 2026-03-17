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

	// Use system entity planets (authoritative) instead of player.OwnedPlanets (stale)
	game := context.GetGame()
	if game == nil {
		return
	}
	for _, sys := range game.GetSystems() {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				rps.processRefineries(planet)
			}
		}
	}
}

// processRefineries processes all refineries on a planet
func (rps *RefineryProductionSystem) processRefineries(planet *entities.Planet) {
	for _, buildingEntity := range planet.Buildings {
		if building, ok := buildingEntity.(*entities.Building); ok {
			if building.BuildingType == entities.BuildingRefinery && building.IsOperational && building.GetStaffingRatio() > 0 {
				rps.processRefinery(planet, building)
			}
		}
	}
}

// processRefinery processes a single refinery.
// Base: 2 Oil → 3 Fuel per interval. +30% per level. +3% per tech level.
func (rps *RefineryProductionSystem) processRefinery(planet *entities.Planet, refinery *entities.Building) {
	baseOilConsumption := 2
	baseFuelProduction := 3

	// Each level adds 30% throughput
	levelMultiplier := 1.0 + float64(refinery.Level-1)*0.3

	// Power scaling: refineries need power to run efficiently.
	// At 0% power: 25% throughput. At 100% power: full throughput.
	powerFactor := 0.25 + 0.75*planet.GetPowerRatio()

	// Tech bonus: +3% per tech level (consistent with mines and factories)
	techBonus := 1.0 + planet.TechLevel*0.03

	combined := levelMultiplier * powerFactor * techBonus
	oilNeeded := int(float64(baseOilConsumption) * combined)
	fuelProduced := int(float64(baseFuelProduction) * combined)
	if oilNeeded < 1 {
		oilNeeded = 1
	}
	if fuelProduced < 1 {
		fuelProduced = 1
	}

	// Ensure Fuel storage exists
	if _, hasFuel := planet.StoredResources[entities.ResFuel]; !hasFuel {
		planet.AddStoredResource(entities.ResFuel, 0)
	}

	// Market-responsive: if Fuel storage is over 80% capacity, idle the refinery.
	// This prevents overproduction and conserves Oil for other uses.
	// The refinery restarts when Fuel drops below 60%.
	fuelStorage := planet.StoredResources[entities.ResFuel]
	if fuelStorage != nil && fuelStorage.Capacity > 0 {
		fuelRatio := float64(fuelStorage.Amount) / float64(fuelStorage.Capacity)
		if fuelRatio > 0.8 {
			return // Storage nearly full — idle to conserve Oil
		}
	}

	// Check if planet has enough oil
	storedOil, hasOil := planet.StoredResources[entities.ResOil]
	if !hasOil || storedOil.Amount < oilNeeded {
		return
	}

	// Consume oil
	planet.RemoveStoredResource(entities.ResOil, oilNeeded)

	// Produce fuel
	actualFuel := planet.AddStoredResource(entities.ResFuel, fuelProduced)

	// If we couldn't add fuel (storage full), put the oil back
	if actualFuel == 0 {
		planet.AddStoredResource(entities.ResOil, oilNeeded)
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
			if building.BuildingType == entities.BuildingRefinery && building.IsOperational {
				count++
				levelMultiplier := 1.0 + float64(building.Level-1)*0.3
				totalOil += int(2.0 * levelMultiplier)
				totalFuel += int(3.0 * levelMultiplier)
			}
		}
	}

	return count, totalOil, totalFuel
}
