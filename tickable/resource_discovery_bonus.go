package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ResourceDiscoveryBonusSystem{
		BaseSystem: NewBaseSystem("ResourceDiscoveryBonus", 156),
	})
}

// ResourceDiscoveryBonusSystem grants bonus credits when a faction
// first stocks a new resource type on any of their planets. This
// rewards resource diversification — the more types you stock,
// the bigger the bonuses.
//
// First time stocking each resource type:
//   Iron:        +200cr (common, easy)
//   Water:       +200cr
//   Oil:         +300cr
//   Fuel:        +500cr (requires production chain)
//   Rare Metals: +800cr (uncommon)
//   Helium-3:    +1000cr (rare)
//   Electronics: +1500cr (requires factory)
//
// Also tracks "resource collector" achievement: stocking all 7
// types across your empire = 5000cr bonus + announcement.
type ResourceDiscoveryBonusSystem struct {
	*BaseSystem
	discovered map[string]map[string]bool // faction → resource → already discovered
	allSeven   map[string]bool            // faction → already got all 7
}

var discoveryBonuses = map[string]int{
	entities.ResIron:        200,
	entities.ResWater:       200,
	entities.ResOil:         300,
	entities.ResFuel:        500,
	entities.ResRareMetals:  800,
	entities.ResHelium3:     1000,
	entities.ResElectronics: 1500,
}

func (rdbs *ResourceDiscoveryBonusSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := rdbs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if rdbs.discovered == nil {
		rdbs.discovered = make(map[string]map[string]bool)
		rdbs.allSeven = make(map[string]bool)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}

		if rdbs.discovered[player.Name] == nil {
			rdbs.discovered[player.Name] = make(map[string]bool)
		}

		// Scan all owned planets for stocked resources
		for _, sys := range systems {
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner != player.Name {
					continue
				}

				for res, bonus := range discoveryBonuses {
					if rdbs.discovered[player.Name][res] {
						continue
					}
					if planet.GetStoredAmount(res) >= 10 {
						rdbs.discovered[player.Name][res] = true
						player.Credits += bonus
						game.LogEvent("event", player.Name,
							fmt.Sprintf("🔬 %s first stocked %s! Resource discovery bonus: +%dcr",
								player.Name, res, bonus))
					}
				}
			}
		}

		// Check for all 7
		if !rdbs.allSeven[player.Name] && len(rdbs.discovered[player.Name]) >= 7 {
			rdbs.allSeven[player.Name] = true
			player.Credits += 5000
			game.LogEvent("event", player.Name,
				fmt.Sprintf("🏆 %s collected ALL 7 resource types! Master Collector bonus: +5000cr!",
					player.Name))
		}
	}

	_ = rand.Intn // suppress
}
