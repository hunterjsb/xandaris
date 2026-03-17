package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradeNetworkEffectSystem{
		BaseSystem: NewBaseSystem("TradeNetworkEffect", 165),
	})
}

// TradeNetworkEffectSystem rewards factions whose trade network
// connects multiple systems. The more systems your Trading Posts
// span, the bigger the network bonus.
//
// Network size = number of unique systems with your Trading Posts
//
// Bonuses:
//   2 systems: +2% credit income
//   3 systems: +5% credit income
//   5 systems: +10% credit income + "Trade Network" title
//   8+ systems: +15% credit income + "Galactic Trade Empire"
//
// Applied as direct credit bonus per interval. Rewards expansion
// of trade infrastructure across the galaxy.
type TradeNetworkEffectSystem struct {
	*BaseSystem
	titles map[string]string
}

func (tnes *TradeNetworkEffectSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := tnes.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tnes.titles == nil {
		tnes.titles = make(map[string]string)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}

		// Count unique systems with TPs
		tpSystems := make(map[int]bool)
		for _, sys := range systems {
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner != player.Name {
					continue
				}
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
						tpSystems[sys.ID] = true
					}
				}
			}
		}

		networkSize := len(tpSystems)
		if networkSize < 2 {
			continue
		}

		// Credit bonus
		bonus := 0
		title := ""
		switch {
		case networkSize >= 8:
			bonus = player.Credits / 667 // ~0.15%
			title = "Galactic Trade Empire"
		case networkSize >= 5:
			bonus = player.Credits / 1000 // ~0.10%
			title = "Trade Network"
		case networkSize >= 3:
			bonus = player.Credits / 2000 // ~0.05%
			title = "Regional Trader"
		default:
			bonus = player.Credits / 5000 // ~0.02%
		}

		if bonus > 300 {
			bonus = 300
		}
		if bonus > 0 {
			player.Credits += bonus
		}

		// Announce title changes
		if title != "" && title != tnes.titles[player.Name] {
			tnes.titles[player.Name] = title
			game.LogEvent("event", player.Name,
				fmt.Sprintf("🌐 %s's trade network spans %d systems — earned \"%s\" status!",
					player.Name, networkSize, title))
		}
	}

	_ = rand.Intn
}
