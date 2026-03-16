package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&IdleShipRecallSystem{
		BaseSystem: NewBaseSystem("IdleShipRecall", 31),
	})
}

// IdleShipRecallSystem is a lightweight safety net for stranded cargo ships.
// It does NOT make trade decisions — that's the LLM agent's job.
// It only handles two cases:
//  1. Ship idle at a foreign system → send it home
//  2. Ship at home with cargo → unload it
//
// This prevents ships from sitting forever after the AI logistics system was removed.
type IdleShipRecallSystem struct {
	*BaseSystem
}

func (isrs *IdleShipRecallSystem) OnTick(tick int64) {
	if tick%100 != 0 {
		return
	}

	ctx := isrs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeCargo {
				continue
			}
			if ship.Status == entities.ShipStatusMoving || ship.DeliveryID != 0 {
				continue
			}

			homeSystem := findHomeSystemID(player, systems)
			if homeSystem < 0 {
				continue
			}

			atHome := ship.CurrentSystem == homeSystem

			if !atHome && ship.GetTotalCargo() == 0 {
				// Idle and empty at a foreign system — go home
				fuelPerTrip := ship.FuelPerJump + int(ship.FuelPerTick*120)
				if ship.CurrentFuel >= fuelPerTrip {
					if game.StartShipJourney(ship, homeSystem) {
						fmt.Printf("[IdleRecall] %s returning home to SYS-%d\n", ship.Name, homeSystem+1)
					}
				}
			}
		}
	}
}

func findHomeSystemID(player *entities.Player, systems []*entities.System) int {
	if player.HomeSystem != nil {
		return player.HomeSystem.ID
	}
	if len(player.OwnedPlanets) > 0 && player.OwnedPlanets[0] != nil {
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if p, ok := e.(*entities.Planet); ok && p.GetID() == player.OwnedPlanets[0].GetID() {
					return sys.ID
				}
			}
		}
	}
	return -1
}
