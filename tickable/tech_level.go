package tickable

import (
	"fmt"
	"math"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TechLevelSystem{
		BaseSystem:    NewBaseSystem("TechLevel", 9),
		prevTechLevel: make(map[int]float64),
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
	prevTechLevel map[int]float64 // planet ID -> previous tech level (for milestone detection)
}

// techMilestones defines the thresholds that trigger alerts when crossed.
var techMilestones = []struct {
	threshold float64
	message   string
}{
	{0.5, "reached Early Industrial — Refinery unlocked!"},
	{1.0, "reached Industrial era — Factory and Shipyard unlocked!"},
	{2.0, "reached Space Age — Fusion Reactor unlocked!"},
	{3.5, "reached Advanced era — all bonuses maximized!"},
}

func (tls *TechLevelSystem) OnTick(tick int64) {
	if tick%50 != 0 {
		return
	}

	ctx := tls.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	// On first tick, seed prevTechLevel from current state to avoid
	// re-firing all milestones after server restart / save load.
	if len(tls.prevTechLevel) == 0 {
		for _, sys := range game.GetSystems() {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
					tls.prevTechLevel[planet.GetID()] = planet.TechLevel
				}
			}
		}
	}

	for _, sys := range game.GetSystems() {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" && planet.Population > 0 {
				prevLevel := tls.prevTechLevel[planet.GetID()]
				updateTechLevel(planet)
				newLevel := planet.TechLevel

				// Check milestone crossings (only fires on NEW transitions)
				for _, m := range techMilestones {
					if prevLevel < m.threshold && newLevel >= m.threshold {
						game.LogEvent("event", planet.Owner,
							fmt.Sprintf("%s %s", planet.Name, m.message))
					}
				}
				tls.prevTechLevel[planet.GetID()] = newLevel
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
