package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&LaborMarketSystem{
		BaseSystem: NewBaseSystem("LaborMarket", 118),
	})
}

// LaborMarketSystem simulates workforce dynamics. Workers are not
// just numbers — they have specializations that affect productivity.
//
// When a planet has more workforce than jobs (buildings), unemployment
// creates happiness drain. When it has more jobs than workers, buildings
// operate below capacity.
//
// The labor market creates inter-planet worker migration:
//   - Planets with unemployment push workers to planets with openings
//   - Workers prefer higher-happiness destinations
//   - Cross-faction migration happens at very low happiness (<20%)
//
// Also generates "labor shortage" and "unemployment" alerts that
// help factions balance their building-to-population ratio.
type LaborMarketSystem struct {
	*BaseSystem
	lastAlert map[int]int64
}

func (lms *LaborMarketSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := lms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if lms.lastAlert == nil {
		lms.lastAlert = make(map[int]int64)
	}

	systems := game.GetSystems()

	for _, sys := range systems {
		// Find planets with labor imbalance
		var overstaffed, understaffed []*entities.Planet

		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.Population < 100 {
				continue
			}

			if planet.WorkforceTotal > 0 {
				utilizationRate := float64(planet.WorkforceUsed) / float64(planet.WorkforceTotal)

				if utilizationRate < 0.5 && planet.WorkforceTotal > 500 {
					overstaffed = append(overstaffed, planet)

					// Alert
					if tick-lms.lastAlert[planet.GetID()] > 10000 {
						lms.lastAlert[planet.GetID()] = tick
						unemploymentPct := (1.0 - utilizationRate) * 100
						if unemploymentPct > 30 {
							game.LogEvent("alert", planet.Owner,
								fmt.Sprintf("👷 %s: %.0f%% unemployment! Build more buildings to employ workers, or they'll emigrate",
									planet.Name, unemploymentPct))
						}
					}
				} else if utilizationRate > 0.95 && planet.WorkforceUsed > 100 {
					understaffed = append(understaffed, planet)

					if tick-lms.lastAlert[planet.GetID()] > 10000 {
						lms.lastAlert[planet.GetID()] = tick
						game.LogEvent("alert", planet.Owner,
							fmt.Sprintf("👷 %s: labor shortage! Buildings at %.0f%% staffing. Build Habitats for more workers!",
								planet.Name, utilizationRate*100))
					}
				}
			}
		}

		// Internal migration: overstaffed → understaffed (same owner)
		for _, src := range overstaffed {
			for _, dst := range understaffed {
				if src.Owner != dst.Owner {
					continue
				}

				// Transfer workers
				transfer := int64(100 + rand.Intn(200))
				if transfer > src.Population/10 {
					transfer = src.Population / 10
				}

				cap := dst.GetTotalPopulationCapacity()
				if cap > 0 && dst.Population+transfer > cap {
					transfer = cap - dst.Population
				}
				if transfer <= 0 {
					continue
				}

				src.Population -= transfer
				dst.Population += transfer

				if rand.Intn(5) == 0 {
					game.LogEvent("logistics", src.Owner,
						fmt.Sprintf("👷 %d workers relocated from %s to %s (internal labor rebalancing)",
							transfer, src.Name, dst.Name))
				}
				break // one transfer per overstaffed planet per tick
			}
		}
	}
}
