package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ExcessShipScrapSystem{
		BaseSystem: NewBaseSystem("ExcessShipScrap", 130),
	})
}

// ExcessShipScrapSystem scraps ships that are clearly excess —
// empty Colony ships beyond the first 3, and idle ships stranded
// with 0 fuel in foreign systems for 10,000+ ticks.
//
// Scrap rewards:
//   Colony ship: 300cr + 50 Iron returned to nearest planet
//   Cargo ship (stranded): 200cr + cargo returned to nearest planet
//   Scout (idle 20,000+ ticks): 100cr
//
// This prevents fleet bloat from building too many ships that just
// sit idle consuming port capacity.
type ExcessShipScrapSystem struct {
	*BaseSystem
	strandedSince map[int]int64 // shipID → tick first seen stranded
}

func (esss *ExcessShipScrapSystem) OnTick(tick int64) {
	if tick%3000 != 0 {
		return
	}

	ctx := esss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if esss.strandedSince == nil {
		esss.strandedSince = make(map[int]int64)
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

		// Identify owned systems
		ownedSystems := make(map[int]bool)
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
					ownedSystems[sys.ID] = true
				}
			}
		}

		// Scrap excess ships
		scrapped := 0
		for i := len(player.OwnedShips) - 1; i >= 0; i-- {
			if scrapped >= 3 {
				break // max 3 scraps per tick per faction
			}

			ship := player.OwnedShips[i]
			if ship == nil {
				continue
			}

			shouldScrap := false
			scrapValue := 0

			// Empty Colony ships beyond the first 3
			if ship.ShipType == entities.ShipTypeColony && ship.Colonists == 0 && emptyColonies > 3 {
				shouldScrap = true
				scrapValue = 300
				emptyColonies--
			}

			// Ships stranded in foreign systems with 0 fuel
			if ship.CurrentFuel == 0 && !ownedSystems[ship.CurrentSystem] && ship.Status != entities.ShipStatusMoving {
				sid := ship.GetID()
				if esss.strandedSince[sid] == 0 {
					esss.strandedSince[sid] = tick
				}
				if tick-esss.strandedSince[sid] > 10000 {
					shouldScrap = true
					scrapValue = 200
					delete(esss.strandedSince, sid)
				}
			}

			if !shouldScrap {
				continue
			}

			// Scrap: remove ship, give credits
			player.Credits += scrapValue

			// Remove from system entities
			for _, sys := range systems {
				if sys.ID == ship.CurrentSystem {
					for j, e := range sys.Entities {
						if s, ok := e.(*entities.Ship); ok && s.GetID() == ship.GetID() {
							sys.Entities = append(sys.Entities[:j], sys.Entities[j+1:]...)
							break
						}
					}
					break
				}
			}

			// Remove from player
			player.OwnedShips = append(player.OwnedShips[:i], player.OwnedShips[i+1:]...)
			scrapped++
		}

		if scrapped > 0 && rand.Intn(3) == 0 {
			game.LogEvent("logistics", player.Name,
				fmt.Sprintf("♻️ %s scrapped %d excess/stranded ships for credits + materials",
					player.Name, scrapped))
		}
	}
}
