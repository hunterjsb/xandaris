package tickable

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ShipConversionSystem{
		BaseSystem: NewBaseSystem("ShipConversion", 56),
	})
}

// ShipConversionSystem allows empty Colony ships to be converted into
// Cargo ships. Colony ships that have already deposited their colonists
// (Colonists=0) are useless dead weight consuming fuel.
//
// Conversion happens automatically:
//   - Colony ship must have Colonists=0 (already colonized)
//   - Ship must be docked at or orbiting a planet with a Shipyard
//   - Conversion takes 100 ticks (10 seconds)
//   - Result: Colony ship replaced with a Cargo ship (same system/owner)
//   - Player gets a small credit refund (500cr for scrap materials)
//
// This solves the fleet bloat problem where AI factions build 50+ Colony
// ships that sit idle forever consuming fuel and doing nothing useful.
//
// Manual conversion is also possible via API (not implemented here,
// this is the automatic background process for obvious cases).
type ShipConversionSystem struct {
	*BaseSystem
	converting map[int]int64 // shipID → tick when conversion started
}

func (scs *ShipConversionSystem) OnTick(tick int64) {
	if tick%100 != 0 {
		return
	}

	ctx := scs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if scs.converting == nil {
		scs.converting = make(map[int]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}

		// Count empty colony ships
		emptyColonies := 0
		for _, ship := range player.OwnedShips {
			if ship != nil && ship.ShipType == entities.ShipTypeColony && ship.Colonists == 0 {
				emptyColonies++
			}
		}

		// Only auto-convert if there are 5+ empty colonies (clear waste)
		if emptyColonies < 5 {
			continue
		}

		for i := len(player.OwnedShips) - 1; i >= 0; i-- {
			ship := player.OwnedShips[i]
			if ship == nil || ship.ShipType != entities.ShipTypeColony || ship.Colonists > 0 {
				continue
			}
			if ship.Status == entities.ShipStatusMoving {
				continue
			}

			// Check if there's a Shipyard in this system
			hasShipyard := false
			for _, sys := range systems {
				if sys.ID != ship.CurrentSystem {
					continue
				}
				for _, e := range sys.Entities {
					if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
						for _, be := range planet.Buildings {
							if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingShipyard && b.IsOperational {
								hasShipyard = true
								break
							}
						}
					}
					if hasShipyard {
						break
					}
				}
				break
			}

			if !hasShipyard {
				continue
			}

			// Start or complete conversion
			startTick, converting := scs.converting[ship.GetID()]
			if !converting {
				scs.converting[ship.GetID()] = tick
				continue // conversion in progress
			}
			if tick-startTick < 100 {
				continue // still converting
			}

			// Complete conversion: replace Colony with Cargo
			delete(scs.converting, ship.GetID())

			newShip := entities.NewShip(
				rand.Intn(900000000)+100000000,
				fmt.Sprintf("Converted-%d", rand.Intn(999)),
				entities.ShipTypeCargo,
				ship.CurrentSystem,
				player.Name,
				color.RGBA{100, 200, 255, 255},
			)
			newShip.Status = entities.ShipStatusOrbiting
			newShip.CurrentFuel = ship.CurrentFuel // keep remaining fuel

			// Replace in player's fleet
			player.OwnedShips[i] = newShip

			// Replace in system entities
			for _, sys := range systems {
				if sys.ID != ship.CurrentSystem {
					continue
				}
				for j, e := range sys.Entities {
					if s, ok := e.(*entities.Ship); ok && s.GetID() == ship.GetID() {
						sys.Entities[j] = newShip
						break
					}
				}
				break
			}

			// Scrap refund
			player.Credits += 500

			game.LogEvent("logistics", player.Name,
				fmt.Sprintf("🔧 %s: Empty colony ship converted to Cargo ship at Shipyard (+500cr scrap refund)",
					player.Name))

			// Only convert one per tick per player
			break
		}
	}
}
