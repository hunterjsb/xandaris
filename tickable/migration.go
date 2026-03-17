package tickable

import (
	"fmt"
	"math"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&MigrationSystem{
		BaseSystem: NewBaseSystem("Migration", 12),
	})
}

// MigrationSystem moves population between planets within the same faction.
// People migrate FROM unhappy/overcrowded planets TO happy/underpopulated ones.
//
// This creates dynamic population flows:
// - Build a great colony (high happiness, good resources) → people move there
// - Neglect a planet (no water, no power) → people leave
// - Overcrowded planets bleed population to newer colonies
//
// Migration also happens BETWEEN factions when happiness is extremely low:
// - Happiness < 20% → 0.1% emigration per interval to the nearest happy faction
// - This creates real consequences for neglecting your citizens
type MigrationSystem struct {
	*BaseSystem
}

func (ms *MigrationSystem) OnTick(tick int64) {
	// Run every 100 ticks (~10 seconds)
	if tick%100 != 0 {
		return
	}

	ctx := ms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil || len(player.OwnedPlanets) < 2 {
			continue
		}
		ms.processInternalMigration(player, systems)
	}

	// Cross-faction emigration from extremely unhappy planets
	ms.processEmigration(players, systems)
}

// processInternalMigration moves population between a player's own planets.
// People flow from low-happiness to high-happiness planets.
func (ms *MigrationSystem) processInternalMigration(player *entities.Player, systems []*entities.System) {
	type planetScore struct {
		planet    *entities.Planet
		score     float64 // attractiveness: happiness * (1 - pop/capacity)
		canLose   bool    // has enough pop to lose some
		canGain   bool    // has room for more
	}

	var scored []planetScore
	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner != player.Name || planet.Population <= 0 {
				continue
			}

			capacity := planet.GetTotalPopulationCapacity()
			if capacity <= 0 {
				capacity = 1
			}
			fillRatio := float64(planet.Population) / float64(capacity)
			openness := 1.0 - fillRatio
			if openness < 0 {
				openness = 0
			}

			// Score: happy + open space = attractive. Unhappy + overcrowded = repulsive.
			score := planet.Happiness * (0.3 + 0.7*openness)

			scored = append(scored, planetScore{
				planet:  planet,
				score:   score,
				canLose: planet.Population > 1000, // don't depopulate tiny colonies
				canGain: fillRatio < 0.95,
			})
		}
	}

	if len(scored) < 2 {
		return
	}

	// Find worst and best planets
	var worst, best *planetScore
	for i := range scored {
		if scored[i].canLose {
			if worst == nil || scored[i].score < worst.score {
				worst = &scored[i]
			}
		}
		if scored[i].canGain {
			if best == nil || scored[i].score > best.score {
				best = &scored[i]
			}
		}
	}

	if worst == nil || best == nil || worst == best {
		return
	}

	// Only migrate if there's a meaningful happiness gap
	gap := best.score - worst.score
	if gap < 0.15 {
		return // not worth migrating for small differences
	}

	// Migrate: 0.5% of worst planet's pop, scaled by gap
	migrants := int64(math.Round(float64(worst.planet.Population) * 0.005 * gap))
	if migrants < 1 {
		migrants = 1
	}
	if migrants > 100 {
		migrants = 100 // cap per interval
	}

	worst.planet.Population -= migrants
	best.planet.Population += migrants

	if migrants > 10 {
		fmt.Printf("[Migration] %d people moved from %s (%.0f%% happy) to %s (%.0f%% happy) [%s]\n",
			migrants, worst.planet.Name, worst.planet.Happiness*100,
			best.planet.Name, best.planet.Happiness*100, player.Name)
	}
}

// processEmigration handles people leaving extremely unhappy factions.
// If a planet's happiness is < 20%, people emigrate to other factions' planets
// in the same system. This creates real consequences for neglect.
func (ms *MigrationSystem) processEmigration(players []*entities.Player, systems []*entities.System) {
	for _, sys := range systems {
		// Find all inhabited planets in this system
		type planetInfo struct {
			planet *entities.Planet
			owner  string
		}
		var planets []planetInfo
		for _, e := range sys.Entities {
			if p, ok := e.(*entities.Planet); ok && p.Owner != "" && p.Population > 0 {
				planets = append(planets, planetInfo{planet: p, owner: p.Owner})
			}
		}

		if len(planets) < 2 {
			continue
		}

		for _, source := range planets {
			if source.planet.Happiness >= 0.20 || source.planet.Population <= 500 {
				continue // not desperate enough to emigrate
			}

			// Find the happiest planet in this system owned by someone else
			var bestDest *entities.Planet
			bestHappy := 0.0
			for _, dest := range planets {
				if dest.owner == source.owner {
					continue
				}
				cap := dest.planet.GetTotalPopulationCapacity()
				if cap > 0 && dest.planet.Population < cap && dest.planet.Happiness > bestHappy {
					bestHappy = dest.planet.Happiness
					bestDest = dest.planet
				}
			}

			if bestDest == nil || bestHappy < 0.30 {
				continue // nowhere better to go
			}

			// Emigrate: 0.1% of population
			emigrants := int64(math.Round(float64(source.planet.Population) * 0.001))
			if emigrants < 1 {
				emigrants = 1
			}
			if emigrants > 50 {
				emigrants = 50
			}

			source.planet.Population -= emigrants
			bestDest.Population += emigrants

			if emigrants > 5 {
				fmt.Printf("[Emigration] %d people left %s (%s, %.0f%% happy) for %s (%s, %.0f%% happy)\n",
					emigrants, source.planet.Name, source.owner, source.planet.Happiness*100,
					bestDest.Name, bestDest.Owner, bestDest.Happiness*100)
			}
		}
	}
}
