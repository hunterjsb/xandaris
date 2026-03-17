package tickable

import (
	"fmt"
)

func init() {
	RegisterSystem(&RouteHealthMonitorSystem{
		BaseSystem: NewBaseSystem("RouteHealthMonitor", 163),
	})
}

// RouteHealthMonitorSystem provides a single clear status for each
// faction's shipping network health. Instead of individual route
// diagnostics (which are noisy), this gives ONE summary.
//
// Health = (routes with trips > 0) / (total active routes) * 100
//
// Status:
//   Excellent (80%+): most routes delivering
//   Good (50-79%): solid but room to improve
//   Poor (20-49%): many stuck routes, needs attention
//   Critical (<20%): logistics network barely functional
//
// Fires every 5000 ticks. One line per faction with actionable info.
type RouteHealthMonitorSystem struct {
	*BaseSystem
	nextReport int64
}

func (rhms *RouteHealthMonitorSystem) OnTick(tick int64) {
	if tick%5000 != 0 {
		return
	}

	ctx := rhms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if rhms.nextReport == 0 {
		rhms.nextReport = tick + 3000
	}
	if tick < rhms.nextReport {
		return
	}
	rhms.nextReport = tick + 5000

	players := ctx.GetPlayers()
	routes := game.GetShippingRoutes()

	// Aggregate per faction
	type routeHealth struct {
		active    int
		delivering int
		totalTrips int
		stuckShips int
	}
	factions := make(map[string]*routeHealth)

	for _, r := range routes {
		if !r.Active {
			continue
		}
		if factions[r.Owner] == nil {
			factions[r.Owner] = &routeHealth{}
		}
		factions[r.Owner].active++
		if r.TripsComplete > 0 {
			factions[r.Owner].delivering++
		}
		factions[r.Owner].totalTrips += r.TripsComplete

		// Check if assigned ship is stuck (0 fuel)
		if r.ShipID != 0 {
			for _, p := range players {
				if p == nil || p.Name != r.Owner {
					continue
				}
				for _, ship := range p.OwnedShips {
					if ship != nil && ship.GetID() == r.ShipID && ship.CurrentFuel == 0 {
						factions[r.Owner].stuckShips++
					}
				}
				break
			}
		}
	}

	for name, health := range factions {
		if health.active == 0 {
			continue
		}

		pct := float64(health.delivering) / float64(health.active) * 100
		status := "Critical"
		emoji := "🔴"
		switch {
		case pct >= 80:
			status = "Excellent"
			emoji = "🟢"
		case pct >= 50:
			status = "Good"
			emoji = "🟡"
		case pct >= 20:
			status = "Poor"
			emoji = "🟠"
		}

		msg := fmt.Sprintf("%s %s logistics: %s (%.0f%%) — %d/%d routes delivering, %d total trips",
			emoji, name, status, pct, health.delivering, health.active, health.totalTrips)
		if health.stuckShips > 0 {
			msg += fmt.Sprintf(", %d ships stuck (0 fuel)", health.stuckShips)
		}

		game.LogEvent("logistics", name, msg)
	}
}
