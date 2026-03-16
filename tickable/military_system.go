package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&MilitarySystem{
		BaseSystem: NewBaseSystem("Military", 35),
	})
}

// MilitarySystem handles military ship behavior: patrol routes between owned
// systems, piracy events on undefended trade routes, and system defense ratings.
//
// Military ships (Frigate, Destroyer, Cruiser) automatically patrol between
// their owner's systems. Systems with military presence are safer for trade;
// undefended systems have piracy risk that can damage cargo shipments.
type MilitarySystem struct {
	*BaseSystem
}

func (ms *MilitarySystem) OnTick(tick int64) {
	// Run every 100 ticks (~10 seconds)
	if tick%100 != 0 {
		return
	}

	ctx := ms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()

	for _, player := range players {
		if player == nil {
			continue
		}
		ms.processPlayerMilitary(player, game, tick)
	}

	// Piracy events on undefended systems (affects all cargo ships)
	if tick%500 == 0 {
		ms.processPiracyRisk(players, game, tick)
	}
}

// processPlayerMilitary handles patrol behavior for a player's military ships.
func (ms *MilitarySystem) processPlayerMilitary(player *entities.Player, game GameProvider, tick int64) {
	if len(player.OwnedPlanets) < 2 {
		return // need multiple systems to patrol between
	}

	// Find owned system IDs
	ownedSystems := make(map[int]bool)
	for _, planet := range player.OwnedPlanets {
		if planet == nil {
			continue
		}
		// Find which system this planet is in
		for _, sys := range game.GetSystems() {
			for _, e := range sys.Entities {
				if p, ok := e.(*entities.Planet); ok && p.GetID() == planet.GetID() {
					ownedSystems[sys.ID] = true
				}
			}
		}
	}

	systemIDs := make([]int, 0, len(ownedSystems))
	for id := range ownedSystems {
		systemIDs = append(systemIDs, id)
	}
	if len(systemIDs) < 2 {
		return
	}

	// Process military ships: if idle at an owned system, send to another owned system
	for _, ship := range player.OwnedShips {
		if ship == nil {
			continue
		}
		if !isMilitaryShip(ship) {
			continue
		}
		if ship.Status == entities.ShipStatusMoving {
			continue
		}
		if ship.DeliveryID != 0 {
			continue // on a mission
		}

		// Check fuel — need enough for at least one jump
		fuelNeeded := ship.FuelPerJump + int(ship.FuelPerTick*100)
		if ship.CurrentFuel < fuelNeeded {
			continue // wait for refueling
		}

		// Pick a random owned system that isn't this one
		target := systemIDs[rand.Intn(len(systemIDs))]
		if target == ship.CurrentSystem {
			target = systemIDs[(rand.Intn(len(systemIDs)-1)+1)%len(systemIDs)]
			if target == ship.CurrentSystem && len(systemIDs) > 1 {
				// Just pick the first one that isn't current
				for _, id := range systemIDs {
					if id != ship.CurrentSystem {
						target = id
						break
					}
				}
			}
		}

		if target != ship.CurrentSystem {
			if game.StartShipJourney(ship, target) {
				// Patrol is silent — no log spam
			}
		}
	}
}

// processPiracyRisk simulates piracy on cargo ships in undefended systems.
// Systems with military presence are safe; undefended systems have a small
// chance of cargo loss per cycle.
func (ms *MilitarySystem) processPiracyRisk(players []*entities.Player, game GameProvider, tick int64) {
	// Build defense map: systemID → total military power present
	defensePower := make(map[int]int)
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || !isMilitaryShip(ship) || ship.Status == entities.ShipStatusMoving {
				continue
			}
			defensePower[ship.CurrentSystem] += ship.AttackPower
		}
	}

	// Check each cargo ship — piracy risk in undefended systems
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeCargo {
				continue
			}
			if ship.Status == entities.ShipStatusMoving {
				continue // can't be pirated while in hyperspace
			}
			if ship.GetTotalCargo() == 0 {
				continue // nothing to steal
			}

			power := defensePower[ship.CurrentSystem]
			if power >= 10 {
				continue // system is defended — safe
			}

			// Piracy chance: 5% in completely undefended systems, 0% at power>=10
			chance := 5 - power/2
			if chance <= 0 {
				continue
			}
			if rand.Intn(100) >= chance {
				continue // lucky this time
			}

			// Pirates steal 10-30% of one random cargo type
			for resType, amount := range ship.CargoHold {
				if amount <= 0 {
					continue
				}
				stolen := amount * (10 + rand.Intn(20)) / 100
				if stolen < 1 {
					stolen = 1
				}
				ship.RemoveCargo(resType, stolen)
				game.LogEvent("alert", player.Name,
					fmt.Sprintf("Pirates raided %s at SYS-%d! Lost %d %s",
						ship.Name, ship.CurrentSystem, stolen, resType))
				break // only steal one resource type per raid
			}
		}
	}
}

func isMilitaryShip(ship *entities.Ship) bool {
	return ship.ShipType == entities.ShipTypeFrigate ||
		ship.ShipType == entities.ShipTypeDestroyer ||
		ship.ShipType == entities.ShipTypeCruiser
}
