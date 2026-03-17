package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&LogisticsBottleneckSystem{
		BaseSystem: NewBaseSystem("LogisticsBottleneck", 167),
	})
}

// LogisticsBottleneckSystem identifies the single biggest bottleneck
// in each faction's logistics and announces it. Instead of many
// diagnostics, this answers: "what ONE thing should I fix?"
//
// Bottleneck priority:
//   1. No cargo ships at all → "Build a Cargo ship!"
//   2. All cargo ships at 0 fuel → "Refuel your fleet!"
//   3. No Trading Posts → "Build Trading Posts!"
//   4. No shipping routes → "Create shipping routes!"
//   5. Routes but no trips → "Check route source stock!"
//   6. Everything working → "Logistics healthy!"
type LogisticsBottleneckSystem struct {
	*BaseSystem
	nextCheck int64
}

func (lbs *LogisticsBottleneckSystem) OnTick(tick int64) {
	ctx := lbs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if lbs.nextCheck == 0 {
		lbs.nextCheck = tick + 5000 + int64(rand.Intn(5000))
	}
	if tick < lbs.nextCheck {
		return
	}
	lbs.nextCheck = tick + 8000 + int64(rand.Intn(5000))

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	routes := game.GetShippingRoutes()

	for _, player := range players {
		if player == nil {
			continue
		}

		// Count fleet composition
		cargoShips := 0
		cargoNoFuel := 0
		for _, ship := range player.OwnedShips {
			if ship == nil {
				continue
			}
			if ship.ShipType == entities.ShipTypeCargo {
				cargoShips++
				if ship.CurrentFuel == 0 {
					cargoNoFuel++
				}
			}
		}

		// Count TPs
		tpCount := 0
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
					for _, be := range planet.Buildings {
						if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
							tpCount++
						}
					}
				}
			}
		}

		// Count routes
		factionRoutes := 0
		factionTrips := 0
		for _, r := range routes {
			if r.Owner == player.Name && r.Active {
				factionRoutes++
				factionTrips += r.TripsComplete
			}
		}

		// Determine bottleneck
		bottleneck := ""
		switch {
		case cargoShips == 0:
			bottleneck = "🔧 #1 bottleneck: NO CARGO SHIPS. Build freighters at a Shipyard to start trading!"
		case cargoNoFuel == cargoShips:
			bottleneck = "🔧 #1 bottleneck: ALL CARGO SHIPS OUT OF FUEL. Build Refineries to produce Fuel!"
		case tpCount == 0:
			bottleneck = "🔧 #1 bottleneck: NO TRADING POSTS. Build TPs to enable market access!"
		case factionRoutes == 0:
			bottleneck = "🔧 #1 bottleneck: NO SHIPPING ROUTES. Create routes to move resources between planets!"
		case factionRoutes > 0 && factionTrips == 0:
			bottleneck = "🔧 #1 bottleneck: ROUTES NOT COMPLETING. Check source planet stock and ship fuel!"
		default:
			// Healthy
			if rand.Intn(5) == 0 {
				bottleneck = fmt.Sprintf("✅ Logistics healthy: %d cargo ships, %d TPs, %d routes (%d trips)",
					cargoShips, tpCount, factionRoutes, factionTrips)
			}
		}

		if bottleneck != "" {
			game.LogEvent("logistics", player.Name,
				fmt.Sprintf("%s: %s", player.Name, bottleneck))
		}
	}
}
