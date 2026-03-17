package tickable

import (
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AutoRefuelPrioritySystem{
		BaseSystem: NewBaseSystem("AutoRefuelPriority", 21),
	})
}

// AutoRefuelPrioritySystem ensures cargo ships on active shipping
// routes get refueled before idle ships. The standard refueling
// system treats all ships equally, but route ships need fuel urgently
// to keep deliveries flowing.
//
// When fuel is scarce on a planet:
//   1. Route-assigned cargo ships refuel first (up to 50/tick)
//   2. Military ships refuel second (patrol duty)
//   3. Idle ships get whatever's left
//
// This prevents the scenario where 50 idle Colony ships drain all
// fuel from a planet while 1 cargo ship on an active route sits empty.
type AutoRefuelPrioritySystem struct {
	*BaseSystem
}

func (arps *AutoRefuelPrioritySystem) OnTick(tick int64) {
	if tick%10 != 0 {
		return
	}

	ctx := arps.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	routes := game.GetShippingRoutes()

	// Build set of ships assigned to active routes
	routeShips := make(map[int]bool)
	for _, route := range routes {
		if route.Active && route.ShipID != 0 {
			routeShips[route.ShipID] = true
		}
	}

	// Priority refuel route ships
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.Status == entities.ShipStatusMoving {
				continue
			}
			if !routeShips[ship.GetID()] {
				continue // not a route ship
			}
			if ship.CurrentFuel >= ship.FuelPerJump*2 {
				continue // has enough fuel for next jump
			}

			// Find owned planet in this system with fuel
			for _, sys := range systems {
				if sys.ID != ship.CurrentSystem {
					continue
				}
				for _, e := range sys.Entities {
					planet, ok := e.(*entities.Planet)
					if !ok || planet.Owner != ship.Owner {
						continue
					}
					available := planet.GetStoredAmount(entities.ResFuel)
					if available <= 0 {
						continue
					}
					needed := ship.MaxFuel - ship.CurrentFuel
					refuel := needed
					if refuel > 50 {
						refuel = 50 // fast refuel for route ships
					}
					if refuel > available {
						refuel = available
					}
					if refuel > 0 {
						planet.RemoveStoredResource(entities.ResFuel, refuel)
						ship.CurrentFuel += refuel
					}
					break
				}
				break
			}
		}
	}
}
