package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradePortUpgradeSystem{
		BaseSystem: NewBaseSystem("TradePortUpgrade", 115),
	})
}

// TradePortUpgradeSystem auto-upgrades Trading Posts on planets that
// generate high docking revenue. When a TP earns enough from foreign
// ships, the profits are reinvested into upgrading the TP level.
//
// Auto-upgrade thresholds (cumulative docking revenue):
//   L1 → L2: 5,000cr earned
//   L2 → L3: 15,000cr earned
//   L3 → L4: 50,000cr earned
//   L4 → L5: 200,000cr earned
//
// This creates a virtuous cycle: more traffic → more revenue →
// better port → more capacity → more traffic.
//
// Also tracks "Port of the Year" — the TP with the most foreign
// ship traffic, announced periodically.
type TradePortUpgradeSystem struct {
	*BaseSystem
	portRevenue map[int]int // planetID → cumulative docking revenue tracked
	nextAward   int64
}

var tpUpgradeThresholds = map[int]int{
	1: 5000,
	2: 15000,
	3: 50000,
	4: 200000,
}

func (tpus *TradePortUpgradeSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := tpus.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tpus.portRevenue == nil {
		tpus.portRevenue = make(map[int]int)
	}

	if tpus.nextAward == 0 {
		tpus.nextAward = tick + 10000
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Count foreign ships per planet and accumulate revenue
	bestPort := ""
	bestPortOwner := ""
	bestTraffic := 0

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			// Find TP
			var tp *entities.Building
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
					tp = b
					break
				}
			}
			if tp == nil {
				continue
			}

			// Count foreign ships
			foreignCount := 0
			for _, p := range players {
				if p == nil || p.Name == planet.Owner {
					continue
				}
				for _, ship := range p.OwnedShips {
					if ship != nil && ship.CurrentSystem == sys.ID && ship.Status != entities.ShipStatusMoving {
						foreignCount++
					}
				}
			}

			if foreignCount > bestTraffic {
				bestTraffic = foreignCount
				bestPort = planet.Name
				bestPortOwner = planet.Owner
			}

			// Accumulate revenue estimate
			pid := planet.GetID()
			tpus.portRevenue[pid] += foreignCount * tp.Level * 5

			// Check for auto-upgrade
			threshold, canUpgrade := tpUpgradeThresholds[tp.Level]
			if canUpgrade && tpus.portRevenue[pid] >= threshold {
				tp.Level++
				tpus.portRevenue[pid] = 0 // reset for next tier

				game.LogEvent("logistics", planet.Owner,
					fmt.Sprintf("🏗️ %s Trading Post auto-upgraded to L%d! High traffic justified expansion. More capacity + lower fees!",
						planet.Name, tp.Level))
			}
		}
	}

	// Port of the Year award
	if tick >= tpus.nextAward && bestTraffic > 5 {
		tpus.nextAward = tick + 15000 + int64(rand.Intn(10000))

		// Bonus to port owner
		for _, p := range players {
			if p != nil && p.Name == bestPortOwner {
				bonus := bestTraffic * 100
				p.Credits += bonus
				game.LogEvent("event", bestPortOwner,
					fmt.Sprintf("🏆 Port of the Year: %s (%s)! %d foreign ships. Bonus: +%dcr",
						bestPort, bestPortOwner, bestTraffic, bonus))
				break
			}
		}
	}
}
