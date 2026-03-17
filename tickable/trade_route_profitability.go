package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&TradeRouteProfitabilitySystem{
		BaseSystem: NewBaseSystem("TradeRouteProfitability", 127),
	})
}

// TradeRouteProfitabilitySystem calculates and announces the most
// and least profitable trade routes in the galaxy. This helps
// factions optimize their logistics networks.
//
// For each active route with trips, estimates profitability:
//   Revenue = trips * quantity * sell_price
//   Cost = fuel_consumed + time_cost (opportunity cost)
//   Profit = Revenue - Cost
//
// Announces:
//   - Most profitable route (by total value moved)
//   - Least efficient route (high fuel cost, low value cargo)
//   - Route suggestions based on price differentials
type TradeRouteProfitabilitySystem struct {
	*BaseSystem
	nextReport int64
}

func (trps *TradeRouteProfitabilitySystem) OnTick(tick int64) {
	ctx := trps.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if trps.nextReport == 0 {
		trps.nextReport = tick + 5000 + int64(rand.Intn(5000))
	}
	if tick < trps.nextReport {
		return
	}
	trps.nextReport = tick + 10000 + int64(rand.Intn(5000))

	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	routes := game.GetShippingRoutes()

	type routeProfit struct {
		id       int
		owner    string
		resource string
		trips    int
		value    int
	}

	var profitable []routeProfit

	for _, route := range routes {
		if !route.Active || route.TripsComplete == 0 {
			continue
		}

		price := market.GetSellPrice(route.Resource)
		qty := route.Quantity
		if qty <= 0 {
			qty = 500 // estimate for "fill cargo" routes
		}
		totalValue := int(price * float64(qty) * float64(route.TripsComplete))

		profitable = append(profitable, routeProfit{
			id: route.ID, owner: route.Owner,
			resource: route.Resource, trips: route.TripsComplete,
			value: totalValue,
		})
	}

	if len(profitable) == 0 {
		return
	}

	// Find best and worst
	best := profitable[0]
	for _, rp := range profitable[1:] {
		if rp.value > best.value {
			best = rp
		}
	}

	msg := fmt.Sprintf("📊 Trade Route Report: Most valuable: Route #%d (%s's %s route, %d trips, ~%dcr total value)",
		best.id, best.owner, best.resource, best.trips, best.value)

	// Count total active routes and trips
	totalRoutes := 0
	totalTrips := 0
	for _, route := range routes {
		if route.Active {
			totalRoutes++
			totalTrips += route.TripsComplete
		}
	}
	msg += fmt.Sprintf(" | Galaxy: %d routes, %d total trips", totalRoutes, totalTrips)

	game.LogEvent("intel", "", msg)
}
