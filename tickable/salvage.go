package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SalvageSystem{
		BaseSystem: NewBaseSystem("Salvage", 39),
	})
}

// SalvageSystem manages wreckage fields left behind by destroyed ships
// and pirate battles. Wreckage can be salvaged by any Scout or Cargo ship
// that visits the system, yielding resources and sometimes credits.
//
// Wreckage types:
//   - Ship debris: destroyed military/cargo ships leave Iron + Rare Metals
//   - Pirate loot: defeated pirate fleets leave stolen goods + bounty bonus
//   - Battle salvage: large battles leave massive debris fields
//
// Wreckage decays over time (5000-10000 ticks). First come, first served.
// This incentivizes scouting battle zones and pirate-cleared systems.
type SalvageSystem struct {
	*BaseSystem
	wreckage map[int][]*Wreckage // systemID → wreckage list
}

// Wreckage represents recoverable debris in a system.
type Wreckage struct {
	SystemID  int
	Source    string // "ship", "pirate", "battle"
	Resources map[string]int
	Credits   int
	TicksLeft int
	Claimed   bool
}

func (ss *SalvageSystem) OnTick(tick int64) {
	if tick%100 != 0 {
		return
	}

	ctx := ss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ss.wreckage == nil {
		ss.wreckage = make(map[int][]*Wreckage)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Decay wreckage
	for sysID, wrecks := range ss.wreckage {
		alive := make([]*Wreckage, 0, len(wrecks))
		for _, w := range wrecks {
			w.TicksLeft -= 100
			if w.TicksLeft > 0 && !w.Claimed {
				alive = append(alive, w)
			}
		}
		if len(alive) == 0 {
			delete(ss.wreckage, sysID)
		} else {
			ss.wreckage[sysID] = alive
		}
	}

	// Check for ships that can salvage
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.Status == entities.ShipStatusMoving {
				continue
			}
			// Scouts and Cargo can salvage
			if ship.ShipType != entities.ShipTypeScout && ship.ShipType != entities.ShipTypeCargo {
				continue
			}

			wrecks := ss.wreckage[ship.CurrentSystem]
			if len(wrecks) == 0 {
				continue
			}

			// Try to salvage the first unclaimed wreck
			for _, w := range wrecks {
				if w.Claimed {
					continue
				}

				ss.claimSalvage(game, player, ship, w, systems)
				break // one salvage per ship per tick
			}
		}
	}

	// Generate wreckage from recent combat (check for damaged ships as proxy)
	ss.generateBattleSalvage(tick, game, players, systems)
}

func (ss *SalvageSystem) claimSalvage(game GameProvider, player *entities.Player, ship *entities.Ship, w *Wreckage, systems []*entities.System) {
	w.Claimed = true

	// Load resources into ship's cargo
	totalLoaded := 0
	for res, amt := range w.Resources {
		loaded := ship.AddCargo(res, amt)
		totalLoaded += loaded
	}

	// Direct credit rewards
	if w.Credits > 0 {
		player.Credits += w.Credits
	}

	sysName := fmt.Sprintf("SYS-%d", w.SystemID+1)
	for _, sys := range systems {
		if sys.ID == w.SystemID {
			sysName = sys.Name
			break
		}
	}

	msg := fmt.Sprintf("♻️ %s salvaged %s wreckage in %s!", ship.Name, w.Source, sysName)
	if totalLoaded > 0 {
		msg += fmt.Sprintf(" Recovered %d units of resources", totalLoaded)
	}
	if w.Credits > 0 {
		msg += fmt.Sprintf(" + %d credits", w.Credits)
	}
	game.LogEvent("explore", player.Name, msg)
}

func (ss *SalvageSystem) generateBattleSalvage(tick int64, game GameProvider, players []*entities.Player, systems []*entities.System) {
	// Check for very damaged ships as evidence of recent battle
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil {
				continue
			}
			// If a military ship is below 30% health, a battle happened here
			if ship.ShipType != entities.ShipTypeFrigate &&
				ship.ShipType != entities.ShipTypeDestroyer &&
				ship.ShipType != entities.ShipTypeCruiser {
				continue
			}
			if ship.GetHealthPercentage() > 30 {
				continue
			}

			// 5% chance to leave wreckage this tick
			if rand.Intn(20) != 0 {
				continue
			}

			// Check if wreckage already exists from this battle
			existing := ss.wreckage[ship.CurrentSystem]
			if len(existing) >= 3 {
				continue // don't flood a system
			}

			w := &Wreckage{
				SystemID:  ship.CurrentSystem,
				Source:    "battle",
				Resources: map[string]int{
					entities.ResIron:       20 + rand.Intn(50),
					entities.ResRareMetals: 5 + rand.Intn(20),
				},
				Credits:   500 + rand.Intn(2000),
				TicksLeft: 5000 + rand.Intn(5000),
			}
			ss.wreckage[ship.CurrentSystem] = append(ss.wreckage[ship.CurrentSystem], w)

			sysName := fmt.Sprintf("SYS-%d", ship.CurrentSystem+1)
			for _, sys := range systems {
				if sys.ID == ship.CurrentSystem {
					sysName = sys.Name
					break
				}
			}
			game.LogEvent("event", "",
				fmt.Sprintf("♻️ Battle wreckage detected in %s! Send a Scout or Cargo ship to salvage", sysName))
		}
	}
}

// AddWreckage lets other systems (like PirateFleets) deposit wreckage.
func (ss *SalvageSystem) AddWreckage(systemID int, source string, resources map[string]int, credits int) {
	if ss.wreckage == nil {
		ss.wreckage = make(map[int][]*Wreckage)
	}
	ss.wreckage[systemID] = append(ss.wreckage[systemID], &Wreckage{
		SystemID:  systemID,
		Source:    source,
		Resources: resources,
		Credits:   credits,
		TicksLeft: 8000 + rand.Intn(4000),
	})
}

// GetWreckageCount returns the number of active wreckage fields (for API).
func (ss *SalvageSystem) GetWreckageCount() int {
	total := 0
	for _, wrecks := range ss.wreckage {
		for _, w := range wrecks {
			if !w.Claimed && w.TicksLeft > 0 {
				total++
			}
		}
	}
	return total
}
