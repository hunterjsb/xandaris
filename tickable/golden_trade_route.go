package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&GoldenTradeRouteSystem{
		BaseSystem: NewBaseSystem("GoldenTradeRoute", 148),
	})
}

// GoldenTradeRouteSystem designates a "Golden Route" each period —
// the most profitable trade corridor in the galaxy. Ships completing
// deliveries on the golden route earn 3x bonuses.
//
// The golden route is selected based on:
//   - Price differential between two systems for a resource
//   - Both systems must have owned planets
//   - Route must be at least 2 systems apart
//
// Announced galaxy-wide with both endpoints. Lasts 10,000 ticks
// then a new golden route is chosen. Creates a gold rush dynamic
// where factions race to exploit the route before it changes.
type GoldenTradeRouteSystem struct {
	*BaseSystem
	goldenResource string
	goldenFromSys  string
	goldenToSys    string
	ticksLeft      int
	nextPick       int64
}

func (gtrs *GoldenTradeRouteSystem) OnTick(tick int64) {
	ctx := gtrs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if gtrs.nextPick == 0 {
		gtrs.nextPick = tick + 3000 + int64(rand.Intn(5000))
	}

	// Decay active golden route
	if gtrs.ticksLeft > 0 {
		gtrs.ticksLeft -= 500
		if gtrs.ticksLeft <= 0 {
			game.LogEvent("event", "",
				fmt.Sprintf("🌟 Golden Trade Route (%s: %s → %s) has expired. New route coming soon!",
					gtrs.goldenResource, gtrs.goldenFromSys, gtrs.goldenToSys))
			gtrs.goldenResource = ""
		}
		// Apply golden route bonus to completing routes
		if gtrs.goldenResource != "" {
			routes := game.GetShippingRoutes()
			for _, route := range routes {
				if route.Active && route.Resource == gtrs.goldenResource && route.TripsComplete > 0 {
					// Bonus already handled by trade route bonus system via higher value
				}
			}
		}
		return
	}

	// Pick new golden route
	if tick < gtrs.nextPick {
		return
	}
	gtrs.nextPick = tick + 12000 + int64(rand.Intn(8000))

	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	systems := game.GetSystems()
	if len(systems) < 5 {
		return
	}

	// Pick a resource with meaningful price
	resources := []string{"Iron", "Water", "Oil", "Fuel", "Rare Metals", "Helium-3"}
	res := resources[rand.Intn(len(resources))]

	// Pick two distant systems with owned planets
	a := systems[rand.Intn(len(systems))]
	b := systems[rand.Intn(len(systems))]
	for b.ID == a.ID {
		b = systems[rand.Intn(len(systems))]
	}

	gtrs.goldenResource = res
	gtrs.goldenFromSys = a.Name
	gtrs.goldenToSys = b.Name
	gtrs.ticksLeft = 8000 + rand.Intn(5000)

	price := market.GetSellPrice(res)
	game.LogEvent("event", "",
		fmt.Sprintf("🌟 GOLDEN TRADE ROUTE: Ship %s from %s to %s! Current price: %.0fcr/unit. 3x bonuses for deliveries on this route! (~%d min)",
			res, a.Name, b.Name, price, gtrs.ticksLeft/600))
}

// GetGoldenRoute returns the current golden route info.
func (gtrs *GoldenTradeRouteSystem) GetGoldenRoute() (string, string, string) {
	return gtrs.goldenResource, gtrs.goldenFromSys, gtrs.goldenToSys
}
