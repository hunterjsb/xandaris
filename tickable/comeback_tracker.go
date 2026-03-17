package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&ComebackTrackerSystem{
		BaseSystem: NewBaseSystem("ComebackTracker", 158),
	})
}

// ComebackTrackerSystem detects and celebrates dramatic reversals
// of fortune. When a faction that was in the bottom 3 climbs to
// the top 3 (or vice versa), it's a major story.
//
// Also tracks:
//   - Biggest single-period credit gain
//   - Biggest single-period credit loss
//   - Fastest rank climb
//   - "Rags to riches" (from <10K credits to >500K)
type ComebackTrackerSystem struct {
	*BaseSystem
	prevRanks    map[string]int
	prevCredits  map[string]int
	lowestCredits map[string]int // all-time lowest credits seen
	nextCheck    int64
}

func (cts *ComebackTrackerSystem) OnTick(tick int64) {
	ctx := cts.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if cts.prevRanks == nil {
		cts.prevRanks = make(map[string]int)
		cts.prevCredits = make(map[string]int)
		cts.lowestCredits = make(map[string]int)
	}

	if cts.nextCheck == 0 {
		cts.nextCheck = tick + 5000
	}
	if tick < cts.nextCheck {
		return
	}
	cts.nextCheck = tick + 5000 + int64(rand.Intn(3000))

	players := game.GetPlayers()

	// Build current rankings by credits
	type entry struct {
		name    string
		credits int
	}
	var ranked []entry
	for _, p := range players {
		if p != nil {
			ranked = append(ranked, entry{p.Name, p.Credits})
		}
	}
	// Sort by credits desc
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].credits > ranked[i].credits {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}

	for i, e := range ranked {
		rank := i + 1
		prevRank := cts.prevRanks[e.name]
		prevCredits := cts.prevCredits[e.name]

		// Track all-time lows
		if cts.lowestCredits[e.name] == 0 || e.credits < cts.lowestCredits[e.name] {
			cts.lowestCredits[e.name] = e.credits
		}

		// Rags to riches: from all-time low <10K to current >500K
		lowest := cts.lowestCredits[e.name]
		if lowest < 10000 && e.credits > 500000 && lowest > 0 {
			if rand.Intn(20) == 0 { // rare announcement
				game.LogEvent("event", e.name,
					fmt.Sprintf("🌟 RAGS TO RICHES: %s once had only %dcr — now commands %dcr! An incredible comeback story!",
						e.name, lowest, e.credits))
			}
		}

		// Big rank changes
		if prevRank > 0 && rank <= 3 && prevRank > 5 {
			game.LogEvent("event", e.name,
				fmt.Sprintf("🚀 COMEBACK! %s surged from #%d to #%d! From underdog to contender!",
					e.name, prevRank, rank))
		}

		// Dramatic falls
		if prevRank > 0 && prevRank <= 3 && rank > 5 {
			game.LogEvent("event", e.name,
				fmt.Sprintf("📉 FALL FROM GRACE: %s dropped from #%d to #%d! What went wrong?",
					e.name, prevRank, rank))
		}

		// Biggest gain/loss
		if prevCredits > 0 {
			delta := e.credits - prevCredits
			if delta > 200000 {
				game.LogEvent("event", e.name,
					fmt.Sprintf("💰 WINDFALL: %s gained %dcr in one period! (now: %dcr)",
						e.name, delta, e.credits))
			} else if delta < -200000 {
				game.LogEvent("event", e.name,
					fmt.Sprintf("💸 CRASH: %s lost %dcr in one period! (now: %dcr)",
						e.name, -delta, e.credits))
			}
		}

		cts.prevRanks[e.name] = rank
		cts.prevCredits[e.name] = e.credits
	}
}
