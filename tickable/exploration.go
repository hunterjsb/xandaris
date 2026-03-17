package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ExplorationSystem{
		BaseSystem: NewBaseSystem("Exploration", 33),
	})
}

// ExplorationSystem lets Scout ships discover anomalies in systems they visit.
// Anomalies provide one-time bonuses when discovered:
//
// - Ancient Ruins: +5000 credits (salvage technology)
// - Derelict Ship: free resources loaded into scout's cargo
// - Mineral Vein: nearby planet gets a new resource deposit
// - Wormhole: creates a temporary hyperlane to a distant system
// - Data Cache: +0.5 tech level for the owner's nearest planet
// - Void Crystal: rare luxury resource worth 10,000 credits
//
// Each system can only be explored once. Scout ships in unexplored systems
// have a chance to discover something each tick.
type ExplorationSystem struct {
	*BaseSystem
	explored map[int]bool // systems already explored
}

func (es *ExplorationSystem) OnTick(tick int64) {
	if tick%50 != 0 {
		return
	}

	ctx := es.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if es.explored == nil {
		es.explored = make(map[int]bool)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeScout {
				continue
			}
			if ship.Status == entities.ShipStatusMoving {
				continue
			}
			if es.explored[ship.CurrentSystem] {
				continue
			}

			// 10% chance per tick to discover something
			if rand.Intn(10) != 0 {
				continue
			}

			es.explored[ship.CurrentSystem] = true
			es.discover(game, player, ship, systems)
		}
	}
}

func (es *ExplorationSystem) discover(game GameProvider, player *entities.Player, ship *entities.Ship, systems []*entities.System) {
	discoveryType := rand.Intn(6)

	switch discoveryType {
	case 0: // Ancient Ruins
		credits := 3000 + rand.Intn(7000)
		player.Credits += credits
		game.LogEvent("explore", player.Name,
			fmt.Sprintf("🏛️ %s discovered Ancient Ruins in SYS-%d! +%d credits from salvaged tech",
				ship.Name, ship.CurrentSystem+1, credits))

	case 1: // Derelict Ship
		resources := []string{entities.ResRareMetals, entities.ResElectronics, entities.ResFuel}
		res := resources[rand.Intn(len(resources))]
		qty := 50 + rand.Intn(200)
		loaded := ship.AddCargo(res, qty)
		if loaded > 0 {
			game.LogEvent("explore", player.Name,
				fmt.Sprintf("🚢 %s found a Derelict Ship! Salvaged %d %s",
					ship.Name, loaded, res))
		}

	case 2: // Mineral Vein — boost a planet's deposit
		for _, sys := range systems {
			if sys.ID != ship.CurrentSystem {
				continue
			}
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
					if len(planet.Resources) > 0 {
						idx := rand.Intn(len(planet.Resources))
						if res, ok := planet.Resources[idx].(*entities.Resource); ok {
							bonus := 15 + rand.Intn(25)
							res.Abundance += bonus
							game.LogEvent("explore", player.Name,
								fmt.Sprintf("⛏️ %s discovered a rich mineral vein! %s on %s +%d abundance",
									ship.Name, res.ResourceType, planet.Name, bonus))
						}
					}
					return
				}
			}
			break
		}

	case 3: // Wormhole — nothing for now, just a lore event
		targetSys := rand.Intn(len(systems))
		game.LogEvent("explore", player.Name,
			fmt.Sprintf("🌀 %s detected an unstable wormhole in SYS-%d! It leads toward SYS-%d but is too dangerous to enter... for now",
				ship.Name, ship.CurrentSystem+1, targetSys+1))

	case 4: // Data Cache — tech boost
		for _, sys := range systems {
			if sys.ID != ship.CurrentSystem {
				continue
			}
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
					planet.TechLevel += 0.3 + rand.Float64()*0.4
					game.LogEvent("explore", player.Name,
						fmt.Sprintf("💾 %s recovered an alien data cache! %s tech level boosted to %.1f",
							ship.Name, planet.Name, planet.TechLevel))
					return
				}
			}
			break
		}

	case 5: // Void Crystal — instant credits
		credits := 5000 + rand.Intn(10000)
		player.Credits += credits
		game.LogEvent("explore", player.Name,
			fmt.Sprintf("💎 %s found a Void Crystal worth %d credits!",
				ship.Name, credits))
	}
}
