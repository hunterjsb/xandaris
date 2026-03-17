package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradeDeficitAlertSystem{
		BaseSystem: NewBaseSystem("TradeDeficitAlert", 147),
	})
}

// TradeDeficitAlertSystem monitors resource imports vs local production
// and warns when a faction is dangerously dependent on imports for
// a critical resource.
//
// For each critical resource (Fuel, Water, Iron):
//   - If stored amount < 20 AND no local production source → "import dependent"
//   - If import dependent for 2+ critical resources → "trade deficit crisis"
//
// This differs from the resource forecast (which predicts depletion timing).
// This identifies STRUCTURAL dependency: you don't produce it at all,
// you completely rely on trade/imports. One blockade or trade disruption
// and you're done.
type TradeDeficitAlertSystem struct {
	*BaseSystem
	lastAlert map[string]int64
}

func (tdas *TradeDeficitAlertSystem) OnTick(tick int64) {
	if tick%5000 != 0 {
		return
	}

	ctx := tdas.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tdas.lastAlert == nil {
		tdas.lastAlert = make(map[string]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}
		if tick-tdas.lastAlert[player.Name] < 15000 {
			continue
		}

		// Check what this faction produces vs needs
		produces := make(map[string]bool)
		criticalLow := make(map[string]bool)

		for _, sys := range systems {
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner != player.Name {
					continue
				}

				// What resources does this planet produce?
				for _, re := range planet.Resources {
					if r, ok := re.(*entities.Resource); ok {
						for _, be := range planet.Buildings {
							if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingMine && b.IsOperational && b.ResourceNodeID == r.GetID() {
								produces[r.ResourceType] = true
							}
						}
					}
				}

				// Check for Refinery (produces Fuel from Oil)
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingRefinery && b.IsOperational {
						produces[entities.ResFuel] = true
					}
				}

				// Check critical resource levels
				for _, res := range []string{entities.ResFuel, entities.ResWater, entities.ResIron} {
					if planet.GetStoredAmount(res) < 20 {
						criticalLow[res] = true
					}
				}
			}
		}

		// Find structural dependencies
		var dependencies []string
		for _, res := range []string{entities.ResFuel, entities.ResWater, entities.ResIron} {
			if criticalLow[res] && !produces[res] {
				dependencies = append(dependencies, res)
			}
		}

		if len(dependencies) >= 2 {
			tdas.lastAlert[player.Name] = tick
			game.LogEvent("alert", player.Name,
				fmt.Sprintf("🚨 %s: STRUCTURAL TRADE DEFICIT! No local production of: %v. You depend entirely on imports. Build Mines + Refineries!",
					player.Name, dependencies))
		} else if len(dependencies) == 1 {
			tdas.lastAlert[player.Name] = tick
			game.LogEvent("logistics", player.Name,
				fmt.Sprintf("⚠️ %s: import-dependent on %s — no local production. Consider building infrastructure!",
					player.Name, dependencies[0]))
		}
	}
}
