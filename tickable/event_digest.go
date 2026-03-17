package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&EventDigestSystem{
		BaseSystem: NewBaseSystem("EventDigest", 162),
	})
}

// EventDigestSystem fires a guaranteed periodic digest event every
// 2000 ticks that summarizes what's happening. Unlike other periodic
// systems that fire every 5000-15000 ticks and get buried, this fires
// frequently enough to always be visible in the last 50 events.
//
// The digest rotates through different summaries:
//   Tick 0: Leaderboard snapshot
//   Tick 1: Shipping/logistics status
//   Tick 2: Galaxy vital signs
//   Tick 3: Random fun fact
type EventDigestSystem struct {
	*BaseSystem
	rotation int
}

func (eds *EventDigestSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := eds.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	eds.rotation = (eds.rotation + 1) % 4

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	routes := game.GetShippingRoutes()

	switch eds.rotation {
	case 0:
		eds.leaderboardDigest(players, systems, game)
	case 1:
		eds.logisticsDigest(players, routes, game)
	case 2:
		eds.vitalSigns(players, systems, game)
	case 3:
		eds.funFact(players, systems, routes, game, tick)
	}
}

func (eds *EventDigestSystem) leaderboardDigest(players []*entities.Player, systems []*entities.System, game GameProvider) {
	// Sort by credits
	type entry struct{ name string; credits int }
	var sorted []entry
	for _, p := range players {
		if p != nil {
			sorted = append(sorted, entry{p.Name, p.Credits})
		}
	}
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].credits > sorted[i].credits {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	if len(sorted) < 2 {
		return
	}

	msg := "📊 "
	for i, e := range sorted {
		if i >= 3 { break }
		medal := []string{"🥇", "🥈", "🥉"}[i]
		cr := fmt.Sprintf("%dcr", e.credits)
		if e.credits > 1000000 {
			cr = fmt.Sprintf("%.1fMcr", float64(e.credits)/1000000)
		} else if e.credits > 1000 {
			cr = fmt.Sprintf("%.0fKcr", float64(e.credits)/1000)
		}
		msg += fmt.Sprintf("%s%s(%s) ", medal, e.name, cr)
	}
	game.LogEvent("intel", "", msg)
}

func (eds *EventDigestSystem) logisticsDigest(players []*entities.Player, routes []ShippingRouteInfo, game GameProvider) {
	active := 0
	delivering := 0
	totalTrips := 0
	for _, r := range routes {
		if r.Active {
			active++
			if r.TripsComplete > 0 {
				delivering++
			}
			totalTrips += r.TripsComplete
		}
	}

	cargoShips := 0
	movingCargo := 0
	for _, p := range players {
		if p == nil { continue }
		for _, s := range p.OwnedShips {
			if s != nil && s.ShipType == entities.ShipTypeCargo {
				cargoShips++
				if s.Status == entities.ShipStatusMoving {
					movingCargo++
				}
			}
		}
	}

	game.LogEvent("intel", "",
		fmt.Sprintf("🚚 Logistics: %d routes (%d delivering, %d total trips) | %d cargo ships (%d in transit)",
			active, delivering, totalTrips, cargoShips, movingCargo))
}

func (eds *EventDigestSystem) vitalSigns(players []*entities.Player, systems []*entities.System, game GameProvider) {
	totalPop := int64(0)
	totalPlanets := 0
	happyPlanets := 0
	crisisPlanets := 0

	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				totalPlanets++
				totalPop += planet.Population
				if planet.Happiness > 0.7 {
					happyPlanets++
				}
				if planet.GetPowerRatio() < 0.3 {
					crisisPlanets++
				}
			}
		}
	}

	game.LogEvent("intel", "",
		fmt.Sprintf("🌍 Vitals: %d planets, %d pop | %d happy, %d in crisis",
			totalPlanets, totalPop, happyPlanets, crisisPlanets))
}

func (eds *EventDigestSystem) funFact(players []*entities.Player, systems []*entities.System, routes []ShippingRouteInfo, game GameProvider, tick int64) {
	facts := []string{
		fmt.Sprintf("The galaxy has been running for %d hours and %d minutes", tick/36000, (tick%36000)/600),
	}

	totalShips := 0
	for _, p := range players {
		if p != nil {
			totalShips += len(p.OwnedShips)
		}
	}
	facts = append(facts, fmt.Sprintf("%d ships sail the void between %d star systems", totalShips, len(systems)))

	totalTrips := 0
	for _, r := range routes {
		totalTrips += r.TripsComplete
	}
	if totalTrips > 0 {
		facts = append(facts, fmt.Sprintf("%d cargo deliveries have been completed across the galaxy", totalTrips))
	}

	game.LogEvent("intel", "", "💫 "+facts[rand.Intn(len(facts))])
}
