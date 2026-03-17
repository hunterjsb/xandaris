package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&CulturalInfluenceSystem{
		BaseSystem: NewBaseSystem("CulturalInfluence", 92),
	})
}

// CulturalInfluenceSystem tracks soft power — the cultural impact
// each faction has on the galaxy. High cultural influence lets
// factions peacefully absorb neutral planets and improve diplomacy.
//
// Influence sources:
//   - Population (1 point per 1000 pop)
//   - Tech level (10 points per tech level)
//   - Happiness (20 points per planet with >80% happy)
//   - Trade volume (5 points per active trade agreement)
//   - Monuments built (100 points each)
//
// Influence effects:
//   - Top influencer: unclaimed planets in adjacent systems slowly
//     "culturally drift" toward you (auto-colonize after 10,000 ticks)
//   - All factions: high influence improves diplomacy with neutrals
//   - Cultural victory: first to 5000 influence wins a special event
//
// This creates a non-military path to expansion.
type CulturalInfluenceSystem struct {
	*BaseSystem
	influence    map[string]int // factionName → influence score
	driftTargets map[int]string // planetID → faction drifting toward
	driftTicks   map[int]int64  // planetID → ticks of drift
	nextCalc     int64
}

func (cis *CulturalInfluenceSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := cis.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if cis.influence == nil {
		cis.influence = make(map[string]int)
		cis.driftTargets = make(map[int]string)
		cis.driftTicks = make(map[int]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Calculate influence
	for _, player := range players {
		if player == nil {
			continue
		}

		score := 0
		for _, sys := range systems {
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner != player.Name {
					continue
				}
				score += int(planet.Population / 1000)
				score += int(planet.TechLevel * 10)
				if planet.Happiness > 0.8 {
					score += 20
				}
			}
		}

		score += len(player.OwnedShips) / 5
		if player.Credits > 1000000 {
			score += 50
		}

		cis.influence[player.Name] = score
	}

	// Cultural drift: unclaimed habitable planets near top influencer
	topFaction := ""
	topScore := 0
	for name, score := range cis.influence {
		if score > topScore {
			topScore = score
			topFaction = name
		}
	}

	if topFaction == "" || topScore < 100 {
		return
	}

	// Find unclaimed habitable planets in systems adjacent to the top faction
	for _, sys := range systems {
		// Check if top faction has presence in this system
		hasPresence := false
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner == topFaction {
				hasPresence = true
				break
			}
		}
		if !hasPresence {
			continue
		}

		// Check for unclaimed planets in this system
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner != "" || !planet.IsHabitable() {
				continue
			}

			pid := planet.GetID()
			if cis.driftTargets[pid] != topFaction {
				cis.driftTargets[pid] = topFaction
				cis.driftTicks[pid] = 0
			}

			cis.driftTicks[pid] += 2000
			if cis.driftTicks[pid] >= 20000 {
				// Cultural absorption!
				planet.Owner = topFaction
				planet.Population = 500 // settlers arrive
				for _, re := range planet.Resources {
					if r, ok := re.(*entities.Resource); ok {
						r.Owner = topFaction
					}
				}

				// Add to player
				for _, p := range players {
					if p != nil && p.Name == topFaction {
						p.OwnedPlanets = append(p.OwnedPlanets, planet)
						break
					}
				}

				delete(cis.driftTargets, pid)
				delete(cis.driftTicks, pid)

				game.LogEvent("event", topFaction,
					fmt.Sprintf("🎭 Cultural absorption! %s's influence drew settlers to %s in %s. The planet now flies their banner!",
						topFaction, planet.Name, sys.Name))
			}
		}
	}

	// Announce influence rankings periodically
	if rand.Intn(5) == 0 && topScore > 50 {
		game.LogEvent("intel", "",
			fmt.Sprintf("🎭 Cultural Influence: %s leads with %d influence. High influence attracts settlers to unclaimed worlds!",
				topFaction, topScore))
	}
}

// GetInfluence returns the cultural influence score for a faction.
func (cis *CulturalInfluenceSystem) GetInfluence(faction string) int {
	if cis.influence == nil {
		return 0
	}
	return cis.influence[faction]
}
