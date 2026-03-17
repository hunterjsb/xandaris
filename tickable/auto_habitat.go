package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AutoHabitatSystem{
		BaseSystem: NewBaseSystem("AutoHabitat", 122),
	})
}

// AutoHabitatSystem auto-builds Habitats on planets where population
// is at 90%+ capacity and the owner has enough credits. This prevents
// the labor shortage alerts from flooding the event log.
//
// Conditions for auto-build:
//   - Planet population >= 90% of housing capacity
//   - Owner has >= 2000 credits
//   - Planet has < 5 Habitats already (don't over-build)
//   - No Habitat currently under construction
//
// Uses the game's AIBuildOnPlanet method so construction goes through
// normal building pipeline with construction time.
type AutoHabitatSystem struct {
	*BaseSystem
	lastBuild map[int]int64 // planetID → last auto-build tick
}

func (ahs *AutoHabitatSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := ahs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ahs.lastBuild == nil {
		ahs.lastBuild = make(map[int]int64)
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
			if tick-ahs.lastBuild[pid] < 5000 {
				continue
			}

			cap := planet.GetTotalPopulationCapacity()
			if cap <= 0 || float64(planet.Population)/float64(cap) < 0.90 {
				continue
			}

			// Count existing habitats
			habitatCount := 0
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingHabitat {
					habitatCount++
				}
			}
			if habitatCount >= 5 {
				continue
			}

			// Check owner credits
			for _, p := range players {
				if p == nil || p.Name != planet.Owner {
					continue
				}
				if p.Credits < 2000 {
					continue
				}

				// Build habitat
				p.Credits -= 1500 // habitat cost
				ahs.lastBuild[pid] = tick
				game.AIBuildOnPlanet(planet, entities.BuildingHabitat, planet.Owner, sys.ID)

				game.LogEvent("logistics", planet.Owner,
					fmt.Sprintf("🏠 Auto-building Habitat on %s — population at %d/%d (%.0f%% capacity)",
						planet.Name, planet.Population, cap,
						float64(planet.Population)/float64(cap)*100))
				break
			}
		}
	}
}
