package tickable

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradeLeagueSystem{
		BaseSystem: NewBaseSystem("TradeLeague", 138),
	})
}

// TradeLeagueSystem organizes periodic trade competitions between
// factions. Each "season" lasts 20,000 ticks and tracks who earns
// the most from trade activities.
//
// Trade League scoring per season:
//   +1 per shipping route trip completed
//   +1 per 1000cr earned from docking fees
//   +2 per freight contract completed
//   +3 per profitable futures contract
//
// At season end: #1 gets 5000cr prize, #2 gets 2000cr, #3 gets 1000cr.
// Season results announced with full standings.
type TradeLeagueSystem struct {
	*BaseSystem
	seasonScores map[string]int
	seasonNum    int
	seasonEnd    int64
}

func (tls *TradeLeagueSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := tls.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tls.seasonScores == nil {
		tls.seasonScores = make(map[string]int)
		tls.seasonEnd = tick + 20000
		tls.seasonNum = 1
	}

	players := ctx.GetPlayers()
	routes := game.GetShippingRoutes()

	// Accumulate scores from shipping trips
	for _, route := range routes {
		if route.Active && route.TripsComplete > 0 {
			tls.seasonScores[route.Owner] += route.TripsComplete / 10 // scaled
		}
	}

	// Credits earned proxy (wealthier = more trade activity)
	for _, p := range players {
		if p != nil && p.Credits > 100000 {
			tls.seasonScores[p.Name]++
		}
	}

	// Season end
	if tick >= tls.seasonEnd {
		tls.endSeason(game, players)
		tls.seasonScores = make(map[string]int)
		tls.seasonNum++
		tls.seasonEnd = tick + 20000
	}

	// Mid-season update
	if tick == tls.seasonEnd-10000 {
		type entry struct {
			name  string
			score int
		}
		var standings []entry
		for name, score := range tls.seasonScores {
			if score > 0 {
				standings = append(standings, entry{name, score})
			}
		}
		sort.Slice(standings, func(i, j int) bool { return standings[i].score > standings[j].score })

		if len(standings) > 0 && rand.Intn(2) == 0 {
			msg := fmt.Sprintf("🏆 Trade League Season %d — Halftime: ", tls.seasonNum)
			for i, s := range standings {
				if i >= 3 {
					break
				}
				msg += fmt.Sprintf("#%d %s(%d) ", i+1, s.name, s.score)
			}
			game.LogEvent("intel", "", msg)
		}
	}
}

func (tls *TradeLeagueSystem) endSeason(game GameProvider, players []*entities.Player) {
	type entry struct {
		name  string
		score int
	}
	var standings []entry
	for name, score := range tls.seasonScores {
		if score > 0 {
			standings = append(standings, entry{name, score})
		}
	}
	if len(standings) == 0 {
		return
	}

	sort.Slice(standings, func(i, j int) bool { return standings[i].score > standings[j].score })

	prizes := []int{5000, 2000, 1000}
	msg := fmt.Sprintf("🏆 TRADE LEAGUE SEASON %d RESULTS: ", tls.seasonNum)

	for i, s := range standings {
		if i >= 3 {
			break
		}
		medal := "🥉"
		if i == 0 {
			medal = "🥇"
		} else if i == 1 {
			medal = "🥈"
		}
		prize := 0
		if i < len(prizes) {
			prize = prizes[i]
		}

		msg += fmt.Sprintf("%s %s(%d pts, +%dcr) ", medal, s.name, s.score, prize)

		for _, p := range players {
			if p != nil && p.Name == s.name {
				p.Credits += prize
				break
			}
		}
	}

	game.LogEvent("event", "", msg)
}
