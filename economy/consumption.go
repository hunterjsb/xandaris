package economy

import (
	"github.com/hunterjsb/xandaris/entities"
)

// ConsumptionRate defines per-resource consumption per tick-interval.
type ConsumptionRate struct {
	ResourceType  string
	PerPopulation float64 // Units consumed per PopDivisor population
	PopDivisor    float64
}

// PopulationConsumption defines resource drain from population.
// These rates apply per 10-tick interval (~1 second at 1x speed).
var PopulationConsumption = []ConsumptionRate{
	{entities.ResWater, 1, 250},        // 1 per 250 pop — life support (heaviest)
	{entities.ResIron, 1, 500},         // 1 per 500 pop — infrastructure
	{entities.ResOil, 1, 800},          // 1 per 800 pop — industry
	{entities.ResRareMetals, 1, 5000},  // 1 per 5000 pop — raw materials (luxury)
	{entities.ResHelium3, 1, 10000},    // 1 per 10000 pop — fusion (luxury)
	{entities.ResElectronics, 1, 3000}, // 1 per 3000 pop — technology goods
	// Fuel consumed by buildings only (Shipyard: 2/interval, Refinery upkeep)
}

// BuildingResourceUpkeep maps building type -> resources consumed per interval.
var BuildingResourceUpkeep = map[string][]struct {
	ResourceType string
	Amount       int
}{
	entities.BuildingMine:        {{entities.ResIron, 1}},
	entities.BuildingTradingPost: {{entities.ResOil, 1}},
	entities.BuildingRefinery:    {{entities.ResOil, 2}, {entities.ResIron, 1}},
	entities.BuildingFactory:     {{entities.ResIron, 1}, {entities.ResOil, 1}},
	entities.BuildingShipyard:    {{entities.ResFuel, 2}, {entities.ResIron, 1}, {entities.ResElectronics, 1}},
	entities.BuildingHabitat:     {{entities.ResWater, 1}},
	entities.BuildingBase:        {},
}

// BuildingCreditUpkeep defines the credit cost per building per interval (+ level - 1).
var BuildingCreditUpkeep = map[string]int{
	entities.BuildingMine:          2,
	entities.BuildingTradingPost:   3,
	entities.BuildingHabitat:       1,
	entities.BuildingRefinery:      4,
	entities.BuildingFactory:       5,
	entities.BuildingShipyard:      6,
	entities.BuildingGenerator:     3,
	entities.BuildingFusionReactor: 8,
}

// ConsumptionResult contains both demand signals and credit drain info.
type ConsumptionResult struct {
	Demand      map[string]float64 // resource type -> consumed amount (demand signal)
	CreditDrain int                // total credits drained from building upkeep
}

// ProcessConsumption drains resources from all planets and returns demand + credit drain.
func ProcessConsumption(players []*entities.Player) ConsumptionResult {
	result := ConsumptionResult{
		Demand: make(map[string]float64),
	}

	for _, player := range players {
		if player == nil {
			continue
		}
		playerCreditDrain := 0

		for _, planet := range player.OwnedPlanets {
			if planet == nil {
				continue
			}

			// Population consumption
			for _, rate := range PopulationConsumption {
				needed := float64(planet.Population) / rate.PopDivisor * rate.PerPopulation
				if needed < 0.5 {
					continue
				}
				result.Demand[rate.ResourceType] += needed
				planet.RemoveStoredResource(rate.ResourceType, int(needed))
			}

			// Building upkeep (resources)
			for _, buildingEntity := range planet.Buildings {
				building, ok := buildingEntity.(*entities.Building)
				if !ok || !building.IsOperational {
					continue
				}
				if upkeeps, found := BuildingResourceUpkeep[building.BuildingType]; found {
					for _, upkeep := range upkeeps {
						result.Demand[upkeep.ResourceType] += float64(upkeep.Amount)
						planet.RemoveStoredResource(upkeep.ResourceType, upkeep.Amount)
					}
				}

				if cost, found := BuildingCreditUpkeep[building.BuildingType]; found {
					// Upkeep scales gradually with level
					playerCreditDrain += cost + (building.Level - 1)
				}
			}
		}

		// Population administration costs (1 credit per 1000 population)
		for _, planet := range player.OwnedPlanets {
			if planet != nil {
				playerCreditDrain += int(planet.Population / 1000)
			}
		}

		// Wealth tax: 0.1% per interval on excess credits.
		// Threshold scales with total population (10cr per citizen).
		// A 5000-pop player can hold 50,000cr tax-free; 20,000-pop = 200,000cr.
		totalPop := int64(0)
		for _, planet := range player.OwnedPlanets {
			if planet != nil {
				totalPop += planet.Population
			}
		}
		taxThreshold := int(totalPop) * 10
		if taxThreshold < 10000 {
			taxThreshold = 10000 // minimum threshold
		}
		if player.Credits > taxThreshold {
			tax := (player.Credits - taxThreshold) / 1000
			if tax > 0 {
				playerCreditDrain += tax
			}
		}

		// Deduct credit upkeep - if can't afford, shut down non-essential buildings
		if playerCreditDrain > 0 {
			if player.Credits >= playerCreditDrain {
				player.Credits -= playerCreditDrain
			} else {
				player.Credits = 0
				for _, planet := range player.OwnedPlanets {
					if planet == nil {
						continue
					}
					for _, be := range planet.Buildings {
						if b, ok := be.(*entities.Building); ok {
							if b.BuildingType != entities.BuildingBase && b.IsOperational {
								b.IsOperational = false
							}
						}
					}
				}
			}
			result.CreditDrain += playerCreditDrain
		}

		// Re-enable buildings once credits recover
		if player.Credits > 500 {
			for _, planet := range player.OwnedPlanets {
				if planet == nil {
					continue
				}
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok {
						if !b.IsOperational && b.BuildingType != entities.BuildingBase {
							b.IsOperational = true
						}
					}
				}
			}
		}
	}

	return result
}
