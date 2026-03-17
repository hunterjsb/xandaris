package tickable

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&DailyBriefingSystem{
		BaseSystem: NewBaseSystem("DailyBriefing", 150),
	})
}

// DailyBriefingSystem generates a comprehensive briefing for each
// faction every ~10,000 ticks (~17 min). This is the ONE event that
// tells a faction everything they need to know.
//
// Briefing format:
//   "BRIEFING for Llama Logistics:
//    Credits: 936K (+120K) | Planets: 10 | Pop: 5849
//    Routes: 8 active (3 completing) | Fleet: 72 ships
//    Threats: power crisis on 2 planets
//    Opportunity: Golden Route for Iron via SYS-5
//    Advice: Build more Refineries"
//
// Consolidates all intel into one actionable summary per faction.
type DailyBriefingSystem struct {
	*BaseSystem
	prevCredits map[string]int
	nextBrief   int64
}

func (dbs *DailyBriefingSystem) OnTick(tick int64) {
	ctx := dbs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if dbs.prevCredits == nil {
		dbs.prevCredits = make(map[string]int)
	}

	if dbs.nextBrief == 0 {
		dbs.nextBrief = tick + 5000 + int64(rand.Intn(5000))
	}
	if tick < dbs.nextBrief {
		return
	}
	dbs.nextBrief = tick + 10000 + int64(rand.Intn(3000))

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	routes := game.GetShippingRoutes()

	for _, player := range players {
		if player == nil {
			continue
		}

		// Gather stats
		creditDelta := player.Credits - dbs.prevCredits[player.Name]
		dbs.prevCredits[player.Name] = player.Credits

		planetCount := 0
		totalPop := int64(0)
		powerCrisis := 0
		unhappy := 0
		bestPlanet := ""
		bestHappiness := 0.0

		for _, sys := range systems {
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner != player.Name {
					continue
				}
				planetCount++
				totalPop += planet.Population
				if planet.GetPowerRatio() < 0.3 {
					powerCrisis++
				}
				if planet.Happiness < 0.2 {
					unhappy++
				}
				if planet.Happiness > bestHappiness {
					bestHappiness = planet.Happiness
					bestPlanet = planet.Name
				}
			}
		}

		// Route stats
		activeRoutes := 0
		completingRoutes := 0
		for _, r := range routes {
			if r.Owner == player.Name && r.Active {
				activeRoutes++
				if r.TripsComplete > 0 {
					completingRoutes++
				}
			}
		}

		// Build briefing
		deltaStr := fmt.Sprintf("%+d", creditDelta)
		msg := fmt.Sprintf("📋 BRIEFING %s: %dcr(%s) | %d planets, %d pop | %d ships",
			player.Name, player.Credits, deltaStr, planetCount, totalPop, len(player.OwnedShips))

		if activeRoutes > 0 {
			msg += fmt.Sprintf(" | Routes: %d(%d delivering)", activeRoutes, completingRoutes)
		}

		// Threats
		threats := []string{}
		if powerCrisis > 0 {
			threats = append(threats, fmt.Sprintf("%d power crisis", powerCrisis))
		}
		if unhappy > 0 {
			threats = append(threats, fmt.Sprintf("%d unhappy planets", unhappy))
		}
		if len(threats) > 0 {
			msg += " | ⚠️"
			for _, t := range threats {
				msg += " " + t
			}
		}

		if bestPlanet != "" {
			msg += fmt.Sprintf(" | Best: %s(%.0f%% happy)", bestPlanet, bestHappiness*100)
		}

		game.LogEvent("intel", player.Name, msg)
	}

	// Also a quick galaxy summary
	type ranked struct {
		name    string
		credits int
	}
	var ranks []ranked
	for _, p := range players {
		if p != nil {
			ranks = append(ranks, ranked{p.Name, p.Credits})
		}
	}
	sort.Slice(ranks, func(i, j int) bool { return ranks[i].credits > ranks[j].credits })

	if len(ranks) >= 2 {
		game.LogEvent("intel", "",
			fmt.Sprintf("📋 Galaxy standings: #1 %s(%dcr) #2 %s(%dcr)",
				ranks[0].name, ranks[0].credits, ranks[1].name, ranks[1].credits))
	}
}
