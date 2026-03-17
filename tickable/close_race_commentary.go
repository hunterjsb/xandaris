package tickable

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&CloseRaceCommentarySystem{
		BaseSystem: NewBaseSystem("CloseRaceCommentary", 136),
	})
}

// CloseRaceCommentarySystem generates exciting play-by-play commentary
// when the top factions are in a close race. When the gap between #1
// and #2 is under 5%, it fires dramatic announcements every few minutes.
//
// This makes the leaderboard race feel like a live sporting event.
type CloseRaceCommentarySystem struct {
	*BaseSystem
	prevScores map[string]int
	nextCheck  int64
}

func (crcs *CloseRaceCommentarySystem) OnTick(tick int64) {
	ctx := crcs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if crcs.prevScores == nil {
		crcs.prevScores = make(map[string]int)
	}

	if crcs.nextCheck == 0 {
		crcs.nextCheck = tick + 5000
	}
	if tick < crcs.nextCheck {
		return
	}
	crcs.nextCheck = tick + 5000 + int64(rand.Intn(3000))

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Calculate power scores
	type fScore struct {
		name  string
		score int
	}
	var scores []fScore

	for _, p := range players {
		if p == nil {
			continue
		}
		s := p.Credits / 100
		s += len(p.OwnedShips) * 20
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == p.Name {
					s += 500 + int(planet.Population/10) + int(planet.TechLevel*100)
				}
			}
		}
		scores = append(scores, fScore{p.Name, s})
	}

	if len(scores) < 2 {
		return
	}

	sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })

	first := scores[0]
	second := scores[1]

	gap := float64(first.score-second.score) / float64(first.score) * 100

	// Detect lead changes
	prevFirst := ""
	prevFirstScore := 0
	for name, s := range crcs.prevScores {
		if s > prevFirstScore {
			prevFirstScore = s
			prevFirst = name
		}
	}

	// Update scores
	for _, s := range scores {
		crcs.prevScores[s.name] = s.score
	}

	// Lead change!
	if prevFirst != "" && prevFirst != first.name {
		game.LogEvent("event", first.name,
			fmt.Sprintf("🔥 LEAD CHANGE! %s overtakes %s for the #1 spot! Gap: %.1f%%",
				first.name, prevFirst, gap))
		return
	}

	// Close race commentary
	if gap > 5 {
		return // not close enough
	}

	templates := []string{
		fmt.Sprintf("🏁 TIGHT RACE: %s leads %s by just %.1f%%! One good trade run could flip this!",
			first.name, second.name, gap),
		fmt.Sprintf("🏁 %s and %s are NECK AND NECK! (gap: %.1f%%) The galaxy watches breathlessly!",
			first.name, second.name, gap),
		fmt.Sprintf("🏁 %.1f%% separates #1 %s from #2 %s. This is anyone's game!",
			gap, first.name, second.name),
	}

	if len(scores) >= 3 {
		third := scores[2]
		gapThird := float64(first.score-third.score) / float64(first.score) * 100
		if gapThird < 15 {
			templates = append(templates,
				fmt.Sprintf("🏁 THREE-WAY RACE: %s, %s, and %s all within %.0f%% of each other!",
					first.name, second.name, third.name, gapThird))
		}
	}

	game.LogEvent("intel", "", templates[rand.Intn(len(templates))])

	_ = math.Abs // suppress
}
