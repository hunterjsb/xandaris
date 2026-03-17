package tickable

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticCensusSystem{
		BaseSystem: NewBaseSystem("GalacticCensus", 76),
	})
}

// GalacticCensusSystem periodically publishes a comprehensive
// galaxy-wide summary of all factions. This creates context for
// strategic decisions and makes the galaxy feel alive.
//
// Census includes:
//   - Population rankings
//   - Territory rankings (planets owned)
//   - Military power rankings
//   - Economic rankings (credits + trade volume)
//   - Technology rankings
//   - Overall power index
//
// Published every ~10,000 ticks (~17 minutes).
// The census also calculates a "balance of power" metric
// to detect if one faction is becoming too dominant.
type GalacticCensusSystem struct {
	*BaseSystem
	nextCensus int64
}

type factionStats struct {
	name       string
	credits    int
	population int64
	planets    int
	military   int // military ship count
	ships      int
	techAvg    float64
	powerIndex int
}

func (gcs *GalacticCensusSystem) OnTick(tick int64) {
	ctx := gcs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if gcs.nextCensus == 0 {
		gcs.nextCensus = tick + 5000 + int64(rand.Intn(5000))
	}
	if tick < gcs.nextCensus {
		return
	}
	gcs.nextCensus = tick + 10000 + int64(rand.Intn(5000))

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Gather stats
	stats := make(map[string]*factionStats)

	for _, p := range players {
		if p == nil {
			continue
		}
		s := &factionStats{name: p.Name, credits: p.Credits}
		for _, ship := range p.OwnedShips {
			if ship == nil {
				continue
			}
			s.ships++
			if ship.ShipType == entities.ShipTypeFrigate ||
				ship.ShipType == entities.ShipTypeDestroyer ||
				ship.ShipType == entities.ShipTypeCruiser {
				s.military++
			}
		}
		stats[p.Name] = s
	}

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}
			s := stats[planet.Owner]
			if s == nil {
				continue
			}
			s.planets++
			s.population += planet.Population
			s.techAvg += planet.TechLevel
		}
	}

	// Calculate power index and tech average
	var entries []*factionStats
	for _, s := range stats {
		if s.planets > 0 {
			s.techAvg /= float64(s.planets)
		}
		s.powerIndex = s.credits/1000 + int(s.population/100) + s.planets*500 + s.military*200 + int(s.techAvg*100)
		entries = append(entries, s)
	}

	if len(entries) == 0 {
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].powerIndex > entries[j].powerIndex
	})

	// Generate census report
	msg := "📜 GALACTIC CENSUS: "
	for i, s := range entries {
		if i >= 5 {
			break
		}
		rank := i + 1
		msg += fmt.Sprintf("#%d %s (power: %d, %d planets, %d pop, %d military) ",
			rank, s.name, s.powerIndex, s.planets, s.population, s.military)
	}

	game.LogEvent("intel", "", msg)

	// Balance of power check
	if len(entries) >= 2 {
		top := entries[0]
		second := entries[1]
		if second.powerIndex > 0 {
			ratio := float64(top.powerIndex) / float64(second.powerIndex)
			if ratio > 3.0 {
				game.LogEvent("intel", "",
					fmt.Sprintf("⚠️ POWER IMBALANCE: %s has %.1fx the power of %s. The galaxy trembles under their dominance!",
						top.name, ratio, second.name))
			}
		}
	}
}
