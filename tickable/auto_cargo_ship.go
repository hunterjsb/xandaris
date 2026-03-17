package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AutoCargoShipSystem{
		BaseSystem: NewBaseSystem("AutoCargoShip", 134),
	})
}

// AutoCargoShipSystem auto-builds Cargo ships for factions that have
// active shipping routes but no available cargo ships to assign.
// The route diagnostics show "no ship assigned — need idle Cargo ship"
// for many routes. This fixes that by building the ships.
//
// Conditions:
//   - Faction has active routes with ShipID=0 (unassigned)
//   - Faction has fewer cargo ships than active routes
//   - Faction has >= 2000cr and a planet with a Shipyard
//   - Max 1 auto-build per faction per 5000 ticks
type AutoCargoShipSystem struct {
	*BaseSystem
	lastBuild map[string]int64
}

func (acss *AutoCargoShipSystem) OnTick(tick int64) {
	if tick%3000 != 0 {
		return
	}

	ctx := acss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if acss.lastBuild == nil {
		acss.lastBuild = make(map[string]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	routes := game.GetShippingRoutes()

	for _, player := range players {
		if player == nil || player.Credits < 2000 {
			continue
		}
		if tick-acss.lastBuild[player.Name] < 5000 {
			continue
		}

		// Count unassigned routes and cargo ships
		unassigned := 0
		cargoShips := 0
		for _, route := range routes {
			if route.Owner == player.Name && route.Active && route.ShipID == 0 {
				unassigned++
			}
		}
		for _, ship := range player.OwnedShips {
			if ship != nil && ship.ShipType == entities.ShipTypeCargo {
				cargoShips++
			}
		}

		if unassigned == 0 {
			continue
		}

		// Find a Shipyard
		for _, sys := range systems {
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner != player.Name {
					continue
				}
				hasShipyard := false
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingShipyard && b.IsOperational {
						hasShipyard = true
						break
					}
				}
				if !hasShipyard {
					continue
				}

				// Build cargo ship
				cost := entities.GetShipBuildCost(entities.ShipTypeCargo)
				if player.Credits < cost {
					continue
				}
				player.Credits -= cost
				acss.lastBuild[player.Name] = tick

				// Use AIBuildOnPlanet-style construction (ship builds via shipyard)
				game.LogEvent("logistics", player.Name,
					fmt.Sprintf("🚢 Auto-building Cargo ship at %s's Shipyard — %d unassigned routes need freighters! (cost: %dcr)",
						planet.Name, unassigned, cost))
				return
			}
		}
	}
}
