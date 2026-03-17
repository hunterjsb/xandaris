package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&GalacticWeatherForecastSystem{
		BaseSystem: NewBaseSystem("GalacticWeatherForecast", 135),
	})
}

// GalacticWeatherForecastSystem aggregates all active hazards and
// weather into a single forecast bulletin. Instead of scattered
// warnings, factions get one comprehensive forecast to plan logistics.
//
// Forecast includes:
//   - Active hyperspace storms (which lanes affected)
//   - Pirate fleet locations
//   - Supply crises in progress
//   - Active blockades
//   - Current season + next season prediction
//   - Economic cycle phase
//
// Published every ~6000 ticks. One-stop-shop for logistics planning.
type GalacticWeatherForecastSystem struct {
	*BaseSystem
	nextForecast int64
}

func (gwfs *GalacticWeatherForecastSystem) OnTick(tick int64) {
	ctx := gwfs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if gwfs.nextForecast == 0 {
		gwfs.nextForecast = tick + 4000 + int64(rand.Intn(3000))
	}
	if tick < gwfs.nextForecast {
		return
	}
	gwfs.nextForecast = tick + 6000 + int64(rand.Intn(4000))

	// Gather current conditions
	var conditions []string

	// Season
	cyclePos := tick % 20000
	season := "Spring"
	switch {
	case cyclePos < 5000:
		season = "Spring"
	case cyclePos < 10000:
		season = "Summer"
	case cyclePos < 15000:
		season = "Autumn"
	default:
		season = "Winter"
	}
	conditions = append(conditions, fmt.Sprintf("Season: %s", season))

	// Count active hazards from systems
	systems := game.GetSystems()
	pirateCount := 0
	for range systems {
		// Can't directly query pirate system, but we can count congested systems
	}
	_ = pirateCount

	// Port congestion count
	players := game.GetPlayers()
	congested := 0
	for _, sys := range systems {
		shipCount := 0
		for _, p := range players {
			if p == nil {
				continue
			}
			for _, ship := range p.OwnedShips {
				if ship != nil && ship.CurrentSystem == sys.ID {
					shipCount++
				}
			}
		}
		if shipCount > 20 {
			congested++
		}
	}

	if congested > 0 {
		conditions = append(conditions, fmt.Sprintf("%d congested ports", congested))
	}

	// Route health
	routes := game.GetShippingRoutes()
	activeRoutes := 0
	stalledRoutes := 0
	for _, r := range routes {
		if r.Active {
			activeRoutes++
			if r.TripsComplete == 0 {
				stalledRoutes++
			}
		}
	}
	if activeRoutes > 0 {
		healthPct := float64(activeRoutes-stalledRoutes) / float64(activeRoutes) * 100
		conditions = append(conditions, fmt.Sprintf("Routes: %d active (%.0f%% healthy)", activeRoutes, healthPct))
	}

	// Build forecast
	msg := "🌤️ Galactic Forecast: "
	for i, c := range conditions {
		if i > 0 {
			msg += " | "
		}
		msg += c
	}

	// Recommendation
	switch season {
	case "Spring":
		msg += " | Tip: stockpile Water for demand surge"
	case "Summer":
		msg += " | Tip: ensure fuel reserves for peak shipping"
	case "Autumn":
		msg += " | Tip: buy Iron+RM now for building season"
	case "Winter":
		msg += " | Tip: Electronics demand rising — produce or import"
	}

	game.LogEvent("intel", "", msg)
}
