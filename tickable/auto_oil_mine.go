package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AutoOilMineSystem{
		BaseSystem: NewBaseSystem("AutoOilMine", 140),
	})
}

// AutoOilMineSystem addresses the root cause of power crises: planets
// need Oil to make Fuel, but many planets with Oil deposits don't
// have Mines built on them. This auto-builds Mines on Oil deposits.
//
// Conditions:
//   - Planet has an Oil resource deposit with no Mine attached
//   - Planet has power crisis (<50% power ratio)
//   - Owner has >= 1000cr
//   - Max 1 auto-mine per faction per 5000 ticks
//
// This closes the production loop: Oil deposit → Mine → Oil → Refinery → Fuel → Generator → Power
type AutoOilMineSystem struct {
	*BaseSystem
	lastBuild map[string]int64
}

func (aoms *AutoOilMineSystem) OnTick(tick int64) {
	if tick%3000 != 0 {
		return
	}

	ctx := aoms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if aoms.lastBuild == nil {
		aoms.lastBuild = make(map[string]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			if tick-aoms.lastBuild[planet.Owner] < 5000 {
				continue
			}

			// Only if in power trouble
			if planet.GetPowerRatio() > 0.5 {
				continue
			}

			// Find unmined Oil deposit
			for _, re := range planet.Resources {
				r, ok := re.(*entities.Resource)
				if !ok || r.ResourceType != entities.ResOil {
					continue
				}

				// Check if this deposit already has a mine
				hasMine := false
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingMine && b.ResourceNodeID == r.GetID() {
						hasMine = true
						break
					}
				}
				if hasMine {
					continue
				}

				// Find player and check credits
				for _, p := range players {
					if p == nil || p.Name != planet.Owner || p.Credits < 1000 {
						continue
					}

					p.Credits -= 800
					aoms.lastBuild[p.Name] = tick
					game.AIBuildOnPlanet(planet, entities.BuildingMine, planet.Owner, sys.ID)

					game.LogEvent("logistics", planet.Owner,
						fmt.Sprintf("⛏️ Auto-building Mine on %s's Oil deposit — power crisis needs Fuel production chain!",
							planet.Name))
					return
				}
			}
		}
	}
}
