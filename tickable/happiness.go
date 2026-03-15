package tickable

import (
	"math"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&HappinessSystem{
		BaseSystem: NewBaseSystem("Happiness", 8),
	})
}

// HappinessSystem computes per-planet happiness from resource fulfillment.
// Happiness drives population growth rate, worker productivity, and credit generation.
//
// Happiness = weighted average of resource sufficiency ratios:
//   - Water (weight 3): critical for survival
//   - Iron, Oil (weight 1): industrial needs
//   - Electronics (weight 1): technology needs
//   - Fuel (weight 0.5): energy needs
//
// A sufficiency ratio = stored / (consumption × buffer intervals).
// Fully stocked = 1.0, empty = 0.0.
//
// Productivity bonus = 0.5 + happiness (range 0.5x - 1.5x).
type HappinessSystem struct {
	*BaseSystem
}

func (hs *HappinessSystem) OnTick(tick int64) {
	if tick%10 != 0 {
		return
	}

	ctx := hs.GetContext()
	if ctx == nil {
		return
	}

	playersIface := ctx.GetPlayers()
	if playersIface == nil {
		return
	}
	players, ok := playersIface.([]*entities.Player)
	if !ok {
		return
	}

	for _, player := range players {
		for _, planet := range player.OwnedPlanets {
			if planet != nil && planet.Population > 0 {
				computeHappiness(planet)
			}
		}
	}
}

// resourceWeight defines how much each resource affects happiness.
var resourceWeights = map[string]float64{
	"Water":       3.0, // Most critical
	"Iron":        1.0,
	"Oil":         1.0,
	"Fuel":        0.5,
	"Electronics": 1.0,
	"Rare Metals": 0.3,
	"Helium-3":    0.3,
}

func computeHappiness(planet *entities.Planet) {
	pop := float64(planet.Population)
	if pop <= 0 {
		planet.Happiness = 0.5
		planet.ProductivityBonus = 1.0
		return
	}

	totalWeight := 0.0
	weightedSum := 0.0

	for _, rate := range economy.PopulationConsumption {
		weight, ok := resourceWeights[rate.ResourceType]
		if !ok {
			weight = 0.5
		}

		// How much does this planet consume per interval?
		consumption := pop / rate.PopDivisor * rate.PerPopulation
		if consumption < 0.5 {
			// Negligible consumption — don't penalize small colonies for luxury goods
			continue
		}

		// How many intervals of buffer do we have? (target: 10 intervals)
		stored := float64(planet.GetStoredAmount(rate.ResourceType))
		bufferIntervals := 10.0
		sufficiency := stored / (consumption * bufferIntervals)
		if sufficiency > 1.0 {
			sufficiency = 1.0
		}

		totalWeight += weight
		weightedSum += sufficiency * weight
	}

	if totalWeight <= 0 {
		planet.Happiness = 0.5
		planet.ProductivityBonus = 1.0
		return
	}

	// Smooth towards target using EMA (prevent wild swings)
	targetHappiness := weightedSum / totalWeight
	alpha := 0.1 // Slow adjustment
	planet.Happiness = planet.Happiness*(1-alpha) + targetHappiness*alpha

	// Clamp
	planet.Happiness = math.Max(0, math.Min(1.0, planet.Happiness))

	// Productivity bonus: 0.5x at 0 happiness, 1.0x at 0.5, 1.5x at 1.0
	planet.ProductivityBonus = 0.5 + planet.Happiness
}
