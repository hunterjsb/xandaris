package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ShippingSystem{
		BaseSystem: NewBaseSystem("Shipping", 29),
	})
}

// ShippingSystem processes active ShippingRoutes on tick.
// Each route auto-cycles: load → travel → unload → return → repeat.
type ShippingSystem struct {
	*BaseSystem
}

func (ss *ShippingSystem) OnTick(tick int64) {
	if tick%20 != 0 {
		return
	}

	ctx := ss.GetContext()
	if ctx == nil {
		return
	}

	gp := ctx.GetGame()
	if gp == nil {
		return
	}

	players := gp.GetPlayers()
	systems := gp.GetSystems()
	systemsMap := gp.GetSystemsMap()

	routes := gp.GetShippingRoutes()
	for _, route := range routes {
		if !route.Active || route.ShipID == 0 {
			continue
		}

		ship := findShipByID(players, route.ShipID)
		if ship == nil || ship.Status == entities.ShipStatusMoving {
			continue
		}
		if ship.DeliveryID != 0 {
			continue
		}

		ss.processRoute(route, ship, gp, players, systems, systemsMap)
	}
}

func (ss *ShippingSystem) processRoute(route ShippingRouteInfo, ship *entities.Ship, gp GameProvider, players []*entities.Player, systems []*entities.System, systemsMap map[int]*entities.System) {
	sourcePlanet := findPlanetByID(systemsMap, route.SourcePlanet)
	destPlanet := findPlanetByID(systemsMap, route.DestPlanet)
	if sourcePlanet == nil || destPlanet == nil {
		return
	}

	sourceSystemID := findSystemForPlanet(sourcePlanet, systems)
	destSystemID := findSystemForPlanet(destPlanet, systems)

	atSource := ship.CurrentSystem == sourceSystemID
	atDest := ship.CurrentSystem == destSystemID

	if atSource && ship.GetTotalCargo() == 0 {
		// At source with empty hold — load cargo
		qty := route.Quantity
		if qty <= 0 {
			qty = ship.MaxCargo - ship.GetTotalCargo()
		}
		if qty <= 0 {
			return
		}
		loaded, err := gp.LoadCargo(ship, sourcePlanet, route.Resource, qty)
		if err != nil || loaded <= 0 {
			return
		}
		fmt.Printf("[Shipping] Route #%d: %s loaded %d %s from %s\n",
			route.ID, ship.Name, loaded, route.Resource, sourcePlanet.Name)

		// Same-system: unload immediately
		if sourceSystemID == destSystemID {
			unloaded, err := gp.UnloadCargo(ship, destPlanet, route.Resource, loaded)
			if err == nil && unloaded > 0 {
				gp.CompleteShippingTrip(route.ID)
				fmt.Printf("[Shipping] Route #%d: %s delivered %d %s to %s (same system)\n",
					route.ID, ship.Name, unloaded, route.Resource, destPlanet.Name)
			}
			return
		}

		// Travel to destination — if fuel insufficient, unload cargo back
		if gp.StartShipJourney(ship, destSystemID) {
			fmt.Printf("[Shipping] Route #%d: %s heading to SYS-%d\n",
				route.ID, ship.Name, destSystemID)
		} else {
			// Can't depart — return cargo to source planet
			gp.UnloadCargo(ship, sourcePlanet, route.Resource, loaded)
		}
	} else if atDest && ship.GetTotalCargo() > 0 {
		// At destination with cargo — unload
		unloaded, err := gp.UnloadCargo(ship, destPlanet, route.Resource, ship.CargoHold[route.Resource])
		if err == nil && unloaded > 0 {
			gp.CompleteShippingTrip(route.ID)
			fmt.Printf("[Shipping] Route #%d: %s delivered %d %s to %s\n",
				route.ID, ship.Name, unloaded, route.Resource, destPlanet.Name)
		}
		// Unload any extra cargo
		for res, amt := range ship.CargoHold {
			if res != route.Resource && amt > 0 {
				gp.UnloadCargo(ship, destPlanet, res, amt)
			}
		}
		// Return to source
		if sourceSystemID != destSystemID {
			gp.StartShipJourney(ship, sourceSystemID)
		}
	} else if !atSource && !atDest {
		// Somewhere else — route to the right place
		if ship.GetTotalCargo() > 0 {
			gp.StartShipJourney(ship, destSystemID)
		} else {
			gp.StartShipJourney(ship, sourceSystemID)
		}
	}
}

func findSystemForPlanet(planet *entities.Planet, systems []*entities.System) int {
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if p, ok := e.(*entities.Planet); ok && p.GetID() == planet.GetID() {
				return sys.ID
			}
		}
	}
	return -1
}
