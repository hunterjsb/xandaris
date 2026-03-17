package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SmugglingSystem{
		BaseSystem: NewBaseSystem("Smuggling", 55),
	})
}

// SmugglingSystem creates underground trade routes that bypass tariffs
// and sanctions but carry risk of interception.
//
// Smuggling happens automatically when:
//   - A cargo ship carries goods through a blockaded system
//   - A sanctioned faction's cargo ship enters a foreign system
//   - A cargo ship carries contraband (Electronics during scarcity)
//
// Risk/reward:
//   - Successful smuggle: goods arrive + 50% bonus credits
//   - Caught (30% chance): cargo seized, faction reputation -100,
//     pilot fined 2x cargo value
//   - Having a Scout in the system reduces catch chance to 10%
//     (the scout acts as a lookout)
//
// This creates emergent gameplay: sanctions make trade harder,
// but smuggling provides a risky workaround. Military factions
// can patrol systems to catch smugglers, earning bounties.
type SmugglingSystem struct {
	*BaseSystem
	smuggleAttempts map[string]int // playerName → successful smuggle count
	caughtCount     map[string]int // playerName → times caught
}

func (ss *SmugglingSystem) OnTick(tick int64) {
	if tick%400 != 0 {
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

	if ss.smuggleAttempts == nil {
		ss.smuggleAttempts = make(map[string]int)
		ss.caughtCount = make(map[string]int)
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
			if ship.Status == entities.ShipStatusMoving || ship.GetTotalCargo() == 0 {
				continue
			}

			// Check if this ship is in a foreign-controlled system
			isForeign := true
			for _, sys := range systems {
				if sys.ID != ship.CurrentSystem {
					continue
				}
				for _, e := range sys.Entities {
					if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
						isForeign = false
						break
					}
				}
				break
			}

			if !isForeign {
				continue // not smuggling if in your own system
			}

			// Check if smuggling conditions apply
			isSmuggling := false
			reason := ""

			// Condition 1: carrying goods through a system with enemy presence
			for _, p := range players {
				if p == nil || p.Name == player.Name {
					continue
				}
				for _, enemyShip := range p.OwnedShips {
					if enemyShip == nil || enemyShip.CurrentSystem != ship.CurrentSystem {
						continue
					}
					if enemyShip.ShipType == entities.ShipTypeFrigate ||
						enemyShip.ShipType == entities.ShipTypeDestroyer ||
						enemyShip.ShipType == entities.ShipTypeCruiser {
						dm := game.GetDiplomacyManager()
						if dm != nil {
							rel := dm.GetRelation(player.Name, p.Name)
							if rel <= -1 {
								isSmuggling = true
								reason = fmt.Sprintf("running cargo past %s's patrol", p.Name)
								break
							}
						}
					}
				}
				if isSmuggling {
					break
				}
			}

			if !isSmuggling {
				continue
			}

			// Determine catch chance
			catchChance := 30 // base 30%

			// Scout reduces catch chance
			hasScout := false
			for _, s := range player.OwnedShips {
				if s != nil && s.ShipType == entities.ShipTypeScout && s.CurrentSystem == ship.CurrentSystem {
					hasScout = true
					break
				}
			}
			if hasScout {
				catchChance = 10 // scout lookout
			}

			sysName := fmt.Sprintf("SYS-%d", ship.CurrentSystem+1)
			for _, sys := range systems {
				if sys.ID == ship.CurrentSystem {
					sysName = sys.Name
					break
				}
			}

			if rand.Intn(100) < catchChance {
				// CAUGHT! Cargo seized, fine applied
				cargoValue := 0
				for res, amt := range ship.CargoHold {
					mkt := game.GetMarketEngine()
					if mkt != nil {
						cargoValue += int(mkt.GetSellPrice(res)) * amt
					}
				}
				// Clear cargo
				ship.CargoHold = make(map[string]int)

				fine := cargoValue * 2
				player.Credits -= fine
				if player.Credits < 0 {
					player.Credits = 0
				}

				ss.caughtCount[player.Name]++

				game.LogEvent("event", player.Name,
					fmt.Sprintf("🚨 %s's %s caught smuggling in %s! Cargo seized + %d cr fine (%s)",
						player.Name, ship.Name, sysName, fine, reason))
			} else {
				// Successful smuggle — bonus credits
				bonus := 0
				for _, amt := range ship.CargoHold {
					bonus += amt * 5 // 5cr per unit smuggled
				}
				player.Credits += bonus
				ss.smuggleAttempts[player.Name]++

				if rand.Intn(3) == 0 { // don't spam events
					game.LogEvent("trade", player.Name,
						fmt.Sprintf("🤫 %s's %s successfully ran cargo past patrols in %s (+%d cr bonus)",
							player.Name, ship.Name, sysName, bonus))
				}
			}
		}
	}
}

// GetSmugglingStats returns smuggling stats for a faction.
func (ss *SmugglingSystem) GetSmugglingStats(playerName string) (successes, caught int) {
	if ss.smuggleAttempts == nil {
		return 0, 0
	}
	return ss.smuggleAttempts[playerName], ss.caughtCount[playerName]
}
