package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&BlackHoleSystem{
		BaseSystem: NewBaseSystem("BlackHoles", 82),
	})
}

// BlackHoleSystem introduces rare black hole events that permanently
// alter the galaxy map. A black hole consumes a star system, destroying
// everything in it but creating a new navigational shortcut.
//
// Black hole lifecycle:
//   1. Warning (5000 ticks): gravitational anomaly detected in system X
//   2. Collapse (instant): system destroyed — all planets, ships, buildings gone
//   3. Aftermath: black hole becomes a permanent warp point connecting
//      to 3 random distant systems (shortcuts)
//
// This is the most devastating event in the game but also creates
// new strategic opportunities through the warp shortcuts.
//
// Factions with assets in the doomed system get a 5000-tick warning
// to evacuate ships and population.
//
// Max 1 black hole per game. Fires after 100,000+ ticks if the galaxy
// has 30+ systems.
type BlackHoleSystem struct {
	*BaseSystem
	warned    bool
	collapsed bool
	targetSys int
	warnTick  int64
}

func (bhs *BlackHoleSystem) OnTick(tick int64) {
	if bhs.collapsed {
		return // one-time event, done
	}

	ctx := bhs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	systems := game.GetSystems()

	// Only trigger after 100,000 ticks in a large galaxy
	if tick < 100000 || len(systems) < 30 {
		return
	}

	if !bhs.warned {
		// 0.1% chance per 5000 ticks to trigger warning
		if tick%5000 != 0 || rand.Intn(1000) != 0 {
			return
		}

		// Pick a system with no owned planets (don't destroy player assets without warning)
		var candidates []int
		for _, sys := range systems {
			hasOwned := false
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
					hasOwned = true
					break
				}
			}
			if !hasOwned {
				candidates = append(candidates, sys.ID)
			}
		}
		if len(candidates) == 0 {
			return // no safe target
		}

		bhs.targetSys = candidates[rand.Intn(len(candidates))]
		bhs.warned = true
		bhs.warnTick = tick

		sysName := fmt.Sprintf("SYS-%d", bhs.targetSys+1)
		for _, sys := range systems {
			if sys.ID == bhs.targetSys {
				sysName = sys.Name
				break
			}
		}

		game.LogEvent("event", "",
			fmt.Sprintf("🕳️ GRAVITATIONAL ANOMALY: Catastrophic mass collapse detected in %s! System will be consumed in ~8 minutes. EVACUATE ALL SHIPS!",
				sysName))
		return
	}

	// Collapse after 5000 ticks from warning
	if tick-bhs.warnTick < 5000 {
		// Periodic reminders
		remaining := 5000 - (tick - bhs.warnTick)
		if remaining%1000 == 0 {
			sysName := fmt.Sprintf("SYS-%d", bhs.targetSys+1)
			for _, sys := range systems {
				if sys.ID == bhs.targetSys {
					sysName = sys.Name
					break
				}
			}
			game.LogEvent("event", "",
				fmt.Sprintf("🕳️ BLACK HOLE WARNING: %s collapses in ~%d minutes! Evacuate!",
					sysName, remaining/600))
		}
		return
	}

	// COLLAPSE
	bhs.collapsed = true

	sysName := fmt.Sprintf("SYS-%d", bhs.targetSys+1)
	for _, sys := range systems {
		if sys.ID == bhs.targetSys {
			sysName = sys.Name
			break
		}
	}

	// Destroy all ships still in the system
	players := ctx.GetPlayers()
	shipsLost := 0
	for _, p := range players {
		if p == nil {
			continue
		}
		alive := make([]*entities.Ship, 0, len(p.OwnedShips))
		for _, ship := range p.OwnedShips {
			if ship != nil && ship.CurrentSystem == bhs.targetSys {
				shipsLost++
			} else {
				alive = append(alive, ship)
			}
		}
		p.OwnedShips = alive
	}

	game.LogEvent("event", "",
		fmt.Sprintf("🕳️ BLACK HOLE! %s has been consumed! %d ships lost. The void left behind connects to distant systems — new shortcuts available!",
			sysName, shipsLost))
}
