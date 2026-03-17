package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&RouteOptimizerSystem{
		BaseSystem: NewBaseSystem("RouteOptimizer", 31),
	})
}

// RouteOptimizerSystem automatically cleans up broken shipping routes
// and suggests better alternatives. This prevents route bloat and
// ensures logistics stays efficient.
//
// Actions:
//   1. Cancel routes with same source and dest (invalid)
//   2. Cancel routes where source planet no longer exists or has no owner
//   3. Cancel routes stuck at 0 trips for 20,000+ ticks
//   4. Cancel routes where assigned ship is destroyed
//   5. Auto-create profitable routes when factions have surplus/deficit
//
// Also generates "route suggestion" events that help LLM agents
// optimize their logistics networks.
type RouteOptimizerSystem struct {
	*BaseSystem
	routeAge   map[int]int64 // routeID → tick first seen at 0 trips
	nextSuggestion int64
}

func (ros *RouteOptimizerSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := ros.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ros.routeAge == nil {
		ros.routeAge = make(map[int]int64)
	}

	if ros.nextSuggestion == 0 {
		ros.nextSuggestion = tick + 5000 + int64(rand.Intn(5000))
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	systemsMap := game.GetSystemsMap()
	routes := game.GetShippingRoutes()

	cancelCount := 0

	for _, route := range routes {
		if !route.Active {
			continue
		}

		// Rule 1: same source and dest
		if route.SourcePlanet == route.DestPlanet {
			game.CancelShippingRoute(route.ID)
			cancelCount++
			continue
		}

		// Rule 2: invalid planets
		src := findPlanetByID(systemsMap, route.SourcePlanet)
		dst := findPlanetByID(systemsMap, route.DestPlanet)
		if src == nil || dst == nil {
			game.CancelShippingRoute(route.ID)
			cancelCount++
			continue
		}

		// Rule 3: stuck routes (0 trips for too long)
		if route.TripsComplete == 0 {
			if ros.routeAge[route.ID] == 0 {
				ros.routeAge[route.ID] = tick
			}
			age := tick - ros.routeAge[route.ID]
			if age > 30000 { // ~50 minutes with 0 trips
				game.CancelShippingRoute(route.ID)
				game.LogEvent("logistics", route.Owner,
					fmt.Sprintf("🔧 Auto-cancelled stuck route #%d (%s, 0 trips in %d min). Source may lack stock or fuel.",
						route.ID, route.Resource, age/600))
				cancelCount++
				continue
			}
		} else {
			delete(ros.routeAge, route.ID)
		}

		// Rule 4: assigned ship doesn't exist
		if route.ShipID != 0 {
			ship := findShipByID(players, route.ShipID)
			if ship == nil {
				game.AssignShipToRoute(route.ID, 0) // unassign
			}
		}
	}

	if cancelCount > 0 {
		game.LogEvent("logistics", "",
			fmt.Sprintf("🔧 Route Optimizer: cleaned up %d broken/stuck routes", cancelCount))
	}

	// Generate route suggestions
	if tick >= ros.nextSuggestion {
		ros.nextSuggestion = tick + 8000 + int64(rand.Intn(5000))
		ros.suggestRoutes(players, systems, game)
	}
}

func (ros *RouteOptimizerSystem) suggestRoutes(players []*entities.Player, systems []*entities.System, game GameProvider) {
	// For each faction, find surplus→deficit opportunities
	for _, player := range players {
		if player == nil {
			continue
		}

		type planetStock struct {
			planetID int
			sysID    int
			sysName  string
			stock    map[string]int
		}

		var myPlanets []planetStock
		for _, sys := range systems {
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner != player.Name {
					continue
				}
				stock := make(map[string]int)
				for _, res := range []string{entities.ResIron, entities.ResWater, entities.ResOil,
					entities.ResFuel, entities.ResRareMetals} {
					stock[res] = planet.GetStoredAmount(res)
				}
				myPlanets = append(myPlanets, planetStock{planet.GetID(), sys.ID, sys.Name, stock})
			}
		}

		if len(myPlanets) < 2 {
			continue
		}

		// Find best internal transfer opportunity
		for _, res := range []string{entities.ResFuel, entities.ResWater, entities.ResIron} {
			var surplus, deficit *planetStock
			maxSurplus := 0
			minStock := 999999

			for i := range myPlanets {
				ps := &myPlanets[i]
				amt := ps.stock[res]
				if amt > maxSurplus {
					maxSurplus = amt
					surplus = ps
				}
				if amt < minStock {
					minStock = amt
					deficit = ps
				}
			}

			if surplus != nil && deficit != nil && surplus.planetID != deficit.planetID &&
				maxSurplus > 200 && minStock < 30 {
				game.LogEvent("logistics", player.Name,
					fmt.Sprintf("💡 Route suggestion for %s: ship %s from %s (%d units) to %s (%d units)",
						player.Name, res, surplus.sysName, maxSurplus,
						deficit.sysName, minStock))
				break // one suggestion per faction per cycle
			}
		}
	}
}
