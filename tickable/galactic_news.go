package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticNewsSystem{
		BaseSystem: NewBaseSystem("GalacticNews", 45),
	})
}

// GalacticNewsSystem generates narrative headlines from game state.
// Headlines create immersion and summarize what's happening.
type GalacticNewsSystem struct {
	*BaseSystem
	lastHeadline int64
}

func (gns *GalacticNewsSystem) OnTick(tick int64) {
	if gns.lastHeadline == 0 {
		gns.lastHeadline = tick
	}
	if tick-gns.lastHeadline < 3000 {
		return // ~5 min between headlines
	}
	gns.lastHeadline = tick

	ctx := gns.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	headline := gns.generateHeadline(players, systems)
	if headline != "" {
		game.LogEvent("news", "", headline)
	}
}

func (gns *GalacticNewsSystem) generateHeadline(players []*entities.Player, systems []*entities.System) string {
	// Pick a random headline type
	switch rand.Intn(8) {
	case 0: // Richest faction
		var richest *entities.Player
		for _, p := range players {
			if p != nil && (richest == nil || p.Credits > richest.Credits) {
				richest = p
			}
		}
		if richest != nil {
			return fmt.Sprintf("📰 GALACTIC TIMES: %s leads galactic economy with %d credits — analysts predict continued dominance",
				richest.Name, richest.Credits)
		}

	case 1: // Most planets
		var biggest *entities.Player
		maxPlanets := 0
		for _, p := range players {
			if p == nil {
				continue
			}
			count := 0
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if pl, ok := e.(*entities.Planet); ok && pl.Owner == p.Name {
						count++
					}
				}
			}
			if count > maxPlanets {
				maxPlanets = count
				biggest = p
			}
		}
		if biggest != nil {
			return fmt.Sprintf("📰 GALACTIC TIMES: %s controls %d planets — the largest empire in known space",
				biggest.Name, maxPlanets)
		}

	case 2: // Population milestone
		var totalPop int64
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if p, ok := e.(*entities.Planet); ok && p.Owner != "" {
					totalPop += p.Population
				}
			}
		}
		return fmt.Sprintf("📰 CENSUS: Galaxy population reaches %d — demand for housing and resources grows",
			totalPop)

	case 3: // Trade commentary
		templates := []string{
			"📰 MARKETS: Commodity traders report increased volatility across all sectors",
			"📰 MARKETS: Analysts debate whether Electronics shortage will ease this quarter",
			"📰 MARKETS: Oil futures spike as trade routes face pirate disruption",
			"📰 MARKETS: Rare Metals glut continues — prices at historic lows",
		}
		return templates[rand.Intn(len(templates))]

	case 4: // Military
		templates := []string{
			"📰 SECURITY: Pirate activity reported in outer systems — mercenary demand rises",
			"📰 SECURITY: Military buildup observed in contested sectors",
			"📰 SECURITY: Trade route safety improving as factions deploy escort fleets",
		}
		return templates[rand.Intn(len(templates))]

	case 5: // Science
		var maxTech float64
		var techLeader string
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if p, ok := e.(*entities.Planet); ok && p.Owner != "" && p.TechLevel > maxTech {
					maxTech = p.TechLevel
					techLeader = p.Owner
				}
			}
		}
		if techLeader != "" {
			return fmt.Sprintf("📰 SCIENCE: %s leads technological advancement at level %.1f — new breakthroughs expected",
				techLeader, maxTech)
		}

	case 6: // Diplomacy
		templates := []string{
			"📰 DIPLOMACY: Tensions rise between competing factions over resource-rich systems",
			"📰 DIPLOMACY: Trade alliance negotiations underway between major factions",
			"📰 DIPLOMACY: Galactic Council session scheduled — several proposals pending",
		}
		return templates[rand.Intn(len(templates))]

	case 7: // Human interest
		templates := []string{
			"📰 CULTURE: New colony celebrates first harvest festival",
			"📰 CULTURE: Terraforming efforts show promising results on frontier worlds",
			"📰 CULTURE: Population boom on several core worlds — housing demand surges",
			"📰 CULTURE: Legendary ship sighting sparks explorer frenzy across the galaxy",
		}
		return templates[rand.Intn(len(templates))]
	}

	return ""
}
