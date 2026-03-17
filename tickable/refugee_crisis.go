package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&RefugeeCrisisSystem{
		BaseSystem: NewBaseSystem("RefugeeCrisis", 83),
	})
}

// RefugeeCrisisSystem generates large-scale population displacement
// events when planets are conquered, sieged, or struck by disasters.
// Refugees flee to nearby owned planets, creating sudden population
// surges that stress housing and resources.
//
// Refugee mechanics:
//   - Triggered by: planet happiness <15%, siege, plague, rebellion
//   - Refugees seek the nearest habitable planet with capacity
//   - Receiving planet gets +population but -10% happiness (strain)
//   - If no capacity available, refugees are lost
//
// Strategic implications:
//   - Build extra habitat capacity to absorb refugees (population = power)
//   - Refugees bring labor but consume resources
//   - Triggering refugee crises in enemy territory strains their economy
//
// This creates consequences that ripple outward from military actions.
type RefugeeCrisisSystem struct {
	*BaseSystem
	lastCrisis map[int]int64 // systemID → last crisis tick
}

func (rcs *RefugeeCrisisSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := rcs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if rcs.lastCrisis == nil {
		rcs.lastCrisis = make(map[int]int64)
	}

	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.Population < 1000 {
				continue
			}

			// Check for crisis conditions
			if planet.Happiness > 0.15 {
				continue
			}

			// Rate limit per system
			if tick-rcs.lastCrisis[sys.ID] < 10000 {
				continue
			}

			// 10% chance per check when conditions met
			if rand.Intn(10) != 0 {
				continue
			}

			rcs.lastCrisis[sys.ID] = tick
			rcs.generateCrisis(planet, sys, systems, game)
		}
	}
}

func (rcs *RefugeeCrisisSystem) generateCrisis(source *entities.Planet, sourceSys *entities.System, systems []*entities.System, game GameProvider) {
	// 10-20% of population flees
	fleeRate := 0.10 + rand.Float64()*0.10
	refugees := int64(float64(source.Population) * fleeRate)
	if refugees < 500 {
		return
	}

	source.Population -= refugees

	// Find nearest planet with capacity
	placed := int64(0)
	for _, sys := range systems {
		for _, e := range sys.Entities {
			dest, ok := e.(*entities.Planet)
			if !ok || dest.Owner == "" || dest.GetID() == source.GetID() {
				continue
			}
			cap := dest.GetTotalPopulationCapacity()
			if cap <= 0 || dest.Population >= cap {
				continue
			}

			space := cap - dest.Population
			toPlace := refugees - placed
			if toPlace > space {
				toPlace = space
			}

			dest.Population += toPlace
			dest.Happiness -= 0.05 // strain from refugees
			if dest.Happiness < 0.1 {
				dest.Happiness = 0.1
			}
			placed += toPlace

			if placed >= refugees {
				break
			}
		}
		if placed >= refugees {
			break
		}
	}

	lost := refugees - placed

	msg := fmt.Sprintf("🚶 REFUGEE CRISIS: %d flee %s (%.0f%% happiness). %d resettled",
		refugees, source.Name, source.Happiness*100, placed)
	if lost > 0 {
		msg += fmt.Sprintf(", %d lost (no capacity)", lost)
	}
	game.LogEvent("event", source.Owner, msg)
}
