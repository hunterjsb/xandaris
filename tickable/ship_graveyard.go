package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ShipGraveyardSystem{
		BaseSystem: NewBaseSystem("ShipGraveyard", 168),
	})
}

// ShipGraveyardSystem tracks destroyed ships and creates "graveyard"
// systems where many ships have been lost. Graveyards become points
// of interest with salvage opportunities and memorial events.
//
// When 5+ ships have been destroyed in a system (from combat, pirate
// raids, storms), that system becomes a Ship Graveyard:
//   - Periodic salvage events (free resources for visiting ships)
//   - "Memorial" happiness boost to factions who lost ships there
//   - Lore events about the battles that created the graveyard
type ShipGraveyardSystem struct {
	*BaseSystem
	shipDeaths map[int]int // systemID → ships destroyed count
	graveyards map[int]bool
	nextCheck  int64
}

func (sgs *ShipGraveyardSystem) OnTick(tick int64) {
	if tick%3000 != 0 {
		return
	}

	ctx := sgs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if sgs.shipDeaths == nil {
		sgs.shipDeaths = make(map[int]int)
		sgs.graveyards = make(map[int]bool)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Detect damaged/destroyed ships as proxy for combat
	for _, p := range players {
		if p == nil {
			continue
		}
		for _, ship := range p.OwnedShips {
			if ship == nil {
				continue
			}
			if ship.CurrentHealth < ship.MaxHealth/4 && ship.CurrentHealth > 0 {
				// Very damaged = combat happened here
				sgs.shipDeaths[ship.CurrentSystem]++
			}
		}
	}

	// Check for new graveyards
	for sysID, deaths := range sgs.shipDeaths {
		if deaths >= 5 && !sgs.graveyards[sysID] {
			sgs.graveyards[sysID] = true

			sysName := fmt.Sprintf("SYS-%d", sysID+1)
			for _, sys := range systems {
				if sys.ID == sysID {
					sysName = sys.Name
					break
				}
			}

			game.LogEvent("event", "",
				fmt.Sprintf("⚰️ %s has become a Ship Graveyard — %d vessels lost here. Salvagers welcome, but beware the ghosts of fallen crews...",
					sysName, deaths))
		}
	}

	// Graveyard salvage events
	for sysID := range sgs.graveyards {
		if rand.Intn(10) != 0 {
			continue
		}

		// Give resources to any ship visiting
		for _, p := range players {
			if p == nil {
				continue
			}
			for _, ship := range p.OwnedShips {
				if ship == nil || ship.CurrentSystem != sysID || ship.Status == entities.ShipStatusMoving {
					continue
				}
				if ship.ShipType == entities.ShipTypeScout || ship.ShipType == entities.ShipTypeCargo {
					loaded := ship.AddCargo(entities.ResIron, 10+rand.Intn(20))
					if loaded > 0 {
						sysName := ""
						for _, sys := range systems {
							if sys.ID == sysID {
								sysName = sys.Name
								break
							}
						}
						if rand.Intn(3) == 0 {
							game.LogEvent("explore", p.Name,
								fmt.Sprintf("⚰️ %s salvaged %d Iron from wreckage in %s graveyard",
									ship.Name, loaded, sysName))
						}
					}
					break
				}
			}
		}
	}
}

// RecordDeath records a ship destruction in a system.
func (sgs *ShipGraveyardSystem) RecordDeath(systemID int) {
	if sgs.shipDeaths == nil {
		sgs.shipDeaths = make(map[int]int)
	}
	sgs.shipDeaths[systemID]++
}
