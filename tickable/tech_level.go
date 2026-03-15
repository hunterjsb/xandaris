package tickable

import (
	"math"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TechLevelSystem{
		BaseSystem: NewBaseSystem("TechLevel", 9),
	})
}

// TechLevelSystem accumulates technology from Electronics consumption.
// Higher tech level provides:
//   - Faster construction (+5% per tech level)
//   - Better mine extraction (+3% per tech level)
//   - Higher population capacity bonus
//
// Tech level grows logarithmically — easy to reach 1.0, hard to reach 5.0.
// Decays slowly if Electronics supply drops (use it or lose it).
type TechLevelSystem struct {
	*BaseSystem
}

func (tls *TechLevelSystem) OnTick(tick int64) {
	if tick%50 != 0 {
		return
	}

	ctx := tls.GetContext()
	if ctx == nil {
		return
	}

	players := ctx.GetPlayers()

	for _, player := range players {
		for _, planet := range player.OwnedPlanets {
			if planet != nil && planet.Population > 0 {
				updateTechLevel(planet)
			}
		}
	}
}

func updateTechLevel(planet *entities.Planet) {
	elecStored := float64(planet.GetStoredAmount(entities.ResElectronics))
	pop := float64(planet.Population)

	if pop <= 0 {
		return
	}

	// Electronics per capita determines tech growth
	// Target: 1 Electronics per 100 pop for max growth
	elecPerCapita := elecStored / (pop / 100.0)
	if elecPerCapita > 1.0 {
		elecPerCapita = 1.0
	}

	// Tech grows towards a target based on electronics availability
	// Target tech = log2(1 + electronics_per_capita * 10)
	// Max realistic target ≈ 3.5 at full electronics
	targetTech := math.Log2(1 + elecPerCapita*10)

	// Smooth approach via EMA
	alpha := 0.05
	if planet.TechLevel < targetTech {
		// Growing — slow
		planet.TechLevel = planet.TechLevel*(1-alpha) + targetTech*alpha
	} else {
		// Decaying — even slower (tech doesn't vanish instantly)
		planet.TechLevel = planet.TechLevel*(1-alpha*0.3) + targetTech*(alpha*0.3)
	}

	// Clamp
	if planet.TechLevel < 0 {
		planet.TechLevel = 0
	}
	if planet.TechLevel > 5 {
		planet.TechLevel = 5
	}
}
