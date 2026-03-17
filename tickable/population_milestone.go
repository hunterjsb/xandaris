package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&PopulationMilestoneSystem{
		BaseSystem: NewBaseSystem("PopulationMilestones", 106),
	})
}

// PopulationMilestoneSystem celebrates population growth achievements
// and grants bonuses for reaching milestones.
//
// Planet milestones:
//   1,000 pop:  "Settlement" — +100cr bonus
//   5,000 pop:  "Town" — +500cr bonus
//   10,000 pop: "City" — +1000cr + 0.1 tech bonus
//   25,000 pop: "Metropolis" — +2500cr + 0.2 tech bonus
//   50,000 pop: "Megacity" — +5000cr + 0.3 tech + happiness boost
//   100,000 pop: "Ecumenopolis" — +10000cr + galactic announcement
//
// Faction-wide milestones:
//   10,000 total pop: "Growing Empire"
//   50,000 total pop: "Major Power"
//   100,000 total pop: "Superpower"
//
// Milestones only fire once per planet (tracked).
type PopulationMilestoneSystem struct {
	*BaseSystem
	planetMilestones  map[int]int64   // planetID → highest milestone reached
	factionMilestones map[string]int64 // factionName → highest total pop milestone
}

var popMilestones = []struct {
	threshold int64
	title     string
	credits   int
	techBonus float64
}{
	{100000, "Ecumenopolis", 10000, 0.5},
	{50000, "Megacity", 5000, 0.3},
	{25000, "Metropolis", 2500, 0.2},
	{10000, "City", 1000, 0.1},
	{5000, "Town", 500, 0},
	{1000, "Settlement", 100, 0},
}

func (pms *PopulationMilestoneSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := pms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if pms.planetMilestones == nil {
		pms.planetMilestones = make(map[int]int64)
		pms.factionMilestones = make(map[string]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Track faction totals
	factionPop := make(map[string]int64)

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			factionPop[planet.Owner] += planet.Population
			pid := planet.GetID()
			currentMilestone := pms.planetMilestones[pid]

			// Check planet milestones
			for _, m := range popMilestones {
				if planet.Population >= m.threshold && currentMilestone < m.threshold {
					pms.planetMilestones[pid] = m.threshold

					// Grant bonus
					for _, p := range players {
						if p != nil && p.Name == planet.Owner {
							p.Credits += m.credits
							break
						}
					}
					if m.techBonus > 0 {
						planet.TechLevel += m.techBonus
					}

					game.LogEvent("event", planet.Owner,
						fmt.Sprintf("🏙️ %s has become a %s! (%d population) +%dcr",
							planet.Name, m.title, planet.Population, m.credits))
					break // only announce highest new milestone
				}
			}
		}
	}

	// Check faction milestones
	factionThresholds := []struct {
		threshold int64
		title     string
	}{
		{100000, "Superpower"},
		{50000, "Major Power"},
		{10000, "Growing Empire"},
	}

	for faction, pop := range factionPop {
		current := pms.factionMilestones[faction]
		for _, ft := range factionThresholds {
			if pop >= ft.threshold && current < ft.threshold {
				pms.factionMilestones[faction] = ft.threshold
				game.LogEvent("event", faction,
					fmt.Sprintf("👥 %s is now a %s! Total population: %d",
						faction, ft.title, pop))
				break
			}
		}
	}
}
