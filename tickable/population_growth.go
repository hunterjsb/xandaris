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

// PopulationGrowthSystem adjusts planetary populations toward their effective capacity.
type PopulationGrowthSystem struct {
	*BaseSystem
	tickCounter int64
}

// OnTick updates population levels at a cadence tied to system priority.
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
			pgs.updatePopulation(planet)
		}
	}
}

func (pgs *PopulationGrowthSystem) updatePopulation(planet *entities.Planet) {
	if planet == nil {
		return
	}

	capacity := planet.GetTotalPopulationCapacity()
	if capacity <= 0 {
		if planet.Population > 0 {
			loss := int64(math.Ceil(float64(planet.Population) * 0.05))
			if loss < 1 {
				loss = 1
			}
			planet.Population -= loss
			if planet.Population < 0 {
				planet.Population = 0
			}
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

	growthRate := 0.005 * habitability
	if planet.Population == 0 {
		initialGrowth := math.Round(float64(capacity) * 0.0005)
		growth := int64(math.Max(1, initialGrowth))
		planet.Population = growth
		return
	}

	deficit := capacity - planet.Population
	if deficit <= 0 {
		return
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
