package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&FleetCombatSystem{
		BaseSystem: NewBaseSystem("FleetCombat", 37),
	})
}

// FleetCombatSystem resolves combat when hostile factions' military ships
// occupy the same system. Combat is automatic between Hostile factions.
//
// Combat resolution:
//   - Each ship contributes attack power based on type
//   - Damage distributed randomly across enemy ships
//   - Destroyed ships are removed; damaged ships lose health
//   - Victor captures a percentage of destroyed ships' cargo
//
// Ship combat stats:
//   Scout:     5 atk, 50 hp (weak but fast)
//   Frigate:   15 atk, 120 hp (anti-pirate, escort)
//   Destroyer: 30 atk, 200 hp (main combat ship)
//   Cruiser:   50 atk, 350 hp (capital ship)
//   Cargo:     2 atk, 80 hp (vulnerable, needs escort)
type FleetCombatSystem struct {
	*BaseSystem
}

func (fcs *FleetCombatSystem) OnTick(tick int64) {
	// Combat resolves every 200 ticks (~20 seconds)
	if tick%200 != 0 {
		return
	}

	ctx := fcs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	dm := game.GetDiplomacyManager()
	if dm == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// For each system, check for hostile factions with military ships
	for _, sys := range systems {
		fcs.resolveSystemCombat(sys, players, dm, game)
	}
}

type factionFleet struct {
	player *entities.Player
	ships  []*entities.Ship
	power  int
}

func (fcs *FleetCombatSystem) resolveSystemCombat(sys *entities.System, players []*entities.Player, dm interface{ GetRelation(a, b string) int }, game GameProvider) {
	// Build fleet presence per faction in this system
	fleets := make(map[string]*factionFleet)
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.CurrentSystem != sys.ID || ship.Status == entities.ShipStatusMoving {
				continue
			}
			// Only military ships participate in combat
			if ship.ShipType != entities.ShipTypeFrigate &&
				ship.ShipType != entities.ShipTypeDestroyer &&
				ship.ShipType != entities.ShipTypeCruiser {
				continue
			}

			if fleets[player.Name] == nil {
				fleets[player.Name] = &factionFleet{player: player}
			}
			fleets[player.Name].ships = append(fleets[player.Name].ships, ship)
			fleets[player.Name].power += ship.AttackPower
		}
	}

	if len(fleets) < 2 {
		return
	}

	// Check for hostile pairs
	factionNames := make([]string, 0, len(fleets))
	for name := range fleets {
		factionNames = append(factionNames, name)
	}

	for i := 0; i < len(factionNames); i++ {
		for j := i + 1; j < len(factionNames); j++ {
			a, b := factionNames[i], factionNames[j]
			relation := dm.GetRelation(a, b)
			if relation > -2 {
				continue // only Hostile factions fight
			}

			// COMBAT!
			fleetA := fleets[a]
			fleetB := fleets[b]
			fcs.resolveBattle(fleetA, fleetB, sys, game)
			return // one battle per system per tick
		}
	}
}

func (fcs *FleetCombatSystem) resolveBattle(attacker, defender *factionFleet, sys *entities.System, game GameProvider) {
	// Each side deals damage based on total attack power
	// Damage is distributed randomly across enemy ships

	// Attacker strikes
	fcs.dealDamage(attacker, defender)
	// Defender strikes back
	fcs.dealDamage(defender, attacker)

	// Remove destroyed ships
	aLost := fcs.removeDestroyed(attacker)
	dLost := fcs.removeDestroyed(defender)

	if aLost > 0 || dLost > 0 {
		game.LogEvent("combat", attacker.player.Name,
			fmt.Sprintf("⚔️ Battle in %s! %s lost %d ships, %s lost %d ships",
				sys.Name, attacker.player.Name, aLost, defender.player.Name, dLost))

		// Losing ships degrades relations further
		if aLost > 0 {
			game.LogEvent("combat", defender.player.Name,
				fmt.Sprintf("⚔️ %s destroyed %d of %s's ships in %s!",
					defender.player.Name, aLost, attacker.player.Name, sys.Name))
		}
	}
}

func (fcs *FleetCombatSystem) dealDamage(attacker, defender *factionFleet) {
	totalDamage := 0
	for _, ship := range attacker.ships {
		totalDamage += ship.AttackPower
	}

	// Add some randomness (80-120% of base damage)
	totalDamage = int(float64(totalDamage) * (0.8 + rand.Float64()*0.4))

	// Distribute damage across defender's ships
	for totalDamage > 0 && len(defender.ships) > 0 {
		target := defender.ships[rand.Intn(len(defender.ships))]
		dmg := totalDamage
		if dmg > target.AttackPower*2 {
			dmg = target.AttackPower * 2 // don't overkill a single ship
		}
		target.CurrentHealth -= dmg
		totalDamage -= dmg
	}
}

func (fcs *FleetCombatSystem) removeDestroyed(fleet *factionFleet) int {
	destroyed := 0
	alive := make([]*entities.Ship, 0, len(fleet.ships))
	for _, ship := range fleet.ships {
		if ship.CurrentHealth <= 0 {
			destroyed++
			// Remove from player's ship list
			for i, s := range fleet.player.OwnedShips {
				if s == ship {
					fleet.player.OwnedShips = append(fleet.player.OwnedShips[:i], fleet.player.OwnedShips[i+1:]...)
					break
				}
			}
		} else {
			alive = append(alive, ship)
		}
	}
	fleet.ships = alive
	return destroyed
}
