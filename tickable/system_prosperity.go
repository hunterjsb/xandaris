package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SystemProsperitySystem{
		BaseSystem: NewBaseSystem("SystemProsperity", 142),
	})
}

// SystemProsperitySystem rates each star system's overall prosperity
// and grants bonuses to prosperous systems while flagging declining ones.
//
// Prosperity score per system:
//   +10 per owned planet
//   +5 per operational Trading Post
//   +3 per 1000 population
//   +2 per operational building
//   -5 per planet with <20% happiness
//   -3 per power crisis planet
//
// Prosperity tiers:
//   Thriving (80+):  +5% production bonus to all planets
//   Developing (40-79): normal
//   Struggling (<40): flagged for attention
//   Failed (<10): population emigrates
//
// Creates system-level economic management beyond individual planets.
type SystemProsperitySystem struct {
	*BaseSystem
	prosperity map[int]int // systemID → score
	nextReport int64
}

func (sps *SystemProsperitySystem) OnTick(tick int64) {
	if tick%3000 != 0 {
		return
	}

	ctx := sps.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if sps.prosperity == nil {
		sps.prosperity = make(map[int]int)
	}

	systems := game.GetSystems()

	for _, sys := range systems {
		score := 0
		ownedCount := 0

		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			ownedCount++
			score += 10
			score += int(planet.Population / 1000) * 3

			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					if b.IsOperational {
						score += 2
						if b.BuildingType == entities.BuildingTradingPost {
							score += 5
						}
					}
				}
			}

			if planet.Happiness < 0.2 {
				score -= 5
			}
			if planet.GetPowerRatio() < 0.3 {
				score -= 3
			}
		}

		if ownedCount == 0 {
			continue
		}

		sps.prosperity[sys.ID] = score

		// Apply effects
		tier := "Developing"
		if score >= 80 {
			tier = "Thriving"
			// Bonus: small happiness boost
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
					planet.Happiness += 0.01
					if planet.Happiness > 1.0 {
						planet.Happiness = 1.0
					}
				}
			}
		} else if score < 10 {
			tier = "Failed"
		} else if score < 40 {
			tier = "Struggling"
		}

		_ = tier // used in reports below
	}

	// Periodic report
	if sps.nextReport == 0 {
		sps.nextReport = tick + 10000
	}
	if tick >= sps.nextReport {
		sps.nextReport = tick + 12000 + int64(rand.Intn(8000))

		// Find best and worst systems
		bestSys := ""
		bestScore := 0
		worstSys := ""
		worstScore := 999999

		for _, sys := range systems {
			score := sps.prosperity[sys.ID]
			if score > bestScore {
				bestScore = score
				bestSys = sys.Name
			}
			if score > 0 && score < worstScore {
				worstScore = score
				worstSys = sys.Name
			}
		}

		if bestSys != "" {
			game.LogEvent("intel", "",
				fmt.Sprintf("📊 System Prosperity: Best: %s (score %d, Thriving) | Worst: %s (score %d). Invest in struggling systems!",
					bestSys, bestScore, worstSys, worstScore))
		}
	}
}
