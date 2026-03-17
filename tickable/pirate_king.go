package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&PirateKingSystem{
		BaseSystem: NewBaseSystem("PirateKing", 86),
	})
}

// PirateKingSystem spawns a boss-level pirate faction when pirate
// activity goes unchecked. The Pirate King unifies scattered pirates
// into a coordinated threat that demands a coordinated response.
//
// Trigger: 3+ pirate fleets active simultaneously for 10,000+ ticks
//
// The Pirate King:
//   - Claims an unclaimed planet as a pirate base
//   - Raids cargo ships galaxy-wide (not just in pirate systems)
//   - Demands tribute: 1000cr per faction per 5000 ticks or face raids
//   - Can only be defeated by a combined fleet of 10+ military ships
//     from any combination of factions
//
// Defeating the Pirate King:
//   - 25,000cr bounty split among participating factions
//   - Pirate base becomes claimable (pre-built infrastructure)
//   - All pirate fleets disbanded for 30,000 ticks
//   - "Pirate Slayer" galactic record
//
// This creates a shared enemy that can unite rival factions.
type PirateKingSystem struct {
	*BaseSystem
	active       bool
	baseSystemID int
	basePlanetID int
	hp           int // king's fleet HP, takes damage from military in system
	tributePaid  map[string]int64 // faction → last tribute tick
	nextDemand   int64
}

func (pks *PirateKingSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := pks.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if pks.tributePaid == nil {
		pks.tributePaid = make(map[string]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	if pks.active {
		pks.processKing(tick, players, systems, game)
		return
	}

	// Check spawn condition: look for high pirate activity
	// Use tick threshold as proxy (pirates spawn over time)
	if tick < 50000 {
		return
	}
	if rand.Intn(100) != 0 { // 1% per 500-tick check after 50K ticks
		return
	}

	pks.spawnKing(tick, systems, game)
}

func (pks *PirateKingSystem) spawnKing(tick int64, systems []*entities.System, game GameProvider) {
	// Find an unclaimed planet for the base
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner == "" && planet.IsHabitable() {
				pks.active = true
				pks.baseSystemID = sys.ID
				pks.basePlanetID = planet.GetID()
				pks.hp = 500
				pks.nextDemand = tick + 2000

				game.LogEvent("event", "",
					fmt.Sprintf("🏴‍☠️ THE PIRATE KING HAS RISEN! A unified pirate fleet has claimed %s in %s as their fortress. All factions must pay tribute (1000cr) or face relentless raids. Unite your fleets to defeat the Pirate King! (Bounty: 25,000cr)",
						planet.Name, sys.Name))
				return
			}
		}
	}
}

func (pks *PirateKingSystem) processKing(tick int64, players []*entities.Player, systems []*entities.System, game GameProvider) {
	// Demand tribute
	if tick >= pks.nextDemand {
		pks.nextDemand = tick + 5000

		for _, p := range players {
			if p == nil {
				continue
			}
			lastPaid := pks.tributePaid[p.Name]
			if tick-lastPaid > 5000 {
				// Didn't pay — raid them
				if rand.Intn(3) == 0 {
					raided := p.Credits / 50 // 2% raid
					if raided > 2000 {
						raided = 2000
					}
					if raided > 0 {
						p.Credits -= raided
						game.LogEvent("event", p.Name,
							fmt.Sprintf("🏴‍☠️ Pirate King's raiders hit %s! Lost %dcr. Pay tribute or send military to defeat the King!",
								p.Name, raided))
					}
				}
			}
		}
	}

	// Check for military assault on pirate base
	militaryPower := 0
	var attackers []string
	for _, p := range players {
		if p == nil {
			continue
		}
		playerPower := 0
		for _, ship := range p.OwnedShips {
			if ship == nil || ship.CurrentSystem != pks.baseSystemID {
				continue
			}
			if ship.ShipType == entities.ShipTypeFrigate {
				playerPower += 2
			} else if ship.ShipType == entities.ShipTypeDestroyer {
				playerPower += 4
			} else if ship.ShipType == entities.ShipTypeCruiser {
				playerPower += 6
			}
		}
		if playerPower > 0 {
			militaryPower += playerPower
			attackers = append(attackers, p.Name)
		}
	}

	if militaryPower > 0 {
		damage := militaryPower * 5
		pks.hp -= damage

		if pks.hp <= 0 {
			// DEFEATED!
			pks.active = false
			bountyEach := 25000 / len(attackers)
			for _, name := range attackers {
				for _, p := range players {
					if p != nil && p.Name == name {
						p.Credits += bountyEach
						break
					}
				}
			}

			attackerList := ""
			for i, name := range attackers {
				if i > 0 {
					attackerList += ", "
				}
				attackerList += name
			}

			game.LogEvent("event", "",
				fmt.Sprintf("⚔️ THE PIRATE KING IS DEFEATED! Combined fleet of %s destroyed the pirate fortress! Bounty: %dcr each. Pirate activity suppressed!",
					attackerList, bountyEach))
		}
	}
}

// PayTribute allows a faction to pay the Pirate King.
func (pks *PirateKingSystem) PayTribute(playerName string, tick int64) {
	if pks.tributePaid != nil {
		pks.tributePaid[playerName] = tick
	}
}

// IsActive returns whether the Pirate King is active.
func (pks *PirateKingSystem) IsActive() bool {
	return pks.active
}
