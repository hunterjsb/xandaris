package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TreasureFleetSystem{
		BaseSystem: NewBaseSystem("TreasureFleets", 85),
	})
}

// TreasureFleetSystem spawns NPC treasure fleets that travel through
// the galaxy carrying enormous wealth. Factions can intercept them
// with military ships to seize the treasure.
//
// Treasure fleet:
//   - Appears in a random system, heading toward another
//   - Carries 10,000-50,000 credits worth of goods
//   - Protected by 2-4 escort ships (virtual, not entity ships)
//   - Moves every 2000 ticks (crosses ~1 system)
//   - Intercepted by having 3+ military ships in its current system
//
// Interception:
//   - 60% success chance with 3 military ships
//   - 80% with 5+
//   - 95% with 8+
//   - Failed interception = fleet escapes, no loot
//   - Success = credits + resources divided among interceptor's faction
//
// This creates a "heist" mechanic: spot the fleet, mobilize forces,
// intercept before it reaches its destination. Multiple factions
// racing to intercept creates emergent PvP competition.
type TreasureFleetSystem struct {
	*BaseSystem
	fleets    []*TreasureFleet
	nextFleet int64
}

// TreasureFleet represents a traveling NPC treasure convoy.
type TreasureFleet struct {
	CurrentSystem int
	DestSystem    int
	SysName       string
	Value         int
	Escorts       int // virtual escort count
	TicksInSystem int
	Active        bool
	Intercepted   bool
}

func (tfs *TreasureFleetSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := tfs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tfs.nextFleet == 0 {
		tfs.nextFleet = tick + 8000 + int64(rand.Intn(10000))
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Process active fleets
	for _, fleet := range tfs.fleets {
		if !fleet.Active {
			continue
		}

		fleet.TicksInSystem += 500

		// Check for interception
		tfs.checkInterception(fleet, players, systems, game)
		if fleet.Intercepted {
			continue
		}

		// Move fleet every 2000 ticks
		if fleet.TicksInSystem >= 2000 {
			fleet.TicksInSystem = 0
			// Move toward destination
			connected := game.GetConnectedSystems(fleet.CurrentSystem)
			if len(connected) == 0 {
				fleet.Active = false
				continue
			}

			// Pick the connected system closest to destination
			// Simple: just pick a random connected system
			fleet.CurrentSystem = connected[rand.Intn(len(connected))]

			// Update name
			fleet.SysName = fmt.Sprintf("SYS-%d", fleet.CurrentSystem+1)
			for _, sys := range systems {
				if sys.ID == fleet.CurrentSystem {
					fleet.SysName = sys.Name
					break
				}
			}

			// Check if reached destination
			if fleet.CurrentSystem == fleet.DestSystem {
				fleet.Active = false
				game.LogEvent("event", "",
					fmt.Sprintf("💰 Treasure fleet reached its destination safely. %dcr in goods delivered — opportunity missed!",
						fleet.Value))
				continue
			}

			game.LogEvent("event", "",
				fmt.Sprintf("💰 Treasure fleet spotted passing through %s! Value: %dcr, %d escorts. Send military to intercept!",
					fleet.SysName, fleet.Value, fleet.Escorts))
		}
	}

	// Spawn new fleet
	if tick >= tfs.nextFleet {
		tfs.nextFleet = tick + 15000 + int64(rand.Intn(15000))

		activeCount := 0
		for _, f := range tfs.fleets {
			if f.Active {
				activeCount++
			}
		}
		if activeCount < 2 {
			tfs.spawnFleet(game, systems)
		}
	}
}

func (tfs *TreasureFleetSystem) checkInterception(fleet *TreasureFleet, players []*entities.Player, systems []*entities.System, game GameProvider) {
	// Count military ships per faction in fleet's system
	type factionForce struct {
		player *entities.Player
		ships  int
	}
	forces := make(map[string]*factionForce)

	for _, p := range players {
		if p == nil {
			continue
		}
		count := 0
		for _, ship := range p.OwnedShips {
			if ship == nil || ship.CurrentSystem != fleet.CurrentSystem {
				continue
			}
			if ship.ShipType == entities.ShipTypeFrigate ||
				ship.ShipType == entities.ShipTypeDestroyer ||
				ship.ShipType == entities.ShipTypeCruiser {
				count++
			}
		}
		if count >= 3 {
			forces[p.Name] = &factionForce{p, count}
		}
	}

	if len(forces) == 0 {
		return
	}

	// Find faction with most military ships
	var interceptor *factionForce
	for _, f := range forces {
		if interceptor == nil || f.ships > interceptor.ships {
			interceptor = f
		}
	}

	// Success chance based on ship count vs escorts
	successChance := 40 + interceptor.ships*10
	if successChance > 95 {
		successChance = 95
	}

	if rand.Intn(100) >= successChance {
		// Failed interception
		game.LogEvent("event", interceptor.player.Name,
			fmt.Sprintf("💰 %s attempted to intercept the treasure fleet in %s but the escorts fought them off!",
				interceptor.player.Name, fleet.SysName))
		return
	}

	// Success!
	fleet.Intercepted = true
	fleet.Active = false
	interceptor.player.Credits += fleet.Value

	game.LogEvent("event", interceptor.player.Name,
		fmt.Sprintf("💰 %s intercepted the treasure fleet in %s! Seized %dcr in goods! (%d military ships vs %d escorts)",
			interceptor.player.Name, fleet.SysName, fleet.Value, interceptor.ships, fleet.Escorts))
}

func (tfs *TreasureFleetSystem) spawnFleet(game GameProvider, systems []*entities.System) {
	if len(systems) < 5 {
		return
	}

	a := rand.Intn(len(systems))
	b := rand.Intn(len(systems))
	for b == a {
		b = rand.Intn(len(systems))
	}

	value := 10000 + rand.Intn(40000)
	escorts := 2 + rand.Intn(3)

	fleet := &TreasureFleet{
		CurrentSystem: systems[a].ID,
		DestSystem:    systems[b].ID,
		SysName:       systems[a].Name,
		Value:         value,
		Escorts:       escorts,
		Active:        true,
	}
	tfs.fleets = append(tfs.fleets, fleet)

	game.LogEvent("event", "",
		fmt.Sprintf("💰 TREASURE FLEET departed from %s heading to %s! Carrying %dcr in goods with %d escorts. Intercept with 3+ warships!",
			systems[a].Name, systems[b].Name, value, escorts))
}
