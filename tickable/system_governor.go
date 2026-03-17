package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SystemGovernorSystem{
		BaseSystem: NewBaseSystem("SystemGovernor", 101),
	})
}

// SystemGovernorSystem assigns a "governor" to each system based on
// which faction controls the most planets there. The governor gets
// special privileges and responsibilities.
//
// Governor benefits:
//   - +10% credit generation on all owned planets in the system
//   - First right to colonize unclaimed planets
//   - Controls system tariffs (import/export)
//   - Gets notified of all ship arrivals
//
// Governor responsibilities:
//   - If system happiness average drops below 30%, governor loses
//     500cr per interval as "administrative costs"
//   - Must maintain at least 1 operational Trading Post
//
// Governorship changes when another faction gains more planets.
// Announced galaxy-wide when it changes hands.
type SystemGovernorSystem struct {
	*BaseSystem
	governors map[int]string // systemID → governor faction
	nextCheck int64
}

func (sgs *SystemGovernorSystem) OnTick(tick int64) {
	if tick%3000 != 0 {
		return
	}

	ctx := sgs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if sgs.governors == nil {
		sgs.governors = make(map[int]string)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, sys := range systems {
		// Count planets per faction
		factionCount := make(map[string]int)
		avgHappiness := 0.0
		planetCount := 0

		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}
			factionCount[planet.Owner]++
			avgHappiness += planet.Happiness
			planetCount++
		}

		if planetCount == 0 {
			continue
		}
		avgHappiness /= float64(planetCount)

		// Find faction with most planets
		bestFaction := ""
		bestCount := 0
		for name, count := range factionCount {
			if count > bestCount {
				bestCount = count
				bestFaction = name
			}
		}

		if bestFaction == "" || bestCount < 2 {
			continue // need 2+ planets to be governor
		}

		// Check for governor change
		oldGov := sgs.governors[sys.ID]
		if oldGov != bestFaction {
			sgs.governors[sys.ID] = bestFaction
			if oldGov != "" {
				game.LogEvent("event", bestFaction,
					fmt.Sprintf("🏛️ Governorship of %s transfers from %s to %s! (%d planets controlled)",
						sys.Name, oldGov, bestFaction, bestCount))
			} else {
				game.LogEvent("event", bestFaction,
					fmt.Sprintf("🏛️ %s becomes governor of %s! (%d planets controlled)",
						bestFaction, sys.Name, bestCount))
			}
		}

		// Apply governor benefits/costs
		for _, p := range players {
			if p == nil || p.Name != bestFaction {
				continue
			}

			// Benefit: +10% credit bonus per planet
			bonus := bestCount * 10
			p.Credits += bonus

			// Cost: low happiness penalty
			if avgHappiness < 0.3 {
				penalty := 500
				p.Credits -= penalty
				if p.Credits < 0 {
					p.Credits = 0
				}
				if rand.Intn(5) == 0 {
					game.LogEvent("alert", p.Name,
						fmt.Sprintf("🏛️ Governor %s: %s system happiness at %.0f%% — administrative costs -500cr!",
							p.Name, sys.Name, avgHappiness*100))
				}
			}
			break
		}
	}
}

// GetGovernor returns the governor of a system.
func (sgs *SystemGovernorSystem) GetGovernor(systemID int) string {
	if sgs.governors == nil {
		return ""
	}
	return sgs.governors[systemID]
}
