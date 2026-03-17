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

	// Use system entity planets (authoritative) instead of player.OwnedPlanets (stale)
	game := ctx.GetGame()
	if game == nil {
		return
	}
	for _, sys := range game.GetSystems() {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" && planet.Population > 0 {
				computeHappiness(planet)
			}
		}
	}
}

// resourceWeight defines how much each resource affects happiness.
var resourceWeights = map[string]float64{
	entities.ResWater:      3.0, // Most critical
	entities.ResIron:       1.0,
	entities.ResOil:        1.0,
	entities.ResFuel:       0.5,
	entities.ResElectronics: 1.0,
	entities.ResRareMetals: 0.3,
	entities.ResHelium3:    0.3,
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

	// Power matters but isn't existential — a primitive colony survives without it.
	// Weight 2.0, with a floor of 0.3 so zero power doesn't obliterate happiness.
	powerRatio := planet.GetPowerRatio()
	if planet.PowerConsumed > 0 {
		powerScore := 0.3 + 0.7*powerRatio // floor at 0.3 even with zero power
		totalWeight += 2.0
		weightedSum += powerScore * 2.0
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
