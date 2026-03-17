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
	{entities.ResWater, 1, 1000},        // 1 per 1000 pop — life support
	{entities.ResIron, 1, 2000},        // 1 per 2000 pop — infrastructure
	{entities.ResOil, 1, 3000},         // 1 per 3000 pop — industry
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
	entities.BuildingMine:        {},
	entities.BuildingTradingPost: {{entities.ResOil, 1}},
	entities.BuildingRefinery:    {{entities.ResOil, 2}},
	entities.BuildingFactory:     {{entities.ResOil, 1}},
	entities.BuildingShipyard:    {{entities.ResFuel, 2}, {entities.ResElectronics, 1}},
	entities.BuildingHabitat:     {{entities.ResWater, 1}},
	entities.BuildingBase:        {},
	entities.BuildingResearchLab: {{entities.ResWater, 1}}, // researchers need water
}

// BuildingCreditUpkeep defines the credit cost per building per interval (+ level - 1).
// Core production buildings (Mine, Generator, Habitat, Base, TP) have zero upkeep
// so colonies can always sustain basic resource extraction. Only advanced buildings
// that transform resources (Refinery, Factory) or enable expansion (Shipyard) cost credits.
var BuildingCreditUpkeep = map[string]int{
	entities.BuildingMine:          0, // free — core production must always work
	entities.BuildingTradingPost:   0, // free — trade infrastructure is essential
	entities.BuildingHabitat:       0, // free — population growth is essential
	entities.BuildingGenerator:     0, // free — power is essential
	entities.BuildingFusionReactor: 2, // low — advanced power
	entities.BuildingRefinery:      3, // moderate — resource processing
	entities.BuildingFactory:       5, // higher — manufacturing
	entities.BuildingShipyard:      6, // highest — ship construction
	entities.BuildingResearchLab:   4, // moderate — research operations
}

// ConsumptionResult contains both demand signals and credit drain info.
type ConsumptionResult struct {
	Demand      map[string]float64 // resource type -> consumed amount (demand signal)
	CreditDrain int                // total credits drained from building upkeep
}

// ProcessConsumption drains resources from all planets and returns demand + credit drain.
// Uses system entity planets (authoritative) to avoid stale pointer issues after save/load.
func ProcessConsumption(players []*entities.Player, systems []*entities.System) ConsumptionResult {
	result := ConsumptionResult{
		Demand: make(map[string]float64),
	}

	// Build player lookup by name
	playerMap := make(map[string]*entities.Player)
	for _, p := range players {
		if p != nil {
			playerMap[p.Name] = p
		}
	}

	// Track credit drain per player
	playerCreditDrain := make(map[string]int)

	// Process all owned planets from system entities (authoritative)
	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
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

			// Building upkeep (resources + credits) — only for staffed buildings
			for _, buildingEntity := range planet.Buildings {
				building, ok := buildingEntity.(*entities.Building)
				if !ok || !building.IsOperational || building.GetStaffingRatio() <= 0 {
					continue
				}
				if upkeeps, found := BuildingResourceUpkeep[building.BuildingType]; found {
					for _, upkeep := range upkeeps {
						result.Demand[upkeep.ResourceType] += float64(upkeep.Amount)
						planet.RemoveStoredResource(upkeep.ResourceType, upkeep.Amount)
					}
				}

				if cost, found := BuildingCreditUpkeep[building.BuildingType]; found {
					playerCreditDrain[planet.Owner] += cost + (building.Level - 1)
				}
			}

			// Population administration costs (1 credit per 1000 population)
			playerCreditDrain[planet.Owner] += int(planet.Population / 1000)
		}
	}

	// Apply credit drain and wealth tax per player
	for _, player := range players {
		if player == nil {
			continue
		}
		drain := playerCreditDrain[player.Name]

		// Wealth tax: 0.05% per interval on excess credits.
		totalPop := int64(0)
		totalBuildings := 0
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if p, ok := e.(*entities.Planet); ok && p.Owner == player.Name {
					totalPop += p.Population
					totalBuildings += len(p.Buildings)
				}
			}
		}
		taxThreshold := int(totalPop)*100 + totalBuildings*5000
		if taxThreshold < 50000 {
			taxThreshold = 50000
		}
		if player.Credits > taxThreshold {
			tax := (player.Credits - taxThreshold) / 2000
			if tax > 0 {
				drain += tax
			}
		}

		// Deduct credit upkeep — if can't afford, shut down non-essential buildings
		if drain > 0 {
			if player.Credits >= drain {
				player.Credits -= drain
			} else {
				player.Credits = 0
				for _, sys := range systems {
					for _, e := range sys.Entities {
						if p, ok := e.(*entities.Planet); ok && p.Owner == player.Name {
							for _, be := range p.Buildings {
								if b, ok := be.(*entities.Building); ok {
									if b.BuildingType != entities.BuildingBase &&
										b.BuildingType != entities.BuildingTradingPost &&
										b.BuildingType != entities.BuildingMine &&
										b.IsOperational {
										b.IsOperational = false
									}
								}
							}
						}
					}
				}
			}
			result.CreditDrain += drain
		}

		// Re-enable buildings gradually once credits recover.
		if player.Credits > 200 {
			reEnabled := false
			for _, sys := range systems {
				if reEnabled {
					break
				}
				for _, e := range sys.Entities {
					if reEnabled {
						break
					}
					if p, ok := e.(*entities.Planet); ok && p.Owner == player.Name {
						for _, be := range p.Buildings {
							if b, ok := be.(*entities.Building); ok {
								if !b.IsOperational && b.BuildingType != entities.BuildingBase {
									b.IsOperational = true
									reEnabled = true
									break
								}
							}
						}
					}
				}
			}
		}
	}

	return result
}
