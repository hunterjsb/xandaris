package tickable

import (
	"fmt"
)

func init() {
	RegisterSystem(&ShippingMilestoneSystem{
		BaseSystem: NewBaseSystem("ShippingMilestones", 114),
	})
}

// ShippingMilestoneSystem celebrates logistics achievements and tracks
// the overall health of the galaxy's shipping network.
//
// Ship milestones (per ship):
//   10 jumps:   "Seasoned" — ship has proven reliability
//   50 jumps:   "Veteran Hauler" — +100cr bonus
//   100 jumps:  "Road Warrior" — +500cr bonus
//
// Faction milestones:
//   First cargo delivery ever
//   10 total deliveries
//   100 total deliveries (Trade Baron title)
//   1000 total deliveries (Logistics Emperor)
//
// Galaxy milestones:
//   Total galaxy-wide deliveries: 100, 500, 1000
//   Announces shipping network statistics
type ShippingMilestoneSystem struct {
	*BaseSystem
	factionDeliveries map[string]int // faction → total known deliveries
	galaxyAnnounced   map[int]bool   // threshold → already announced
}

func (sms *ShippingMilestoneSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := sms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if sms.factionDeliveries == nil {
		sms.factionDeliveries = make(map[string]int)
		sms.galaxyAnnounced = make(map[int]bool)
	}

	players := ctx.GetPlayers()
	routes := game.GetShippingRoutes()

	// Aggregate deliveries per faction
	factionTrips := make(map[string]int)
	totalGalaxy := 0
	for _, route := range routes {
		factionTrips[route.Owner] += route.TripsComplete
		totalGalaxy += route.TripsComplete
	}

	// Check faction milestones
	milestones := []struct {
		threshold int
		title     string
		bonus     int
	}{
		{1000, "Logistics Emperor", 10000},
		{100, "Trade Baron", 2000},
		{10, "Established Trader", 500},
		{1, "First Delivery", 200},
	}

	for faction, trips := range factionTrips {
		prev := sms.factionDeliveries[faction]
		sms.factionDeliveries[faction] = trips

		for _, m := range milestones {
			if trips >= m.threshold && prev < m.threshold {
				for _, p := range players {
					if p != nil && p.Name == faction {
						p.Credits += m.bonus
						break
					}
				}
				game.LogEvent("event", faction,
					fmt.Sprintf("📦 %s: %s! (%d shipping deliveries) +%dcr",
						faction, m.title, trips, m.bonus))
				break
			}
		}
	}

	// Galaxy-wide milestones
	galaxyThresholds := []int{1000, 500, 100, 50}
	for _, threshold := range galaxyThresholds {
		if totalGalaxy >= threshold && !sms.galaxyAnnounced[threshold] {
			sms.galaxyAnnounced[threshold] = true
			game.LogEvent("event", "",
				fmt.Sprintf("🌐 Galaxy logistics milestone: %d total shipping deliveries across all factions! The trade network grows!",
					totalGalaxy))
			break
		}
	}
}
