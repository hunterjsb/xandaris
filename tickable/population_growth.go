package tickable

import (
	"math"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&PopulationGrowthSystem{
		BaseSystem: NewBaseSystem("PopulationGrowth", 10),
	})
}

// PopulationGrowthSystem adjusts planetary populations based on capacity AND resource availability.
type PopulationGrowthSystem struct {
	*BaseSystem
	tickCounter int64
}

func (pgs *PopulationGrowthSystem) OnTick(tick int64) {
	pgs.tickCounter++

	interval := int64(pgs.GetPriority())
	if interval <= 0 {
		interval = 1
	}
	if pgs.tickCounter%interval != 0 {
		return
	}

	context := pgs.GetContext()
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
				pgs.updatePopulation(planet)
			}
		}
	}
}

func (pgs *PopulationGrowthSystem) updatePopulation(planet *entities.Planet) {
	if planet == nil {
		return
	}

	defer planet.RebalanceWorkforce()

	capacity := planet.GetTotalPopulationCapacity()
	if capacity <= 0 {
		if planet.Population > 500 {
			loss := int64(math.Ceil(float64(planet.Population) * 0.05))
			planet.Population -= loss
			if planet.Population < 500 {
				planet.Population = 500 // Colony core survives
			}
		} else if planet.Population <= 0 {
			planet.Population = 500 // Bootstrap dead colony
		}
		return
	}

	// Resource-dependent growth: population only grows if essential resources are available.
	// Water is the critical life-support resource.
	waterAvail := planet.GetStoredAmount(entities.ResWater)
	foodSufficiency := 1.0
	if planet.Population > 0 {
		// How many intervals of Water consumption can we sustain?
		// Consumption: 1 per 100 pop per interval
		waterNeeded := float64(planet.Population) / 100.0
		if waterNeeded > 0 {
			foodSufficiency = float64(waterAvail) / (waterNeeded * 5) // 5 intervals of buffer
			if foodSufficiency > 1.0 {
				foodSufficiency = 1.0
			}
		}
	}

	// If starving (less than 1 interval of water), population declines slowly
	if foodSufficiency < 0.2 {
		loss := int64(math.Ceil(float64(planet.Population) * 0.002)) // 0.2% decline per interval
		if loss < 1 {
			loss = 1
		}
		planet.Population -= loss
		if planet.Population < 500 {
			planet.Population = 500 // Colony core survives
		}
		return
	}

	if planet.Population >= capacity {
		return
	}

	habitability := float64(planet.Habitability) / 100.0
	if habitability <= 0 {
		return
	}

	// Logarithmic growth: fast when small, slows as population grows.
	// At 2000 pop: ~6/interval. At 5000: ~8. At 10000: ~9. At 20000: ~10.
	// This prevents exponential growth from overwhelming fixed resource production.
	if planet.Population == 0 {
		planet.Population = 100
		return
	}

	deficit := capacity - planet.Population
	if deficit <= 0 {
		return
	}

	// Happiness boosts growth: happy planets grow up to 1.5x faster
	happinessMultiplier := planet.ProductivityBonus
	if happinessMultiplier <= 0 {
		happinessMultiplier = 1.0
	}

	logFactor := math.Log10(float64(planet.Population)+1) / math.Log10(50000)
	growthRate := 0.004 * habitability * foodSufficiency * happinessMultiplier * (1.0 - logFactor)
	if growthRate < 0.0005 {
		growthRate = 0.0005
	}

	growth := int64(math.Round(float64(planet.Population) * growthRate))
	if growth < 1 {
		growth = 1
	}
	if growth > deficit {
		growth = deficit
	}

	planet.Population += growth
}
