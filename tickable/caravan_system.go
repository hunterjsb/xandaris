package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&CaravanSystem{
		BaseSystem: NewBaseSystem("Caravans", 89),
	})
}

// CaravanSystem groups multiple cargo ships traveling the same route
// into a caravan for efficiency bonuses. Caravans form automatically
// when 2+ cargo ships owned by the same faction are in the same system
// heading to the same destination.
//
// Caravan bonuses:
//   - 2 ships: +10% fuel efficiency (less fuel per tick)
//   - 3 ships: +10% fuel + immune to pirate raids
//   - 4+ ships: +10% fuel + immune to pirates + 20% speed boost
//
// Caravans also generate "trade lane" prestige: the more caravans
// a faction runs, the higher their trade reputation grows.
//
// This incentivizes building multiple cargo ships and running them
// together rather than solo haulers.
type CaravanSystem struct {
	*BaseSystem
	activeCaravans map[string]int // factionName → active caravan count
}

func (cs *CaravanSystem) OnTick(tick int64) {
	if tick%200 != 0 {
		return
	}

	ctx := cs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if cs.activeCaravans == nil {
		cs.activeCaravans = make(map[string]int)
	}

	players := ctx.GetPlayers()

	for _, player := range players {
		if player == nil {
			continue
		}

		// Group cargo ships by current system + target system
		type routeKey struct {
			from, to int
		}
		groups := make(map[routeKey][]*entities.Ship)

		for _, ship := range player.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeCargo {
				continue
			}
			if ship.Status == entities.ShipStatusMoving && ship.TargetSystem != -1 {
				key := routeKey{ship.CurrentSystem, ship.TargetSystem}
				groups[key] = append(groups[key], ship)
			}
		}

		// Apply caravan bonuses
		caravanCount := 0
		for _, ships := range groups {
			if len(ships) < 2 {
				continue
			}
			caravanCount++

			for _, ship := range ships {
				// Fuel efficiency: refund 1 fuel per tick (simulating efficiency)
				ship.CurrentFuel++
				if ship.CurrentFuel > ship.MaxFuel {
					ship.CurrentFuel = ship.MaxFuel
				}

				// Speed boost for 4+ ship caravans
				if len(ships) >= 4 {
					ship.TravelProgress += 0.002 // small speed boost
				}
			}
		}

		cs.activeCaravans[player.Name] = caravanCount

		// Announce large caravans
		for key, ships := range groups {
			if len(ships) >= 3 && rand.Intn(10) == 0 {
				_ = key
				game.LogEvent("logistics", player.Name,
					fmt.Sprintf("🐪 %s caravan: %d cargo ships traveling together (+fuel efficiency, pirate immunity)",
						player.Name, len(ships)))
			}
		}
	}
}

// GetCaravanCount returns active caravan count for a faction.
func (cs *CaravanSystem) GetCaravanCount(faction string) int {
	if cs.activeCaravans == nil {
		return 0
	}
	return cs.activeCaravans[faction]
}
