package tickable

import (
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ShipRefuelingSystem{
		BaseSystem: NewBaseSystem("ShipRefueling", 20),
	})
}

// ShipRefuelingSystem refuels ships from planet Fuel storage when orbiting owned planets.
// Creates real demand for Fuel from the fleet — ships consume Fuel to operate.
type ShipRefuelingSystem struct {
	*BaseSystem
}

func (srs *ShipRefuelingSystem) OnTick(tick int64) {
	if tick%10 != 0 {
		return
	}

	ctx := srs.GetContext()
	if ctx == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := ctx.GetGame().GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.Status == entities.ShipStatusMoving {
				continue
			}
			if ship.CurrentFuel >= ship.MaxFuel {
				continue // already full
			}

			// Find owned planet in this system
			planet := findPlanetAtOrbit(ship, systems)
			if planet == nil {
				continue // no owned planet in this system
			}

			// Refuel from planet's Fuel storage
			needed := ship.MaxFuel - ship.CurrentFuel
			available := planet.GetStoredAmount(entities.ResFuel)
			if available <= 0 {
				continue
			}

			// Take up to 5 fuel per interval (gradual refueling)
			refuelAmount := needed
			if refuelAmount > 5 {
				refuelAmount = 5
			}
			if refuelAmount > available {
				refuelAmount = available
			}

			planet.RemoveStoredResource(entities.ResFuel, refuelAmount)
			ship.Refuel(refuelAmount)
		}
	}
}

func findPlanetAtOrbit(ship *entities.Ship, systems []*entities.System) *entities.Planet {
	for _, sys := range systems {
		if sys.ID != ship.CurrentSystem {
			continue
		}
		// Find any owned planet in this system (for refueling, exact orbit doesn't matter)
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner == ship.Owner {
				return planet
			}
		}
		break
	}
	return nil
}
