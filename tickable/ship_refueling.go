package tickable

import (
	"math"

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

	playersIface := ctx.GetPlayers()
	if playersIface == nil {
		return
	}
	players, ok := playersIface.([]*entities.Player)
	if !ok {
		return
	}

	sp, ok := ctx.GetGame().(SystemsProvider)
	if !ok {
		return
	}
	systems := sp.GetSystems()

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

			// Find planet at ship's orbit
			planet := findPlanetAtOrbit(ship, systems)
			if planet == nil || planet.Owner != player.Name {
				continue // not at an owned planet
			}

			// Refuel from planet's Fuel storage
			needed := ship.MaxFuel - ship.CurrentFuel
			available := planet.GetStoredAmount("Fuel")
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

			planet.RemoveStoredResource("Fuel", refuelAmount)
			ship.Refuel(refuelAmount)
		}
	}
}

func findPlanetAtOrbit(ship *entities.Ship, systems []*entities.System) *entities.Planet {
	for _, sys := range systems {
		if sys.ID != ship.CurrentSystem {
			continue
		}
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok {
				if math.Abs(ship.GetOrbitDistance()-planet.GetOrbitDistance()) < 1.0 {
					return planet
				}
			}
		}
		break
	}
	return nil
}
