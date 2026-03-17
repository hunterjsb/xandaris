package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradeNetworkScoreSystem{
		BaseSystem: NewBaseSystem("TradeNetworkScore", 120),
	})
}

// TradeNetworkScoreSystem evaluates the galaxy's trade connectivity
// and rates each faction's logistics network quality.
//
// Network score components per faction:
//   - Systems reached: how many systems have your ships? (+10 per system)
//   - Route coverage: shipping routes covering different resources (+20 each)
//   - TP network: Trading Posts across multiple systems (+15 per TP)
//   - Cargo capacity: total cargo ship hold space (+1 per 100 units)
//   - Delivery rate: trips completed per route (+5 per avg trip)
//
// Grades:
//   S: 200+ (logistics empire)
//   A: 150-199 (excellent network)
//   B: 100-149 (solid foundation)
//   C: 50-99 (developing)
//   D: 0-49 (isolated)
//
// Announced periodically with faction grades.
type TradeNetworkScoreSystem struct {
	*BaseSystem
	nextReport int64
}

func (tnss *TradeNetworkScoreSystem) OnTick(tick int64) {
	ctx := tnss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tnss.nextReport == 0 {
		tnss.nextReport = tick + 5000 + int64(rand.Intn(5000))
	}
	if tick < tnss.nextReport {
		return
	}
	tnss.nextReport = tick + 10000 + int64(rand.Intn(5000))

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	routes := game.GetShippingRoutes()

	msg := "🌐 Trade Network Grades: "
	hasData := false

	for _, player := range players {
		if player == nil {
			continue
		}

		score := 0

		// Systems reached
		systemsReached := make(map[int]bool)
		totalCargoCap := 0
		for _, ship := range player.OwnedShips {
			if ship == nil {
				continue
			}
			systemsReached[ship.CurrentSystem] = true
			if ship.ShipType == entities.ShipTypeCargo {
				totalCargoCap += ship.MaxCargo
			}
		}
		score += len(systemsReached) * 10
		score += totalCargoCap / 100

		// TP network
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
					for _, be := range planet.Buildings {
						if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
							score += 15
						}
					}
				}
			}
		}

		// Route coverage
		routeResources := make(map[string]bool)
		totalTrips := 0
		routeCount := 0
		for _, route := range routes {
			if route.Owner == player.Name && route.Active {
				routeResources[route.Resource] = true
				totalTrips += route.TripsComplete
				routeCount++
			}
		}
		score += len(routeResources) * 20
		if routeCount > 0 {
			avgTrips := totalTrips / routeCount
			score += avgTrips * 5
		}

		// Grade
		grade := "D"
		switch {
		case score >= 200:
			grade = "S"
		case score >= 150:
			grade = "A"
		case score >= 100:
			grade = "B"
		case score >= 50:
			grade = "C"
		}

		if score > 10 {
			msg += fmt.Sprintf("%s:%s(%d) ", player.Name, grade, score)
			hasData = true
		}
	}

	if hasData {
		game.LogEvent("intel", "", msg)
	}
}
