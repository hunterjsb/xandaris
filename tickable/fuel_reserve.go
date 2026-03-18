package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&FuelReserveSystem{
		BaseSystem: NewBaseSystem("FuelReserve", 11),
	})
}

// FuelReserveSystem ensures planets maintain a minimum fuel reserve
// for ship operations. Without fuel, the entire logistics chain breaks:
// ships can't move, routes don't complete, trade stops.
//
// Mechanics:
//   - Each owned planet with a Generator keeps a fuel reserve proportional to fleet size
//   - If fuel drops below reserve threshold, a warning fires
//   - Planets with Generators but 0 stored fuel get an emergency 10 Fuel injection
//     (representing basic generator output being diverted to fuel cells)
//   - This prevents the death spiral: no fuel → ships stranded → can't get fuel
//
// Reserve threshold = max(20, ships_in_system * 10)
//
// This runs at high priority (11) so fuel is available before shipping (29).
type FuelReserveSystem struct {
	*BaseSystem
	lastWarning map[int]int64 // planetID → last warning tick
}

func (frs *FuelReserveSystem) OnTick(tick int64) {
	if tick%50 != 0 {
		return
	}

	ctx := frs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if frs.lastWarning == nil {
		frs.lastWarning = make(map[int]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, sys := range systems {
		// Count ships needing fuel in this system
		shipsInSystem := 0
		for _, p := range players {
			if p == nil {
				continue
			}
			for _, ship := range p.OwnedShips {
				if ship != nil && ship.CurrentSystem == sys.ID && ship.Status != entities.ShipStatusMoving {
					shipsInSystem++
				}
			}
		}

		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			// Check if planet has a Generator
			hasGenerator := false
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingGenerator && b.IsOperational {
					hasGenerator = true
					break
				}
			}
			if !hasGenerator {
				continue
			}

			stored := planet.GetStoredAmount(entities.ResFuel)
			reserve := 20
			if shipsInSystem*10 > reserve {
				reserve = shipsInSystem * 10
			}
			if reserve > 100 {
				reserve = 100 // cap — don't set unreachable reserve for congested systems
			}

			// Warn if fuel is below reserve (reduced frequency)
			if stored < reserve {
				lastWarn := frs.lastWarning[planet.GetID()]
				if tick-lastWarn > 10000 {
					frs.lastWarning[planet.GetID()] = tick
					game.LogEvent("logistics", planet.Owner,
						fmt.Sprintf("⛽ Low fuel reserve on %s! %d/%d Fuel. Ships may be stranded. Build Refineries or import Fuel!",
							planet.Name, stored, reserve))
				}
			}
		}
	}
}
