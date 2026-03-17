package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&VictorySystem{
		BaseSystem: NewBaseSystem("Victory", 40),
	})
}

// VictorySystem checks for victory conditions and announces progress.
// Multiple paths to victory:
//
//   Economic Victory:  Accumulate 50,000,000 credits
//   Domination Victory: Control 20+ planets
//   Tech Victory:      Reach tech level 5.0 on any planet
//   Trade Victory:     Complete 100 shipping route trips
//   Population Victory: Reach 1,000,000 total population
//
// When a faction achieves a victory condition, it's announced galaxy-wide.
// The game continues (sandbox) but the achievement is permanent.
type VictorySystem struct {
	*BaseSystem
	achieved map[string]map[string]bool // player → victory type → achieved
}

type victoryCondition struct {
	name      string
	check     func(player *entities.Player, systems []*entities.System, routes int) bool
	threshold string
}

var victories = []victoryCondition{
	{
		name:      "Economic",
		threshold: "50,000,000 credits",
		check: func(p *entities.Player, _ []*entities.System, _ int) bool {
			return p.Credits >= 50_000_000
		},
	},
	{
		name:      "Domination",
		threshold: "20 planets",
		check: func(p *entities.Player, systems []*entities.System, _ int) bool {
			count := 0
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if planet, ok := e.(*entities.Planet); ok && planet.Owner == p.Name {
						count++
					}
				}
			}
			return count >= 20
		},
	},
	{
		name:      "Technology",
		threshold: "tech level 5.0",
		check: func(p *entities.Player, systems []*entities.System, _ int) bool {
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if planet, ok := e.(*entities.Planet); ok && planet.Owner == p.Name && planet.TechLevel >= 5.0 {
						return true
					}
				}
			}
			return false
		},
	},
	{
		name:      "Trade",
		threshold: "100 shipping trips",
		check: func(_ *entities.Player, _ []*entities.System, trips int) bool {
			return trips >= 100
		},
	},
	{
		name:      "Population",
		threshold: "1,000,000 citizens",
		check: func(p *entities.Player, systems []*entities.System, _ int) bool {
			var total int64
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if planet, ok := e.(*entities.Planet); ok && planet.Owner == p.Name {
						total += planet.Population
					}
				}
			}
			return total >= 1_000_000
		},
	},
}

func (vs *VictorySystem) OnTick(tick int64) {
	// Check every 1000 ticks (~100 seconds)
	if tick%1000 != 0 {
		return
	}

	ctx := vs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if vs.achieved == nil {
		vs.achieved = make(map[string]map[string]bool)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Count shipping trips per player
	playerTrips := make(map[string]int)
	for _, route := range game.GetShippingRoutes() {
		playerTrips[route.Owner] += route.TripsComplete
	}

	for _, player := range players {
		if player == nil {
			continue
		}

		if vs.achieved[player.Name] == nil {
			vs.achieved[player.Name] = make(map[string]bool)
		}

		trips := playerTrips[player.Name]

		for _, vc := range victories {
			if vs.achieved[player.Name][vc.name] {
				continue // already achieved
			}

			if vc.check(player, systems, trips) {
				vs.achieved[player.Name][vc.name] = true
				game.LogEvent("victory", player.Name,
					fmt.Sprintf("🏆 %s achieved %s Victory! (%s)",
						player.Name, vc.name, vc.threshold))
			}
		}
	}
}
