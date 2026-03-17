package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AutoTradingPostSystem{
		BaseSystem: NewBaseSystem("AutoTradingPost", 151),
	})
}

// AutoTradingPostSystem ensures every owned planet has at least a
// basic Trading Post. Without a TP, planets can't participate in
// local exchange, can't receive docking fees, and can't be part of
// the trade network. Many planets are missing TPs.
//
// Builds a L1 Trading Post on any owned planet that:
//   - Has population > 500
//   - Has no Trading Post
//   - Owner has > 3000 credits
//   - Has at least one other building (not a fresh colony)
//
// Max 1 auto-TP per faction per 5000 ticks.
type AutoTradingPostSystem struct {
	*BaseSystem
	lastBuild map[string]int64
}

func (atps *AutoTradingPostSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := atps.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if atps.lastBuild == nil {
		atps.lastBuild = make(map[string]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.Population < 500 {
				continue
			}

			if tick-atps.lastBuild[planet.Owner] < 5000 {
				continue
			}

			// Check for existing TP
			hasTP := false
			buildingCount := 0
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					buildingCount++
					if b.BuildingType == entities.BuildingTradingPost {
						hasTP = true
					}
				}
			}

			if hasTP || buildingCount < 2 {
				continue
			}

			// Find owner
			for _, p := range players {
				if p == nil || p.Name != planet.Owner || p.Credits < 3000 {
					continue
				}

				p.Credits -= 2000
				atps.lastBuild[p.Name] = tick
				game.AIBuildOnPlanet(planet, entities.BuildingTradingPost, planet.Owner, sys.ID)

				game.LogEvent("logistics", planet.Owner,
					fmt.Sprintf("🏗️ Auto-building Trading Post on %s — opens market access + docking fees!",
						planet.Name))
				break
			}
		}
	}
}
