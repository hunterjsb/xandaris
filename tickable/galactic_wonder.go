package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticWonderSystem{
		BaseSystem: NewBaseSystem("GalacticWonder", 169),
	})
}

// GalacticWonderSystem generates rare natural wonders that appear
// in systems and provide unique passive bonuses. Unlike monuments
// (built by factions), wonders are discovered and claimed.
//
// Wonders:
//   Crystal Nebula: system where it appears gets +20% Electronics output
//   Eternal Spring: planet never drops below 50% happiness
//   Dark Star: system generates 100MW free power for all planets
//   Living Asteroid: slowly produces Rare Metals (5/interval)
//   Singing Void: +0.1 tech/interval to all planets in system
//
// One wonder spawns every ~30,000 ticks. Controlled by whichever
// faction has the most planets in the system.
type GalacticWonderSystem struct {
	*BaseSystem
	wonders   []*GalacticNaturalWonder
	nextSpawn int64
}

type GalacticNaturalWonder struct {
	Name     string
	SystemID int
	SysName  string
	Effect   string
	Active   bool
}

var wonderDefs = []struct {
	name   string
	effect string
}{
	{"Crystal Nebula", "+20% Electronics output"},
	{"Eternal Spring", "minimum 50% happiness"},
	{"Dark Star", "+100MW free power"},
	{"Living Asteroid", "+5 Rare Metals/interval"},
	{"Singing Void", "+0.1 tech/interval"},
}

func (gws *GalacticWonderSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := gws.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if gws.nextSpawn == 0 {
		gws.nextSpawn = tick + 15000 + int64(rand.Intn(15000))
	}

	systems := game.GetSystems()

	// Apply wonder effects
	for _, w := range gws.wonders {
		if !w.Active {
			continue
		}

		for _, sys := range systems {
			if sys.ID != w.SystemID {
				continue
			}

			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner == "" {
					continue
				}

				switch w.Name {
				case "Eternal Spring":
					if planet.Happiness < 0.5 {
						planet.Happiness = 0.5
					}
				case "Dark Star":
					planet.PowerGenerated += 5 // small per-tick contribution
				case "Living Asteroid":
					if rand.Intn(10) == 0 {
						planet.AddStoredResource(entities.ResRareMetals, 1)
					}
				case "Singing Void":
					planet.TechLevel += 0.001
				case "Crystal Nebula":
					if rand.Intn(10) == 0 {
						planet.AddStoredResource(entities.ResElectronics, 1)
					}
				}
			}
			break
		}
	}

	// Spawn new wonder
	if tick >= gws.nextSpawn {
		gws.nextSpawn = tick + 30000 + int64(rand.Intn(20000))

		// Pick an unspawned wonder
		spawned := make(map[string]bool)
		for _, w := range gws.wonders {
			spawned[w.Name] = true
		}

		var available []struct{ name, effect string }
		for _, d := range wonderDefs {
			if !spawned[d.name] {
				available = append(available, d)
			}
		}
		if len(available) == 0 || len(systems) == 0 {
			return
		}

		def := available[rand.Intn(len(available))]
		sys := systems[rand.Intn(len(systems))]

		gws.wonders = append(gws.wonders, &GalacticNaturalWonder{
			Name:     def.name,
			SystemID: sys.ID,
			SysName:  sys.Name,
			Effect:   def.effect,
			Active:   true,
		})

		game.LogEvent("event", "",
			fmt.Sprintf("✨ NATURAL WONDER DISCOVERED: %s in %s! Effect: %s. The galaxy marvels at this cosmic anomaly!",
				def.name, sys.Name, def.effect))
	}
}
