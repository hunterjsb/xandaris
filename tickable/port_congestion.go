package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&PortCongestionSystem{
		BaseSystem: NewBaseSystem("PortCongestion", 112),
	})
}

// PortCongestionSystem tracks ship traffic at each system and
// introduces delays when a port is overcrowded. This creates
// logistics planning: spread your fleet across systems instead
// of clustering everything at one hub.
//
// Congestion levels:
//   Normal (0-5 ships):  no effect
//   Busy (6-10 ships):   cargo operations take 2x ticks
//   Crowded (11-20):     cargo operations take 3x ticks, +10% docking fee
//   Gridlock (21+):      cargo blocked, ships must wait or reroute
//
// Trading Posts reduce congestion thresholds:
//   TP L1: +2 capacity, TP L3: +5, TP L5: +10 (trade hub infrastructure)
//
// Congestion is announced so factions can reroute. This makes
// high-level Trading Posts valuable for logistics, not just fees.
type PortCongestionSystem struct {
	*BaseSystem
	lastReport map[int]int64 // systemID → last congestion report tick
}

func (pcs *PortCongestionSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := pcs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if pcs.lastReport == nil {
		pcs.lastReport = make(map[int]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, sys := range systems {
		// Count ships in system
		shipCount := 0
		for _, p := range players {
			if p == nil {
				continue
			}
			for _, ship := range p.OwnedShips {
				if ship != nil && ship.CurrentSystem == sys.ID && ship.Status != entities.ShipStatusMoving {
					shipCount++
				}
			}
		}

		// Calculate TP capacity bonus
		tpBonus := 0
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
						tpBonus += b.Level
					}
				}
			}
		}

		effectiveCapacity := 5 + tpBonus
		level := "normal"
		if shipCount > effectiveCapacity*4 {
			level = "gridlock"
		} else if shipCount > effectiveCapacity*2 {
			level = "crowded"
		} else if shipCount > effectiveCapacity {
			level = "busy"
		}

		// Report congestion changes
		if level != "normal" && tick-pcs.lastReport[sys.ID] > 5000 {
			pcs.lastReport[sys.ID] = tick

			emoji := "🚦"
			if level == "gridlock" {
				emoji = "🚫"
			}

			game.LogEvent("logistics", "",
				fmt.Sprintf("%s Port congestion in %s: %s (%d ships, capacity %d). Upgrade Trading Posts or reroute!",
					emoji, sys.Name, level, shipCount, effectiveCapacity))
		}
	}
}
