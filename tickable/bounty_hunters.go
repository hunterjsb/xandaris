package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&BountyHunterSystem{
		BaseSystem: NewBaseSystem("BountyHunters", 103),
	})
}

// BountyHunterSystem spawns NPC bounty hunter ships that pursue
// factions with negative galactic reputation (from aggression,
// sanctions, piracy, market manipulation).
//
// Bounty hunters:
//   - Appear when a faction accumulates 3+ "infractions" (from other systems)
//   - Chase the target faction's ships across systems
//   - If they catch a ship: disable it (health → 25%) and steal 50% cargo
//   - Can be fought off by military escorts (Frigate+ defeats hunter)
//   - Disappear after 10,000 ticks or if target pays a 5000cr "clearance fee"
//
// This creates consequences for aggressive play beyond sanctions:
// actual NPC ships hunting your fleet.
type BountyHunterSystem struct {
	*BaseSystem
	hunters    []*BountyHunter
	infractions map[string]int
	nextSpawn  int64
}

// BountyHunter tracks an NPC hunter pursuing a faction.
type BountyHunter struct {
	TargetFaction string
	CurrentSystem int
	TicksLeft     int
	Active        bool
}

func (bhs *BountyHunterSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := bhs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if bhs.infractions == nil {
		bhs.infractions = make(map[string]int)
	}

	players := ctx.GetPlayers()

	// Process active hunters
	for _, hunter := range bhs.hunters {
		if !hunter.Active {
			continue
		}
		hunter.TicksLeft -= 500
		if hunter.TicksLeft <= 0 {
			hunter.Active = false
			continue
		}

		bhs.processHunter(hunter, players, game)
	}

	// Spawn hunters for factions with high infractions
	if bhs.nextSpawn == 0 {
		bhs.nextSpawn = tick + 10000
	}
	if tick >= bhs.nextSpawn {
		bhs.nextSpawn = tick + 15000 + int64(rand.Intn(10000))

		for name, count := range bhs.infractions {
			if count < 3 {
				continue
			}

			// Check not already hunted
			alreadyHunted := false
			for _, h := range bhs.hunters {
				if h.Active && h.TargetFaction == name {
					alreadyHunted = true
					break
				}
			}
			if alreadyHunted {
				continue
			}

			// Find a target ship to determine starting system
			startSys := -1
			for _, p := range players {
				if p == nil || p.Name != name {
					continue
				}
				for _, ship := range p.OwnedShips {
					if ship != nil {
						startSys = ship.CurrentSystem
						break
					}
				}
				break
			}
			if startSys < 0 {
				continue
			}

			bhs.hunters = append(bhs.hunters, &BountyHunter{
				TargetFaction: name,
				CurrentSystem: startSys,
				TicksLeft:     10000,
				Active:        true,
			})
			bhs.infractions[name] = 0

			game.LogEvent("event", name,
				fmt.Sprintf("🎯 BOUNTY HUNTER dispatched against %s! An NPC hunter is tracking your ships. Pay 5000cr clearance or defend with military escorts!",
					name))
		}
	}
}

func (bhs *BountyHunterSystem) processHunter(hunter *BountyHunter, players []*entities.Player, game GameProvider) {
	for _, p := range players {
		if p == nil || p.Name != hunter.TargetFaction {
			continue
		}

		// Find target ships in hunter's system
		for _, ship := range p.OwnedShips {
			if ship == nil || ship.CurrentSystem != hunter.CurrentSystem || ship.Status == entities.ShipStatusMoving {
				continue
			}

			// Check for military escort
			hasEscort := false
			for _, other := range p.OwnedShips {
				if other != nil && other.CurrentSystem == hunter.CurrentSystem &&
					(other.ShipType == entities.ShipTypeFrigate ||
						other.ShipType == entities.ShipTypeDestroyer ||
						other.ShipType == entities.ShipTypeCruiser) {
					hasEscort = true
					break
				}
			}

			if hasEscort {
				// Escort fights off hunter
				hunter.Active = false
				p.Credits += 1000
				game.LogEvent("event", p.Name,
					fmt.Sprintf("⚔️ %s's military escort fought off the bounty hunter! +1000cr bounty collected",
						p.Name))
				return
			}

			// Hunter catches unescorted ship
			if ship.ShipType == entities.ShipTypeCargo && rand.Intn(3) == 0 {
				ship.CurrentHealth = ship.MaxHealth / 4
				// Steal cargo
				for res, amt := range ship.CargoHold {
					stolen := amt / 2
					ship.CargoHold[res] -= stolen
					if ship.CargoHold[res] <= 0 {
						delete(ship.CargoHold, res)
					}
				}
				game.LogEvent("event", p.Name,
					fmt.Sprintf("🎯 Bounty hunter caught %s's %s! Ship disabled, 50%% cargo stolen. Get military escorts!",
						p.Name, ship.Name))
				return
			}
		}

		// Move hunter toward target ships
		for _, ship := range p.OwnedShips {
			if ship != nil && ship.CurrentSystem != hunter.CurrentSystem {
				connected := game.GetConnectedSystems(hunter.CurrentSystem)
				if len(connected) > 0 {
					hunter.CurrentSystem = connected[rand.Intn(len(connected))]
				}
				break
			}
		}
		break
	}
}

// AddInfraction records an infraction for a faction.
func (bhs *BountyHunterSystem) AddInfraction(faction string) {
	if bhs.infractions == nil {
		bhs.infractions = make(map[string]int)
	}
	bhs.infractions[faction]++
}

// PayClearance removes active hunters for a faction.
func (bhs *BountyHunterSystem) PayClearance(faction string) bool {
	for _, h := range bhs.hunters {
		if h.Active && h.TargetFaction == faction {
			h.Active = false
			return true
		}
	}
	return false
}
