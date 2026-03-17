package tickable

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradeHubSystem{
		BaseSystem: NewBaseSystem("TradeHubs", 67),
	})
}

// TradeHubSystem identifies and rewards systems that function as
// galactic trade hubs. A trade hub is a system with:
//   - Multiple factions' planets present
//   - High Trading Post levels
//   - Active cargo traffic
//   - Resource diversity
//
// Hub benefits:
//   - All trades in the system get +10% credits
//   - Ships refuel 2x faster
//   - Attracts NPC freighters (passive income)
//   - Announces hub status galaxy-wide
//
// Hub score is recalculated every 5000 ticks.
// Top 3 systems become official trade hubs.
type TradeHubSystem struct {
	*BaseSystem
	hubs       map[int]int  // systemID → hub score
	nextUpdate int64
}

func (ths *TradeHubSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := ths.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ths.hubs == nil {
		ths.hubs = make(map[int]int)
	}

	if ths.nextUpdate == 0 {
		ths.nextUpdate = tick + 3000
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Apply hub bonuses
	for sysID, score := range ths.hubs {
		if score < 50 {
			continue // not a hub
		}
		ths.applyHubBonuses(sysID, players, systems, game)
	}

	// Recalculate hub scores
	if tick >= ths.nextUpdate {
		ths.nextUpdate = tick + 5000
		ths.recalculateHubs(systems, players, game)
	}
}

func (ths *TradeHubSystem) applyHubBonuses(sysID int, players []*entities.Player, systems []*entities.System, game GameProvider) {
	for _, sys := range systems {
		if sys.ID != sysID {
			continue
		}

		// NPC freighter income: passive credits for planet owners
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
						// Hub income: 5cr per TP level per interval
						income := b.Level * 5
						for _, p := range players {
							if p != nil && p.Name == planet.Owner {
								p.Credits += income
								break
							}
						}
					}
				}
			}
		}
		break
	}
}

func (ths *TradeHubSystem) recalculateHubs(systems []*entities.System, players []*entities.Player, game GameProvider) {
	type hubCandidate struct {
		sysID   int
		sysName string
		score   int
	}
	var candidates []hubCandidate

	for _, sys := range systems {
		score := 0

		// Factor 1: faction diversity (more factions = more trade)
		factions := make(map[string]bool)
		tpLevels := 0
		resourceDiversity := make(map[string]bool)

		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}
			factions[planet.Owner] = true

			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
					tpLevels += b.Level
				}
			}

			for _, res := range []string{entities.ResIron, entities.ResWater, entities.ResOil,
				entities.ResFuel, entities.ResRareMetals, entities.ResHelium3, entities.ResElectronics} {
				if planet.GetStoredAmount(res) > 10 {
					resourceDiversity[res] = true
				}
			}
		}

		score += len(factions) * 15            // multi-faction presence
		score += tpLevels * 5                  // trading infrastructure
		score += len(resourceDiversity) * 3    // resource variety

		// Factor 2: ship traffic
		for _, p := range players {
			if p == nil {
				continue
			}
			for _, ship := range p.OwnedShips {
				if ship != nil && ship.CurrentSystem == sys.ID && ship.Status != entities.ShipStatusMoving {
					score += 2
					if ship.ShipType == entities.ShipTypeCargo {
						score += 5 // cargo ships worth more
					}
				}
			}
		}

		if score > 20 {
			candidates = append(candidates, hubCandidate{sys.ID, sys.Name, score})
		}
		ths.hubs[sys.ID] = score
	}

	// Sort and announce top hubs
	if len(candidates) == 0 {
		return
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if rand.Intn(3) == 0 { // don't announce every time
		msg := "🏗️ Trade Hub Rankings: "
		for i, c := range candidates {
			if i >= 3 {
				break
			}
			medal := "🥉"
			if i == 0 {
				medal = "🥇"
			} else if i == 1 {
				medal = "🥈"
			}
			msg += fmt.Sprintf("%s %s (score: %d) ", medal, c.sysName, c.score)
		}
		game.LogEvent("logistics", "", msg)
	}
}

// IsTradeHub returns whether a system qualifies as a trade hub.
func (ths *TradeHubSystem) IsTradeHub(systemID int) bool {
	if ths.hubs == nil {
		return false
	}
	return ths.hubs[systemID] >= 50
}

// GetHubScore returns the hub score for a system.
func (ths *TradeHubSystem) GetHubScore(systemID int) int {
	if ths.hubs == nil {
		return 0
	}
	return ths.hubs[systemID]
}
