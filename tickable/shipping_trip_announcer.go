package tickable

import (
	"fmt"
)

func init() {
	RegisterSystem(&ShippingTripAnnouncerSystem{
		BaseSystem: NewBaseSystem("ShippingTripAnnouncer", 202),
	})
}

// ShippingTripAnnouncerSystem watches for new shipping trip completions
// and announces them individually. The route bonus system gives credits,
// but this system generates the narrative event for each delivery.
//
// Tracks last known trip count per route. When it increases:
//   "📦 Llama's Converted-563 delivered Iron to Planet X! (trip #3)"
//
// Makes each delivery feel like an achievement, not just a number.
type ShippingTripAnnouncerSystem struct {
	*BaseSystem
	lastTrips map[int]int // routeID → last known trip count
}

func (stas *ShippingTripAnnouncerSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := stas.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if stas.lastTrips == nil {
		stas.lastTrips = make(map[int]int)
	}

	routes := game.GetShippingRoutes()

	for _, route := range routes {
		if !route.Active {
			continue
		}

		last := stas.lastTrips[route.ID]
		if route.TripsComplete > last {
			newTrips := route.TripsComplete - last
			stas.lastTrips[route.ID] = route.TripsComplete

			// Skip initial load (don't announce trip #0→existing)
			if last == 0 && route.TripsComplete > 5 {
				continue // was already running, don't spam
			}

			game.LogEvent("logistics", route.Owner,
				fmt.Sprintf("📦 %s: Route #%d delivered %s (trip #%d)",
					route.Owner, route.ID, route.Resource, route.TripsComplete))

			// Milestone announcements
			milestones := []int{10, 25, 50, 100, 200, 500}
			for _, m := range milestones {
				if last < m && route.TripsComplete >= m {
					game.LogEvent("event", route.Owner,
						fmt.Sprintf("🎉 Route #%d hit %d trips! %s's %s route is legendary!",
							route.ID, m, route.Owner, route.Resource))
				}
			}

			_ = newTrips
		}
	}
}
