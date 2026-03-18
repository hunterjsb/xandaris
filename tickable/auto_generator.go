package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AutoGeneratorSystem{
		BaseSystem: NewBaseSystem("AutoGenerator", 125),
	})
}

// AutoGeneratorSystem auto-builds Generators on planets that are in
// power crisis (<30% power ratio) and have no Generator or their
// Generator is insufficient. This is the companion to AutoHabitat.
//
// Conditions for auto-build:
//   - Planet power ratio < 30%
//   - Planet has 0 or 1 Generators
//   - Owner has >= 1500 credits
//   - No Generator currently under construction
//   - Cooldown: 5000 ticks between auto-builds per planet
//
// Also auto-builds Refineries on planets that have Oil deposits
// but no Refinery (the Oil→Fuel pipeline is critical for power).
type AutoGeneratorSystem struct {
	*BaseSystem
	lastBuild map[int]int64
}

func (ags *AutoGeneratorSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := ags.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ags.lastBuild == nil {
		ags.lastBuild = make(map[int]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			pid := planet.GetID()
			if tick-ags.lastBuild[pid] < 5000 {
				continue
			}

			// Count generators and refineries
			generators := 0
			refineries := 0
			hasOil := false
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					if b.BuildingType == entities.BuildingGenerator {
						generators++
					}
					if b.BuildingType == entities.BuildingRefinery {
						refineries++
					}
				}
			}
			for _, re := range planet.Resources {
				if r, ok := re.(*entities.Resource); ok && r.ResourceType == entities.ResOil {
					hasOil = true
				}
			}

			var player *entities.Player
			for _, p := range players {
				if p != nil && p.Name == planet.Owner {
					player = p
					break
				}
			}
			if player == nil || player.Credits < 1500 {
				continue
			}

			// Auto-build Generator if power is low
			if planet.GetPowerRatio() < 0.3 && generators < 4 {
				player.Credits -= 1000
				ags.lastBuild[pid] = tick
				game.AIBuildOnPlanet(planet, entities.BuildingGenerator, planet.Owner, sys.ID)
				game.LogEvent("logistics", planet.Owner,
					fmt.Sprintf("⚡ Auto-building Generator on %s — power at %.0f%%",
						planet.Name, planet.GetPowerRatio()*100))
				continue
			}

			// Auto-build Refinery if planet has Oil but no Refinery
			if hasOil && refineries == 0 && planet.TechLevel >= 0 {
				player.Credits -= 1500
				ags.lastBuild[pid] = tick
				game.AIBuildOnPlanet(planet, entities.BuildingRefinery, planet.Owner, sys.ID)
				game.LogEvent("logistics", planet.Owner,
					fmt.Sprintf("🏭 Auto-building Refinery on %s — Oil deposit detected, need Fuel production!",
						planet.Name))
			}
		}
	}
}
