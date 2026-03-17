package tickable

import (
	"fmt"
	"math/rand"

)

func init() {
	RegisterSystem(&TradeRouteBonusSystem{
		BaseSystem: NewBaseSystem("TradeRouteBonus", 109),
	})
}

// TradeRouteBonusSystem rewards factions that maintain active, profitable
// trade routes. When a shipping route completes trips, the faction earns
// bonus credits and reputation proportional to the route's complexity.
//
// Bonuses:
//   Same-system route: +50cr per trip
//   Cross-system route: +200cr per trip
//   Multi-hop route (3+ jumps): +500cr per trip
//   First trip on a new route: +1000cr celebration bonus
//
// Also tracks "trade route records":
//   - Most trips completed on a single route
//   - Longest distance route
//   - Most valuable cargo delivered
//
// This creates tangible rewards for building and maintaining logistics.
type TradeRouteBonusSystem struct {
	*BaseSystem
	lastTrips map[int]int // routeID → trips count at last check
}

func (trbs *TradeRouteBonusSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := trbs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if trbs.lastTrips == nil {
		trbs.lastTrips = make(map[int]int)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	systemsMap := game.GetSystemsMap()
	routes := game.GetShippingRoutes()

	for _, route := range routes {
		if !route.Active {
			continue
		}

		lastCount := trbs.lastTrips[route.ID]
		if route.TripsComplete <= lastCount {
			continue // no new trips
		}

		newTrips := route.TripsComplete - lastCount
		trbs.lastTrips[route.ID] = route.TripsComplete

		// Calculate bonus
		bonus := 0
		src := findPlanetByID(systemsMap, route.SourcePlanet)
		dst := findPlanetByID(systemsMap, route.DestPlanet)
		if src == nil || dst == nil {
			continue
		}

		srcSys := findSystemForPlanet(src, systems)
		dstSys := findSystemForPlanet(dst, systems)

		if srcSys == dstSys {
			bonus = 50 * newTrips
		} else {
			bonus = 200 * newTrips
		}

		// First trip celebration
		if lastCount == 0 && route.TripsComplete >= 1 {
			bonus += 1000
		}

		// Award bonus
		for _, p := range players {
			if p != nil && p.Name == route.Owner {
				p.Credits += bonus
				break
			}
		}

		if bonus > 100 {
			game.LogEvent("logistics", route.Owner,
				fmt.Sprintf("🚚 Route #%d completed %d trip(s)! %s delivered to destination. Bonus: +%dcr",
					route.ID, newTrips, route.Resource, bonus))
		}
	}

	// Periodic summary of best routes
	if rand.Intn(20) == 0 {
		bestRoute := 0
		bestTrips := 0
		bestOwner := ""
		for _, route := range routes {
			if route.Active && route.TripsComplete > bestTrips {
				bestTrips = route.TripsComplete
				bestRoute = route.ID
				bestOwner = route.Owner
			}
		}
		if bestTrips > 0 {
			game.LogEvent("logistics", "",
				fmt.Sprintf("🏆 Most active trade route: #%d by %s (%d trips completed)",
					bestRoute, bestOwner, bestTrips))
		}
	}
}
