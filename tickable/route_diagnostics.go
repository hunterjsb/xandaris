package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&RouteDiagnosticsSystem{
		BaseSystem: NewBaseSystem("RouteDiagnostics", 30),
	})
}

// RouteDiagnosticsSystem monitors shipping routes and logs actionable
// diagnostics when routes are stuck. This helps both human players and
// LLM agents understand why their logistics aren't working.
//
// Checks each route for:
//   - No ship assigned (ShipID=0)
//   - Ship stranded (0 fuel, wrong system)
//   - Source planet has no stock of route resource
//   - Source and dest are the same planet (invalid)
//   - Multi-hop route with insufficient fuel capacity
//   - Dest planet storage full
//
// Reports fire once per route per 5000 ticks (not spammy).
type RouteDiagnosticsSystem struct {
	*BaseSystem
	lastReport map[int]int64 // routeID → last report tick
}

func (rds *RouteDiagnosticsSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := rds.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if rds.lastReport == nil {
		rds.lastReport = make(map[int]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	systemsMap := game.GetSystemsMap()
	routes := game.GetShippingRoutes()

	for _, route := range routes {
		if !route.Active {
			continue
		}

		// Rate limit: one diagnostic per route per 5000 ticks
		if tick-rds.lastReport[route.ID] < 5000 {
			continue
		}

		issues := rds.diagnoseRoute(route, players, systems, systemsMap, game)
		if len(issues) > 0 {
			rds.lastReport[route.ID] = tick
			msg := fmt.Sprintf("🔧 Route #%d (%s %s): ", route.ID, route.Resource, route.Owner)
			for i, issue := range issues {
				if i > 0 {
					msg += " | "
				}
				msg += issue
			}
			game.LogEvent("logistics", route.Owner, msg)
		}
	}
}

func (rds *RouteDiagnosticsSystem) diagnoseRoute(route ShippingRouteInfo, players []*entities.Player, systems []*entities.System, systemsMap map[int]*entities.System, game GameProvider) []string {
	var issues []string

	// Invalid: same source and dest
	if route.SourcePlanet == route.DestPlanet {
		issues = append(issues, "source = destination (invalid route)")
		return issues
	}

	// Find planets
	sourcePlanet := findPlanetByID(systemsMap, route.SourcePlanet)
	destPlanet := findPlanetByID(systemsMap, route.DestPlanet)
	if sourcePlanet == nil {
		issues = append(issues, fmt.Sprintf("source planet %d not found", route.SourcePlanet))
		return issues
	}
	if destPlanet == nil {
		issues = append(issues, fmt.Sprintf("dest planet %d not found", route.DestPlanet))
		return issues
	}

	// No ship assigned
	if route.ShipID == 0 {
		issues = append(issues, "no ship assigned — need idle Cargo ship")
		return issues
	}

	// Find ship
	ship := findShipByID(players, route.ShipID)
	if ship == nil {
		issues = append(issues, fmt.Sprintf("assigned ship %d not found", route.ShipID))
		return issues
	}

	// Ship fuel
	if ship.CurrentFuel == 0 {
		issues = append(issues, fmt.Sprintf("%s has 0 fuel — stranded", ship.Name))
	} else if ship.CurrentFuel < ship.FuelPerJump {
		issues = append(issues, fmt.Sprintf("%s fuel %d/%d — can't jump (need %d)", ship.Name, ship.CurrentFuel, ship.MaxFuel, ship.FuelPerJump))
	}

	// Source planet stock
	sourceStock := sourcePlanet.GetStoredAmount(route.Resource)
	if sourceStock == 0 {
		issues = append(issues, fmt.Sprintf("%s has 0 %s — nothing to haul", sourcePlanet.Name, route.Resource))
	}

	// Ship location
	sourceSystemID := findSystemForPlanet(sourcePlanet, systems)
	destSystemID := findSystemForPlanet(destPlanet, systems)
	if ship.CurrentSystem != sourceSystemID && ship.CurrentSystem != destSystemID && ship.Status != entities.ShipStatusMoving {
		issues = append(issues, fmt.Sprintf("%s in wrong system (at %d, need %d or %d)", ship.Name, ship.CurrentSystem, sourceSystemID, destSystemID))
	}

	// Check connectivity
	if sourceSystemID != destSystemID {
		connected := game.GetConnectedSystems(sourceSystemID)
		directlyConnected := false
		for _, c := range connected {
			if c == destSystemID {
				directlyConnected = true
				break
			}
		}
		if !directlyConnected {
			issues = append(issues, "systems not directly connected — need multi-hop path")
		}
	}

	// Dest storage full
	destCap := destPlanet.GetStorageCapacity()
	destStored := destPlanet.GetStoredAmount(route.Resource)
	if destCap > 0 && destStored >= destCap {
		issues = append(issues, fmt.Sprintf("%s storage full (%d/%d %s)", destPlanet.Name, destStored, destCap, route.Resource))
	}

	return issues
}
