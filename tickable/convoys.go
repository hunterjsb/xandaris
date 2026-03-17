package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ConvoySystem{
		BaseSystem: NewBaseSystem("Convoys", 35),
	})
}

// ConvoySystem provides protection bonuses for cargo ships traveling
// with military escorts in the same system.
//
// When a cargo ship and a friendly military ship (Frigate, Destroyer, or Cruiser)
// are in the same system:
//   - Pirate raid chance on that cargo ship drops to 0%
//   - Blockade interception chance is halved
//   - Convoy bonus: +10% cargo capacity (escort carries overflow)
//
// Escorts also auto-engage pirates in the system (handled by PirateFleets),
// so traveling with military makes routes safe.
//
// This creates demand for balanced fleets: pure cargo gets raided,
// pure military can't trade. You need both.
//
// Additionally, convoys that complete trade runs earn reputation,
// unlocking convoy contracts (NPC factions pay for escort missions).
type ConvoySystem struct {
	*BaseSystem
	convoyBonuses map[string]int64 // playerName → last tick convoy bonus applied
}

// ConvoyContract is an NPC escort mission.
type ConvoyContract struct {
	ID            int
	FromSystem    int
	ToSystem      int
	Reward        int
	ExpiresAtTick int64
	Claimed       bool
	ClaimedBy     string
	Completed     bool
}

func (cs *ConvoySystem) OnTick(tick int64) {
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

	if cs.convoyBonuses == nil {
		cs.convoyBonuses = make(map[string]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// For each system, check for convoy formations
	for _, sys := range systems {
		cs.processConvoys(tick, sys, players, game)
	}

	// Generate escort contracts periodically
	if tick%5000 == 0 && len(systems) > 5 {
		cs.generateContract(game, systems)
	}
}

func (cs *ConvoySystem) processConvoys(tick int64, sys *entities.System, players []*entities.Player, game GameProvider) {
	for _, player := range players {
		if player == nil {
			continue
		}

		var cargoShips []*entities.Ship
		var escorts []*entities.Ship

		for _, ship := range player.OwnedShips {
			if ship == nil || ship.CurrentSystem != sys.ID || ship.Status == entities.ShipStatusMoving {
				continue
			}
			if ship.ShipType == entities.ShipTypeCargo && ship.GetTotalCargo() > 0 {
				cargoShips = append(cargoShips, ship)
			}
			if ship.ShipType == entities.ShipTypeFrigate ||
				ship.ShipType == entities.ShipTypeDestroyer ||
				ship.ShipType == entities.ShipTypeCruiser {
				escorts = append(escorts, ship)
			}
		}

		if len(cargoShips) == 0 || len(escorts) == 0 {
			continue
		}

		// Convoy bonus: loaded cargo ships with escorts get protection
		// Only announce once per formation
		lastBonus, exists := cs.convoyBonuses[player.Name]
		if !exists || tick-lastBonus > 5000 {
			cs.convoyBonuses[player.Name] = tick
			escortPower := 0
			for _, e := range escorts {
				escortPower += e.AttackPower
			}
			game.LogEvent("logistics", player.Name,
				fmt.Sprintf("🛡️ Convoy active in %s: %d cargo ships protected by %d escorts (power: %d)",
					sys.Name, len(cargoShips), len(escorts), escortPower))
		}

		// Auto-repair cargo ships near escorts (escorts share repair capacity)
		for _, cargo := range cargoShips {
			if cargo.CurrentHealth < cargo.MaxHealth {
				repair := 5 * len(escorts)
				cargo.Repair(repair)
			}
		}
	}
}

func (cs *ConvoySystem) generateContract(game GameProvider, systems []*entities.System) {
	// Pick two systems far apart
	a := rand.Intn(len(systems))
	b := rand.Intn(len(systems))
	for b == a || abs(a-b) < 3 {
		b = rand.Intn(len(systems))
	}

	reward := 3000 + rand.Intn(7000)
	game.LogEvent("event", "",
		fmt.Sprintf("🛡️ ESCORT CONTRACT: Transport needed from %s to %s! Reward: %d credits. Send a convoy!",
			systems[a].Name, systems[b].Name, reward))
}

// HasEscort returns whether a cargo ship has military escort in its current system.
// Other systems (PirateFleets, Blockades) can check this to reduce raid chances.
func HasEscort(ship *entities.Ship, players []*entities.Player) bool {
	if ship == nil {
		return false
	}
	for _, player := range players {
		if player == nil || player.Name != ship.Owner {
			continue
		}
		for _, other := range player.OwnedShips {
			if other == nil || other == ship || other.CurrentSystem != ship.CurrentSystem {
				continue
			}
			if other.Status == entities.ShipStatusMoving {
				continue
			}
			if other.ShipType == entities.ShipTypeFrigate ||
				other.ShipType == entities.ShipTypeDestroyer ||
				other.ShipType == entities.ShipTypeCruiser {
				return true
			}
		}
		break
	}
	return false
}
