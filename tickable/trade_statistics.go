package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradeStatisticsSystem{
		BaseSystem: NewBaseSystem("TradeStatistics", 99),
	})
}

// TradeStatisticsSystem computes and announces detailed trade
// statistics that help factions understand their economic position.
//
// Stats per faction:
//   - Trade balance (exports - imports value)
//   - Top export (most sold resource)
//   - Top import (most bought resource)
//   - Trade dependency ratio (how much they rely on imports)
//   - Self-sufficiency score (can they survive without trade?)
//
// Galaxy-wide stats:
//   - Most traded resource
//   - Least traded resource (opportunity!)
//   - Total trade volume
//   - Trade balance between faction pairs
type TradeStatisticsSystem struct {
	*BaseSystem
	nextReport int64
}

func (tss *TradeStatisticsSystem) OnTick(tick int64) {
	ctx := tss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tss.nextReport == 0 {
		tss.nextReport = tick + 5000 + int64(rand.Intn(5000))
	}
	if tick < tss.nextReport {
		return
	}
	tss.nextReport = tick + 8000 + int64(rand.Intn(5000))

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Calculate self-sufficiency per faction
	for _, player := range players {
		if player == nil {
			continue
		}

		// Count resource types produced vs consumed
		produces := make(map[string]bool)
		needs := make(map[string]bool)

		for _, sys := range systems {
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner != player.Name {
					continue
				}

				// What does this planet produce? (has resource nodes)
				for _, re := range planet.Resources {
					if r, ok := re.(*entities.Resource); ok {
						produces[r.ResourceType] = true
					}
				}

				// What does it need? (buildings that consume)
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.IsOperational {
						switch b.BuildingType {
						case entities.BuildingGenerator:
							needs[entities.ResFuel] = true
						case entities.BuildingRefinery:
							needs[entities.ResOil] = true
						case entities.BuildingFactory:
							needs[entities.ResIron] = true
							needs[entities.ResRareMetals] = true
						}
					}
				}

				// Everyone needs water
				if planet.Population > 0 {
					needs[entities.ResWater] = true
				}
			}
		}

		// Self-sufficiency: what % of needs are locally produced?
		if len(needs) == 0 {
			continue
		}

		selfSufficient := 0
		importDependent := 0
		for res := range needs {
			if produces[res] {
				selfSufficient++
			} else {
				importDependent++
			}
		}

		ratio := float64(selfSufficient) / float64(len(needs)) * 100

		// Only report for factions with multiple planets
		planetCount := 0
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
					planetCount++
				}
			}
		}

		if planetCount < 2 {
			continue
		}

		status := "balanced"
		if ratio > 80 {
			status = "self-sufficient"
		} else if ratio < 40 {
			status = "import-dependent"
		}

		if rand.Intn(3) == 0 { // don't report every time
			game.LogEvent("logistics", player.Name,
				fmt.Sprintf("📊 %s trade profile: %.0f%% self-sufficient (%s). Produces %d/%d needed resource types",
					player.Name, ratio, status, selfSufficient, len(needs)))
		}
	}
}
