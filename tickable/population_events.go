package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&PopulationEventSystem{
		BaseSystem: NewBaseSystem("PopulationEvents", 58),
	})
}

// PopulationEventSystem generates interesting population-driven events
// that add flavor and strategic consequences to colony management.
//
// Events:
//   - Baby boom: very happy planet gets burst population growth
//   - Brain drain: low-tech planet loses skilled workers to higher-tech neighbors
//   - Cultural festival: random happiness boost + small credit income
//   - Labor strike: low-happiness planet loses productivity for 1000 ticks
//   - Innovation: high-tech planet discovers efficiency (+5% production)
//   - Epidemic scare: crowded planet temporarily stops growing
//
// These events make population management more dynamic than just
// "build habitats and wait". Each creates a decision point.
type PopulationEventSystem struct {
	*BaseSystem
	nextEvent int64
	strikes   map[int]int64 // planetID → strike end tick
	festivals map[int]int64 // planetID → festival end tick
}

func (pes *PopulationEventSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := pes.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if pes.strikes == nil {
		pes.strikes = make(map[int]int64)
		pes.festivals = make(map[int]int64)
	}

	if pes.nextEvent == 0 {
		pes.nextEvent = tick + 1000 + int64(rand.Intn(3000))
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Process active effects
	for pid, endTick := range pes.strikes {
		if tick >= endTick {
			delete(pes.strikes, pid)
			// Find planet to restore productivity
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if planet, ok := e.(*entities.Planet); ok && planet.GetID() == pid {
						game.LogEvent("event", planet.Owner,
							fmt.Sprintf("✅ Labor strike on %s has ended. Workers returning to their posts", planet.Name))
					}
				}
			}
		}
	}

	// Apply strike effects (reduce productivity)
	for pid := range pes.strikes {
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.GetID() == pid {
					planet.ProductivityBonus *= 0.5 // halve during strike
				}
			}
		}
	}

	// Fire new events
	if tick < pes.nextEvent {
		return
	}
	pes.nextEvent = tick + 2000 + int64(rand.Intn(4000))

	// Collect all owned planets
	var owned []*entities.Planet
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" && planet.Population > 500 {
				owned = append(owned, planet)
			}
		}
	}
	if len(owned) == 0 {
		return
	}

	eventType := rand.Intn(6)
	planet := owned[rand.Intn(len(owned))]

	switch eventType {
	case 0: // Baby boom
		if planet.Happiness < 0.7 {
			return
		}
		cap := planet.GetTotalPopulationCapacity()
		if cap <= 0 || planet.Population >= cap {
			return
		}
		bonus := int64(1000 + rand.Intn(5000))
		if planet.Population+bonus > cap {
			bonus = cap - planet.Population
		}
		planet.Population += bonus
		game.LogEvent("event", planet.Owner,
			fmt.Sprintf("👶 Baby boom on %s! %d new citizens — happiness is contagious!",
				planet.Name, bonus))

	case 1: // Brain drain
		if planet.TechLevel >= 2.0 || planet.Population < 2000 {
			return
		}
		lost := planet.Population / 20 // 5% emigrate
		if lost < 100 {
			return
		}
		planet.Population -= lost
		// Find a high-tech planet to receive them
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if p, ok := e.(*entities.Planet); ok && p.Owner != "" && p.TechLevel >= 2.0 && p.GetID() != planet.GetID() {
					cap := p.GetTotalPopulationCapacity()
					if cap > 0 && p.Population < cap {
						added := lost
						if p.Population+added > cap {
							added = cap - p.Population
						}
						p.Population += added
					}
					break
				}
			}
		}
		game.LogEvent("event", planet.Owner,
			fmt.Sprintf("🧠 Brain drain on %s! %d skilled workers emigrated to higher-tech colonies. Invest in Electronics!",
				planet.Name, lost))

	case 2: // Cultural festival
		credits := 200 + rand.Intn(800)
		planet.Happiness += 0.05
		if planet.Happiness > 1.0 {
			planet.Happiness = 1.0
		}
		for _, p := range players {
			if p != nil && p.Name == planet.Owner {
				p.Credits += credits
				break
			}
		}
		game.LogEvent("event", planet.Owner,
			fmt.Sprintf("🎉 Cultural festival on %s! +%.0f%% happiness, +%d credits from tourism",
				planet.Name, 5.0, credits))

	case 3: // Labor strike
		if planet.Happiness >= 0.4 {
			return // only unhappy planets strike
		}
		if _, striking := pes.strikes[planet.GetID()]; striking {
			return
		}
		pes.strikes[planet.GetID()] = tick + 2000 + int64(rand.Intn(2000))
		game.LogEvent("alert", planet.Owner,
			fmt.Sprintf("✊ LABOR STRIKE on %s! Workers demand better conditions. Production halved until resolved (improve happiness!)",
				planet.Name))

	case 4: // Innovation
		if planet.TechLevel < 2.0 {
			return
		}
		bonus := 0.1 + rand.Float64()*0.2
		planet.TechLevel += bonus
		game.LogEvent("event", planet.Owner,
			fmt.Sprintf("💡 Innovation on %s! Scientists made a breakthrough — tech level +%.1f (now %.1f)",
				planet.Name, bonus, planet.TechLevel))

	case 5: // Crowding stress
		cap := planet.GetTotalPopulationCapacity()
		if cap <= 0 || float64(planet.Population)/float64(cap) < 0.9 {
			return // only fires when near capacity
		}
		planet.Happiness -= 0.1
		if planet.Happiness < 0.1 {
			planet.Happiness = 0.1
		}
		game.LogEvent("event", planet.Owner,
			fmt.Sprintf("🏠 Overcrowding on %s! %d/%d population capacity — build more Habitats!",
				planet.Name, planet.Population, cap))
	}
}
