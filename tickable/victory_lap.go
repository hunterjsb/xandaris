package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&VictoryLapSystem{
		BaseSystem: NewBaseSystem("VictoryLap", 110),
	})
}

// VictoryLapSystem rewards factions that achieve victories with
// ongoing "champion" bonuses. After winning a victory condition,
// the faction becomes the reigning champion in that category
// and gets persistent bonuses until someone else surpasses them.
//
// Champion categories (based on victory.go conditions):
//   Trade Champion:      most shipping trips → +5% trade income
//   Economic Champion:   most credits → +2% interest on savings
//   Military Champion:   most warships → +10% attack power
//   Science Champion:    highest avg tech → +5% tech growth
//   Population Champion: most population → +5% pop growth
//
// Champions are re-evaluated every 10,000 ticks. The title can
// change hands, creating ongoing competition even after victory.
type VictoryLapSystem struct {
	*BaseSystem
	champions map[string]string // category → faction name
	nextEval  int64
}

func (vls *VictoryLapSystem) OnTick(tick int64) {
	if tick%5000 != 0 {
		return
	}

	ctx := vls.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if vls.champions == nil {
		vls.champions = make(map[string]string)
	}

	if vls.nextEval == 0 {
		vls.nextEval = tick + 5000
	}
	if tick < vls.nextEval {
		return
	}
	vls.nextEval = tick + 10000

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	routes := game.GetShippingRoutes()

	type stats struct {
		name     string
		trips    int
		credits  int
		military int
		techAvg  float64
		planets  int
		pop      int64
	}

	var all []stats
	for _, p := range players {
		if p == nil {
			continue
		}
		s := stats{name: p.Name, credits: p.Credits}

		// Count military
		for _, ship := range p.OwnedShips {
			if ship != nil && (ship.ShipType == entities.ShipTypeFrigate ||
				ship.ShipType == entities.ShipTypeDestroyer ||
				ship.ShipType == entities.ShipTypeCruiser) {
				s.military++
			}
		}

		// Count planets + pop + tech
		techTotal := 0.0
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == p.Name {
					s.planets++
					s.pop += planet.Population
					techTotal += planet.TechLevel
				}
			}
		}
		if s.planets > 0 {
			s.techAvg = techTotal / float64(s.planets)
		}

		// Count trips
		for _, route := range routes {
			if route.Owner == p.Name {
				s.trips += route.TripsComplete
			}
		}

		all = append(all, s)
	}

	if len(all) == 0 {
		return
	}

	// Evaluate each category
	categories := []struct {
		name    string
		getValue func(stats) int
	}{
		{"Trade", func(s stats) int { return s.trips }},
		{"Economic", func(s stats) int { return s.credits }},
		{"Military", func(s stats) int { return s.military }},
		{"Science", func(s stats) int { return int(s.techAvg * 100) }},
		{"Population", func(s stats) int { return int(s.pop) }},
	}

	for _, cat := range categories {
		best := ""
		bestVal := 0
		for _, s := range all {
			v := cat.getValue(s)
			if v > bestVal {
				bestVal = v
				best = s.name
			}
		}

		if best == "" || bestVal == 0 {
			continue
		}

		oldChamp := vls.champions[cat.name]
		if best != oldChamp {
			vls.champions[cat.name] = best
			if oldChamp != "" {
				game.LogEvent("event", best,
					fmt.Sprintf("👑 %s dethroned %s as %s Champion!",
						best, oldChamp, cat.name))
			} else {
				game.LogEvent("event", best,
					fmt.Sprintf("👑 %s crowned %s Champion!",
						best, cat.name))
			}
		}
	}

	// Apply champion bonuses
	for _, p := range players {
		if p == nil {
			continue
		}
		bonus := 0
		for _, faction := range vls.champions {
			if faction == p.Name {
				bonus += 25 // small ongoing champion bonus per title
			}
		}
		p.Credits += bonus
	}

	// Announce all champions periodically
	if rand.Intn(3) == 0 {
		msg := "👑 Champions: "
		for cat, faction := range vls.champions {
			msg += fmt.Sprintf("%s=%s ", cat, faction)
		}
		game.LogEvent("intel", "", msg)
	}
}
