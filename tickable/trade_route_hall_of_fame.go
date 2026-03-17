package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&TradeRouteHallOfFameSystem{
		BaseSystem: NewBaseSystem("TradeRouteHallOfFame", 157),
	})
}

// TradeRouteHallOfFameSystem tracks the all-time best performing
// shipping routes and celebrates them. Routes that consistently
// deliver become legendary trade lanes.
//
// Hall of Fame criteria:
//   10 trips:  "Established Route" — named and tracked
//   50 trips:  "Major Trade Lane" — +1000cr bonus
//   100 trips: "Legendary Route" — +5000cr + galactic announcement
//   500 trips: "Eternal Trade Lane" — +20000cr + permanent monument
//
// Only the first route to reach each milestone gets the bonus.
// Creates long-term goals for logistics management.
type TradeRouteHallOfFameSystem struct {
	*BaseSystem
	milestones map[int]int // routeID → highest milestone reached
	nextCheck  int64
}

func (trhof *TradeRouteHallOfFameSystem) OnTick(tick int64) {
	ctx := trhof.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if trhof.milestones == nil {
		trhof.milestones = make(map[int]int)
	}

	if trhof.nextCheck == 0 {
		trhof.nextCheck = tick + 3000
	}
	if tick < trhof.nextCheck {
		return
	}
	trhof.nextCheck = tick + 5000 + int64(rand.Intn(3000))

	players := game.GetPlayers()
	routes := game.GetShippingRoutes()

	milestones := []struct {
		threshold int
		title     string
		bonus     int
	}{
		{500, "Eternal Trade Lane", 20000},
		{100, "Legendary Route", 5000},
		{50, "Major Trade Lane", 1000},
		{10, "Established Route", 200},
	}

	for _, route := range routes {
		if !route.Active || route.TripsComplete == 0 {
			continue
		}

		currentMilestone := trhof.milestones[route.ID]

		for _, m := range milestones {
			if route.TripsComplete >= m.threshold && currentMilestone < m.threshold {
				trhof.milestones[route.ID] = m.threshold

				for _, p := range players {
					if p != nil && p.Name == route.Owner {
						p.Credits += m.bonus
						break
					}
				}

				game.LogEvent("event", route.Owner,
					fmt.Sprintf("🛣️ Route #%d is now a %s! (%d trips of %s) +%dcr",
						route.ID, m.title, route.TripsComplete, route.Resource, m.bonus))
				break
			}
		}
	}
}
