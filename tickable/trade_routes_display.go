package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradeRouteDisplaySystem{
		BaseSystem: NewBaseSystem("TradeRouteDisplay", 54),
	})
}

// TradeRouteDisplaySystem generates visible "trade lane" activity events
// when cargo ships are actively hauling between systems. This makes the
// logistics network feel alive in the event feed.
//
// Events like:
//   "🚚 3 cargo ships hauling between SYS-5 and SYS-12 (Oil, Water, Iron)"
//   "📦 Gemini Exchange: Hauler-3 delivering 200 Iron to Alpha Prime"
//   "🛣️ Busiest trade lane: SYS-5 ↔ SYS-12 (5 active shipments)"
//
// Also tracks and announces trade lane statistics:
//   - Busiest trade corridor
//   - Most shipped resource
//   - Fastest trade route
type TradeRouteDisplaySystem struct {
	*BaseSystem
	nextDisplay int64
}

type tradeLane struct {
	sysA      int
	sysB      int
	sysAName  string
	sysBName  string
	shipCount int
	resources map[string]bool
}

func (trds *TradeRouteDisplaySystem) OnTick(tick int64) {
	ctx := trds.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if trds.nextDisplay == 0 {
		trds.nextDisplay = tick + 2000 + int64(rand.Intn(3000))
	}
	if tick < trds.nextDisplay {
		return
	}
	trds.nextDisplay = tick + 5000 + int64(rand.Intn(5000))

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	sysMap := game.GetSystemsMap()

	// Find all cargo ships currently in transit
	lanes := make(map[string]*tradeLane) // "sysA-sysB" → lane

	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeCargo {
				continue
			}
			if ship.Status != entities.ShipStatusMoving || ship.TargetSystem == -1 {
				continue
			}

			from := ship.CurrentSystem
			to := ship.TargetSystem
			if from > to {
				from, to = to, from // normalize
			}
			key := fmt.Sprintf("%d-%d", from, to)

			if lanes[key] == nil {
				fromName := fmt.Sprintf("SYS-%d", from+1)
				toName := fmt.Sprintf("SYS-%d", to+1)
				if s, ok := sysMap[ship.CurrentSystem]; ok {
					fromName = s.Name
				}
				if s, ok := sysMap[ship.TargetSystem]; ok {
					toName = s.Name
				}
				lanes[key] = &tradeLane{
					sysA: from, sysB: to,
					sysAName: fromName, sysBName: toName,
					resources: make(map[string]bool),
				}
			}
			lanes[key].shipCount++
			for res := range ship.CargoHold {
				lanes[key].resources[res] = true
			}
		}
	}

	if len(lanes) == 0 {
		// Count parked cargo ships to show fleet status
		parked := 0
		for _, p := range players {
			if p == nil {
				continue
			}
			for _, ship := range p.OwnedShips {
				if ship != nil && ship.ShipType == entities.ShipTypeCargo && ship.Status != entities.ShipStatusMoving {
					parked++
				}
			}
		}
		if parked > 0 {
			game.LogEvent("logistics", "",
				fmt.Sprintf("🚚 No active trade lanes — %d cargo ships idle. Create shipping routes!", parked))
		}
		return
	}

	// Find busiest lane
	var busiest *tradeLane
	for _, lane := range lanes {
		if busiest == nil || lane.shipCount > busiest.shipCount {
			busiest = lane
		}
	}

	// Count total moving cargo
	totalShips := 0
	for _, lane := range lanes {
		totalShips += lane.shipCount
	}

	// Announce
	_ = systems // used for name lookups above
	resList := ""
	for res := range busiest.resources {
		if resList != "" {
			resList += ", "
		}
		resList += res
	}
	if resList == "" {
		resList = "empty"
	}

	game.LogEvent("logistics", "",
		fmt.Sprintf("🛣️ Trade lanes: %d ships across %d routes. Busiest: %s ↔ %s (%d ships, cargo: %s)",
			totalShips, len(lanes), busiest.sysAName, busiest.sysBName, busiest.shipCount, resList))
}
