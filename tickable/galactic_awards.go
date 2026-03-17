package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticAwardsSystem{
		BaseSystem: NewBaseSystem("GalacticAwards", 141),
	})
}

// GalacticAwardsSystem holds a periodic awards ceremony recognizing
// factions for specific achievements during the last period.
//
// Awards (handed out every ~15000 ticks):
//   Most Improved: faction with biggest credit growth
//   Best Employer: faction with highest avg happiness
//   Greenest Planet: planet with highest habitability
//   Busiest Port: system with most ship traffic
//   Most Generous: faction that paid most in docking fees to others
//
// Winners get a small credit bonus and galactic recognition.
type GalacticAwardsSystem struct {
	*BaseSystem
	nextCeremony int64
	prevCredits  map[string]int
}

func (gas *GalacticAwardsSystem) OnTick(tick int64) {
	ctx := gas.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if gas.prevCredits == nil {
		gas.prevCredits = make(map[string]int)
		for _, p := range ctx.GetPlayers() {
			if p != nil {
				gas.prevCredits[p.Name] = p.Credits
			}
		}
	}

	if gas.nextCeremony == 0 {
		gas.nextCeremony = tick + 10000 + int64(rand.Intn(5000))
	}
	if tick < gas.nextCeremony {
		return
	}
	gas.nextCeremony = tick + 15000 + int64(rand.Intn(10000))

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Most Improved (biggest credit gain)
	bestGrowth := 0
	bestGrower := ""
	for _, p := range players {
		if p == nil {
			continue
		}
		growth := p.Credits - gas.prevCredits[p.Name]
		if growth > bestGrowth {
			bestGrowth = growth
			bestGrower = p.Name
		}
		gas.prevCredits[p.Name] = p.Credits
	}

	// Best Employer (highest avg happiness)
	type factionHappy struct {
		name     string
		avgHappy float64
		count    int
	}
	happyMap := make(map[string]*factionHappy)
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" && planet.Population > 100 {
				if happyMap[planet.Owner] == nil {
					happyMap[planet.Owner] = &factionHappy{name: planet.Owner}
				}
				happyMap[planet.Owner].avgHappy += planet.Happiness
				happyMap[planet.Owner].count++
			}
		}
	}
	bestHappy := ""
	bestHappyScore := 0.0
	for _, fh := range happyMap {
		if fh.count > 0 {
			avg := fh.avgHappy / float64(fh.count)
			if avg > bestHappyScore {
				bestHappyScore = avg
				bestHappy = fh.name
			}
		}
	}

	// Busiest port
	busiestSys := ""
	busiestCount := 0
	for _, sys := range systems {
		count := 0
		for _, p := range players {
			if p == nil {
				continue
			}
			for _, ship := range p.OwnedShips {
				if ship != nil && ship.CurrentSystem == sys.ID {
					count++
				}
			}
		}
		if count > busiestCount {
			busiestCount = count
			busiestSys = sys.Name
		}
	}

	// Announce ceremony
	msg := "🏆 GALACTIC AWARDS CEREMONY: "
	awards := 0

	if bestGrower != "" && bestGrowth > 0 {
		msg += fmt.Sprintf("Most Improved: %s (+%dcr) ", bestGrower, bestGrowth)
		for _, p := range players {
			if p != nil && p.Name == bestGrower {
				p.Credits += 1000
				break
			}
		}
		awards++
	}
	if bestHappy != "" {
		msg += fmt.Sprintf("| Best Employer: %s (%.0f%% avg happiness) ", bestHappy, bestHappyScore*100)
		for _, p := range players {
			if p != nil && p.Name == bestHappy {
				p.Credits += 1000
				break
			}
		}
		awards++
	}
	if busiestSys != "" {
		msg += fmt.Sprintf("| Busiest Port: %s (%d ships)", busiestSys, busiestCount)
		awards++
	}

	if awards > 0 {
		game.LogEvent("event", "", msg)
	}
}
