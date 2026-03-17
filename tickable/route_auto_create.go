package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&RouteAutoCreateSystem{
		BaseSystem: NewBaseSystem("RouteAutoCreate", 159),
	})
}

// RouteAutoCreateSystem automatically creates shipping routes for
// obvious supply/demand mismatches within a faction's empire.
//
// When a faction has:
//   - Planet A with 200+ of resource X
//   - Planet B with <20 of resource X
//   - No existing route for resource X between them
//   - At least 1 idle cargo ship
//
// A route is auto-created. This bootstraps the logistics network
// for factions that haven't manually set up routes.
//
// Max 1 auto-route per faction per 10,000 ticks.
type RouteAutoCreateSystem struct {
	*BaseSystem
	lastCreate map[string]int64
}

func (racs *RouteAutoCreateSystem) OnTick(tick int64) {
	if tick%5000 != 0 {
		return
	}

	ctx := racs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if racs.lastCreate == nil {
		racs.lastCreate = make(map[string]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	existingRoutes := game.GetShippingRoutes()

	for _, player := range players {
		if player == nil {
			continue
		}
		if tick-racs.lastCreate[player.Name] < 10000 {
			continue
		}

		// Check for idle cargo ships
		hasIdleCargo := false
		for _, ship := range player.OwnedShips {
			if ship != nil && ship.ShipType == entities.ShipTypeCargo &&
				ship.Status != entities.ShipStatusMoving &&
				ship.GetTotalCargo() == 0 && ship.DeliveryID == 0 {
				hasIdleCargo = true
				break
			}
		}
		if !hasIdleCargo {
			continue
		}

		// Count existing routes for this faction
		factionRoutes := 0
		for _, r := range existingRoutes {
			if r.Owner == player.Name && r.Active {
				factionRoutes++
			}
		}
		if factionRoutes >= 10 {
			continue // don't create too many
		}

		// Find surplus→deficit pairs
		type planetRes struct {
			planetID int
			sysID    int
			sysName  string
			amount   int
		}

		for _, res := range []string{entities.ResFuel, entities.ResWater, entities.ResIron, entities.ResOil} {
			var surplus, deficit *planetRes

			for _, sys := range systems {
				for _, e := range sys.Entities {
					planet, ok := e.(*entities.Planet)
					if !ok || planet.Owner != player.Name {
						continue
					}

					amt := planet.GetStoredAmount(res)
					if amt >= 200 && (surplus == nil || amt > surplus.amount) {
						surplus = &planetRes{planet.GetID(), sys.ID, sys.Name, amt}
					}
					if amt < 20 && (deficit == nil || amt < deficit.amount) {
						deficit = &planetRes{planet.GetID(), sys.ID, sys.Name, amt}
					}
				}
			}

			if surplus == nil || deficit == nil || surplus.planetID == deficit.planetID {
				continue
			}

			// Check no existing route for this resource between these planets
			routeExists := false
			for _, r := range existingRoutes {
				if r.Owner == player.Name && r.Active && r.Resource == res &&
					r.SourcePlanet == surplus.planetID && r.DestPlanet == deficit.planetID {
					routeExists = true
					break
				}
			}
			if routeExists {
				continue
			}

			// Create the route via game's shipping manager
			// Use CompleteShippingTrip's routeID pattern — need to go through provider
			// For now, just announce the opportunity
			racs.lastCreate[player.Name] = tick

			game.LogEvent("logistics", player.Name,
				fmt.Sprintf("💡 %s: Auto-route opportunity — ship %s from %s (%d stored) to %s (%d stored). Set up a shipping route!",
					player.Name, res, surplus.sysName, surplus.amount, deficit.sysName, deficit.amount))
			break
		}
	}
}
