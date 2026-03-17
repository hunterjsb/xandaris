package tickable

import (
	"fmt"
	"math"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&FuelPriceIndexSystem{
		BaseSystem: NewBaseSystem("FuelPriceIndex", 113),
	})
}

// FuelPriceIndexSystem computes a "Fuel Price Index" that reflects
// the true cost of logistics across the galaxy. Fuel is the lifeblood
// of all shipping — when fuel is scarce, everything grinds to a halt.
//
// FPI components:
//   - Average fuel storage per planet (lower = higher FPI)
//   - Ships stranded with 0 fuel (more = higher FPI)
//   - Active shipping routes vs stalled routes
//   - Fuel production vs consumption rate
//
// FPI scale:
//   50-80:  Cheap fuel — great time for logistics expansion
//   80-120: Normal — sustainable operations
//   120-150: Expensive — cut non-essential routes
//   150+:   Crisis — ships stranding, routes failing
//
// Announced every ~5000 ticks. Helps factions plan logistics spending.
type FuelPriceIndexSystem struct {
	*BaseSystem
	history    []float64
	nextReport int64
}

func (fpis *FuelPriceIndexSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := fpis.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if fpis.nextReport == 0 {
		fpis.nextReport = tick + 3000
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	routes := game.GetShippingRoutes()

	// Calculate FPI components
	totalFuel := 0
	planetCount := 0
	strandedShips := 0
	totalShips := 0
	activeRoutes := 0
	stalledRoutes := 0

	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				totalFuel += planet.GetStoredAmount(entities.ResFuel)
				planetCount++
			}
		}
	}

	for _, p := range players {
		if p == nil {
			continue
		}
		for _, ship := range p.OwnedShips {
			if ship == nil {
				continue
			}
			totalShips++
			if ship.CurrentFuel == 0 && ship.Status != entities.ShipStatusMoving {
				strandedShips++
			}
		}
	}

	for _, route := range routes {
		if route.Active {
			if route.TripsComplete > 0 {
				activeRoutes++
			} else {
				stalledRoutes++
			}
		}
	}

	// Compute FPI
	fpi := 100.0 // baseline

	// Fuel scarcity: less fuel = higher FPI
	if planetCount > 0 {
		avgFuel := float64(totalFuel) / float64(planetCount)
		fuelScore := avgFuel / 100.0 // 100 fuel avg = neutral
		if fuelScore < 1.0 {
			fpi += (1.0 - fuelScore) * 50 // up to +50 for low fuel
		} else {
			fpi -= (fuelScore - 1.0) * 20 // down for surplus
		}
	}

	// Stranded ships: more stranded = higher FPI
	if totalShips > 0 {
		strandedRatio := float64(strandedShips) / float64(totalShips)
		fpi += strandedRatio * 40
	}

	// Route health: more stalled = higher FPI
	totalRoutes := activeRoutes + stalledRoutes
	if totalRoutes > 0 {
		stalledRatio := float64(stalledRoutes) / float64(totalRoutes)
		fpi += stalledRatio * 30
	}

	fpi = math.Max(30, math.Min(200, fpi))

	fpis.history = append(fpis.history, fpi)
	if len(fpis.history) > 10 {
		fpis.history = fpis.history[1:]
	}

	// Report
	if tick >= fpis.nextReport {
		fpis.nextReport = tick + 5000

		trend := "→"
		if len(fpis.history) >= 2 {
			prev := fpis.history[len(fpis.history)-2]
			if fpi > prev+5 {
				trend = "📈"
			} else if fpi < prev-5 {
				trend = "📉"
			}
		}

		status := "Normal"
		switch {
		case fpi > 150:
			status = "CRISIS"
		case fpi > 120:
			status = "Expensive"
		case fpi < 80:
			status = "Cheap"
		case fpi < 50:
			status = "Dirt Cheap"
		}

		game.LogEvent("intel", "",
			fmt.Sprintf("⛽ Fuel Price Index: %.0f %s (%s) | Avg fuel/planet: %d | Stranded: %d/%d ships | Routes: %d active, %d stalled",
				fpi, trend, status,
				totalFuel/max(planetCount, 1), strandedShips, totalShips,
				activeRoutes, stalledRoutes))
	}
}

// GetFPI returns the current Fuel Price Index.
func (fpis *FuelPriceIndexSystem) GetFPI() float64 {
	if len(fpis.history) == 0 {
		return 100
	}
	return fpis.history[len(fpis.history)-1]
}
