package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&IdleCargoDispatcherSystem{
		BaseSystem: NewBaseSystem("IdleCargoDispatcher", 28),
	})
}

// IdleCargoDispatcherSystem is a brute-force fix for idle cargo ships.
// When a faction has idle cargo ships AND unassigned shipping routes,
// this system directly assigns ships to routes and sends them to
// the source planet.
//
// The existing shipping system's auto-assign only works when ships
// are already at the source planet. This system bridges the gap by:
//   1. Finding unassigned routes
//   2. Finding idle cargo ships (any system)
//   3. Assigning the ship to the route
//   4. Starting the ship's journey toward the source system
//
// Priority 28: runs right before shipping system (29) to prepare.
type IdleCargoDispatcherSystem struct {
	*BaseSystem
}

func (icds *IdleCargoDispatcherSystem) OnTick(tick int64) {
	if tick%100 != 0 {
		return
	}

	ctx := icds.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	systemsMap := game.GetSystemsMap()
	routes := game.GetShippingRoutes()

	// Build set of already-assigned ships
	assignedShips := make(map[int]bool)
	for _, r := range routes {
		if r.Active && r.ShipID != 0 {
			assignedShips[r.ShipID] = true
		}
	}

	for _, route := range routes {
		if !route.Active || route.ShipID != 0 {
			continue
		}

		// Find source planet and its system
		sourcePlanet := findPlanetByID(systemsMap, route.SourcePlanet)
		if sourcePlanet == nil {
			continue
		}
		sourceSystem := findSystemForPlanet(sourcePlanet, systems)
		if sourceSystem < 0 {
			continue
		}

		// Find an idle cargo ship owned by this faction
		for _, p := range players {
			if p == nil || p.Name != route.Owner {
				continue
			}

			for _, ship := range p.OwnedShips {
				if ship == nil || ship.ShipType != entities.ShipTypeCargo {
					continue
				}
				if ship.Status == entities.ShipStatusMoving {
					continue
				}
				if assignedShips[ship.GetID()] {
					continue
				}
				if ship.GetTotalCargo() > 0 {
					continue // has cargo, don't reassign
				}
				if ship.CurrentFuel < ship.FuelPerJump {
					continue // can't travel
				}

				// Assign!
				game.AssignShipToRoute(route.ID, ship.GetID())
				assignedShips[ship.GetID()] = true

				// Start journey to source if not already there
				if ship.CurrentSystem != sourceSystem {
					// Try direct
					if !game.StartShipJourney(ship, sourceSystem) {
						// Hop toward source
						connected := game.GetConnectedSystems(ship.CurrentSystem)
						for _, hop := range connected {
							if game.StartShipJourney(ship, hop) {
								break
							}
						}
					}
				}

				fmt.Printf("[IdleDispatch] Assigned %s to route #%d (%s), dispatching to source SYS-%d\n",
					ship.Name, route.ID, route.Resource, sourceSystem)
				break
			}
			break
		}
	}
}
