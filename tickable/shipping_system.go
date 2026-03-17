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

	// Build set of ships already assigned to routes
	assignedShips := make(map[int]bool)
	routes := gp.GetShippingRoutes()
	for _, route := range routes {
		if route.Active && route.ShipID != 0 {
			assignedShips[route.ShipID] = true
		}
	}

	for _, route := range routes {
		if !route.Active {
			continue
		}

		// Auto-assign: find an idle cargo ship for unassigned routes
		if route.ShipID == 0 {
			sourcePlanet := findPlanetByID(systemsMap, route.SourcePlanet)
			if sourcePlanet == nil {
				continue
			}
			sourceSystem := findSystemForPlanet(sourcePlanet, systems)
			// Find an idle cargo ship owned by the route owner in the source system
			for _, p := range players {
				if p == nil || p.Name != route.Owner {
					continue
				}
				for _, ship := range p.OwnedShips {
					if ship == nil || ship.ShipType != entities.ShipTypeCargo {
						continue
					}
					if ship.Status == entities.ShipStatusMoving || ship.DeliveryID != 0 {
						continue
					}
					if assignedShips[ship.GetID()] {
						continue
					}
					// Prefer ships in the source system
					if ship.CurrentSystem == sourceSystem {
						route.ShipID = ship.GetID()
						assignedShips[ship.GetID()] = true
						gp.AssignShipToRoute(route.ID, ship.GetID())
						fmt.Printf("[Shipping] Auto-assigned %s to route #%d (%s)\n",
							ship.Name, route.ID, route.Resource)
						break
					}
				}
				// Fallback: any idle cargo ship
				if route.ShipID == 0 {
					for _, ship := range p.OwnedShips {
						if ship == nil || ship.ShipType != entities.ShipTypeCargo {
							continue
						}
						if ship.Status == entities.ShipStatusMoving || ship.DeliveryID != 0 {
							continue
						}
						if assignedShips[ship.GetID()] {
							continue
						}
						route.ShipID = ship.GetID()
						assignedShips[ship.GetID()] = true
						gp.AssignShipToRoute(route.ID, ship.GetID())
						fmt.Printf("[Shipping] Auto-assigned %s to route #%d (%s, different system)\n",
							ship.Name, route.ID, route.Resource)
						break
					}
				}
				break
			}
			if route.ShipID == 0 {
				continue // no ship available
			}
		}

		ship := findShipByID(players, route.ShipID)
		if ship == nil || ship.Status == entities.ShipStatusMoving {
			continue
		}
		if ship.DeliveryID != 0 {
			continue
		}
		// Only cargo ships can run trade routes
		if ship.ShipType != entities.ShipTypeCargo {
			continue
		}

		ss.processRoute(route, ship, gp, players, systems, systemsMap)
	}
}

func (ss *ShippingSystem) processRoute(route ShippingRouteInfo, ship *entities.Ship, gp GameProvider, players []*entities.Player, systems []*entities.System, systemsMap map[int]*entities.System) {
	sourcePlanet := findPlanetByID(systemsMap, route.SourcePlanet)
	destPlanet := findPlanetByID(systemsMap, route.DestPlanet)
	if sourcePlanet == nil || destPlanet == nil {
		fmt.Printf("[Shipping] Route #%d: invalid planet IDs (src=%d dst=%d) — deactivating\n",
			route.ID, route.SourcePlanet, route.DestPlanet)
		gp.CancelShippingRoute(route.ID)
		return
	}

	sourceSystemID := findSystemForPlanet(sourcePlanet, systems)
	destSystemID := findSystemForPlanet(destPlanet, systems)

	atSource := ship.CurrentSystem == sourceSystemID
	atDest := ship.CurrentSystem == destSystemID

	// Auto-refuel: if ship is low on fuel, try to refuel from any planet in current system
	if ship.CurrentFuel < ship.FuelPerJump*2 {
		// Try route endpoints first, then any owned planet in current system
		var refuelPlanet *entities.Planet
		if atSource && sourcePlanet.GetStoredAmount("Fuel") > 0 {
			refuelPlanet = sourcePlanet
		} else if atDest && destPlanet.GetStoredAmount("Fuel") > 0 {
			refuelPlanet = destPlanet
		} else {
			// Stranded — find any owned planet in this system with Fuel
			for _, sys := range systems {
				if sys.ID != ship.CurrentSystem {
					continue
				}
				for _, e := range sys.Entities {
					if p, ok := e.(*entities.Planet); ok && p.Owner == ship.Owner && p.GetStoredAmount("Fuel") > 0 {
						refuelPlanet = p
						break
					}
				}
				break
			}
		}
		if refuelPlanet != nil {
			needed := ship.MaxFuel - ship.CurrentFuel
			available := refuelPlanet.GetStoredAmount("Fuel")
			refuel := needed
			if refuel > available {
				refuel = available
			}
			if refuel > 0 {
				refuelPlanet.RemoveStoredResource("Fuel", refuel)
				ship.CurrentFuel += refuel
				fmt.Printf("[Shipping] Route #%d: %s refueled %d from %s (fuel: %d/%d)\n",
					route.ID, ship.Name, refuel, refuelPlanet.Name, ship.CurrentFuel, ship.MaxFuel)
			}
		}
	}

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
			fmt.Printf("[Shipping] Route #%d: LOAD FAILED — %s has %d %s, ship %s in sys %d, planet in sys %d: %v\n",
				route.ID, sourcePlanet.Name, sourcePlanet.GetStoredAmount(route.Resource),
				route.Resource, ship.Name, ship.CurrentSystem, sourceSystemID, err)
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
			fmt.Printf("[Shipping] Route #%d: TRAVEL FAILED — %s fuel=%d/%d, fuelPerJump=%d, from sys %d → sys %d\n",
				route.ID, ship.Name, ship.CurrentFuel, ship.MaxFuel, ship.FuelPerJump, sourceSystemID, destSystemID)
			// Can't depart — return cargo to source planet
			gp.UnloadCargo(ship, sourcePlanet, route.Resource, loaded)
		}
	} else if atDest && ship.GetTotalCargo() == 0 {
		// At destination empty — return to source for next load
		if sourceSystemID != destSystemID {
			gp.StartShipJourney(ship, sourceSystemID)
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
