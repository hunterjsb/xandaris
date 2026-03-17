package tickable

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&PowerRankingsSystem{
		BaseSystem: NewBaseSystem("PowerRankings", 88),
	})
}

// PowerRankingsSystem announces dramatic faction comparisons
// in a sportscaster style. Rather than dry stats, it creates
// narrative around the competition between factions.
//
// Commentary types:
//   - "X is closing the gap on Y!" (credit difference shrinking)
//   - "X's empire is crumbling!" (losing planets)
//   - "Underdog X just passed Y!" (rank change)
//   - "X dominates with Z times more power than nearest rival!"
//   - "The race for first is heating up!"
//
// This makes the leaderboard feel alive and creates storylines.
type PowerRankingsSystem struct {
	*BaseSystem
	prevRanks map[string]int // faction → previous rank
	nextComm  int64
}

func (prs *PowerRankingsSystem) OnTick(tick int64) {
	ctx := prs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if prs.prevRanks == nil {
		prs.prevRanks = make(map[string]int)
	}

	if prs.nextComm == 0 {
		prs.nextComm = tick + 5000 + int64(rand.Intn(5000))
	}
	if tick < prs.nextComm {
		return
	}
	prs.nextComm = tick + 8000 + int64(rand.Intn(8000))

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Calculate power for each faction
	type factionPower struct {
		name    string
		power   int
		planets int
		credits int
	}

	var rankings []factionPower
	for _, p := range players {
		if p == nil {
			continue
		}
		fp := factionPower{name: p.Name, credits: p.Credits}
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == p.Name {
					fp.planets++
					fp.power += int(planet.Population/100) + int(planet.TechLevel*100)
				}
			}
		}
		fp.power += p.Credits / 1000
		fp.power += len(p.OwnedShips) * 10
		fp.power += fp.planets * 200
		rankings = append(rankings, fp)
	}

	if len(rankings) < 2 {
		return
	}

	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].power > rankings[j].power
	})

	// Generate commentary
	commentary := ""

	// Check for rank changes
	for i, fp := range rankings {
		rank := i + 1
		prevRank := prs.prevRanks[fp.name]
		if prevRank > 0 && rank < prevRank {
			commentary = fmt.Sprintf("🔥 UPSET! %s climbed from #%d to #%d!", fp.name, prevRank, rank)
			break
		}
		if prevRank > 0 && rank > prevRank && prevRank <= 3 {
			commentary = fmt.Sprintf("📉 %s dropped from #%d to #%d!", fp.name, prevRank, rank)
			break
		}
	}

	// If no rank change, generate other commentary
	if commentary == "" {
		leader := rankings[0]
		runner := rankings[1]

		gap := leader.power - runner.power
		gapPct := 0.0
		if runner.power > 0 {
			gapPct = float64(gap) / float64(runner.power) * 100
		}

		switch {
		case gapPct > 100:
			commentary = fmt.Sprintf("👑 %s DOMINATES with %dx the power of %s! Can anyone challenge this empire?",
				leader.name, leader.power/runner.power, runner.name)
		case gapPct < 10:
			commentary = fmt.Sprintf("🔥 TIGHT RACE! %s leads %s by just %.0f%%! The galaxy holds its breath!",
				leader.name, runner.name, gapPct)
		case len(rankings) >= 3 && rankings[len(rankings)-1].credits < 10000:
			last := rankings[len(rankings)-1]
			commentary = fmt.Sprintf("⚠️ %s is in CRISIS with only %dcr! Will they survive or crumble?",
				last.name, last.credits)
		default:
			commentary = fmt.Sprintf("📊 %s leads with %d power | %s at %d | %s at %d",
				rankings[0].name, rankings[0].power,
				rankings[1].name, rankings[1].power,
				rankings[2].name, rankings[2].power)
		}
	}

	// Update ranks
	for i, fp := range rankings {
		prs.prevRanks[fp.name] = i + 1
	}

	game.LogEvent("intel", "", commentary)
}
