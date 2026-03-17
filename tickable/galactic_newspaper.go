package tickable

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticNewspaperSystem{
		BaseSystem: NewBaseSystem("GalacticNewspaper", 95),
	})
}

// GalacticNewspaperSystem publishes a periodic "newspaper" with
// multiple stories covering different aspects of galaxy life.
// Unlike individual event systems that each post their own events,
// the newspaper aggregates everything into one coherent bulletin.
//
// Published every ~8000 ticks (~13 minutes). Contains 3-5 stories:
//   - Economic headline (market trends, trade activity)
//   - Military headline (fleet movements, battles, arms races)
//   - Society headline (population, happiness, festivals)
//   - Science headline (tech breakthroughs, discoveries)
//   - Wild card (random interesting fact about the galaxy)
type GalacticNewspaperSystem struct {
	*BaseSystem
	nextEdition int64
	edition     int
}

func (gns *GalacticNewspaperSystem) OnTick(tick int64) {
	ctx := gns.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if gns.nextEdition == 0 {
		gns.nextEdition = tick + 5000 + int64(rand.Intn(5000))
	}
	if tick < gns.nextEdition {
		return
	}
	gns.nextEdition = tick + 8000 + int64(rand.Intn(5000))
	gns.edition++

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	stories := []string{
		gns.economicStory(players, game),
		gns.militaryStory(players),
		gns.societyStory(players, systems),
		gns.scienceStory(systems),
	}

	// Filter empty stories
	var headlines []string
	for _, s := range stories {
		if s != "" {
			headlines = append(headlines, s)
		}
	}

	if len(headlines) == 0 {
		return
	}

	// Publish
	header := fmt.Sprintf("📰 GALACTIC TIMES #%d", gns.edition)
	msg := header
	for _, h := range headlines {
		msg += " | " + h
	}

	game.LogEvent("intel", "", msg)
}

func (gns *GalacticNewspaperSystem) economicStory(players []*entities.Player, game GameProvider) string {
	// Find richest and poorest
	var richest, poorest *entities.Player
	for _, p := range players {
		if p == nil {
			continue
		}
		if richest == nil || p.Credits > richest.Credits {
			richest = p
		}
		if poorest == nil || p.Credits < poorest.Credits {
			poorest = p
		}
	}

	if richest == nil {
		return ""
	}

	templates := []string{
		fmt.Sprintf("💰 %s tops wealth charts at %dcr", richest.Name, richest.Credits),
		fmt.Sprintf("💸 %s struggles with only %dcr in the treasury", poorest.Name, poorest.Credits),
	}

	if richest.Credits > 5000000 {
		templates = append(templates, fmt.Sprintf("💰 %s's fortune exceeds 5 MILLION credits!", richest.Name))
	}

	return templates[rand.Intn(len(templates))]
}

func (gns *GalacticNewspaperSystem) militaryStory(players []*entities.Player) string {
	// Count total military ships
	type fleetSize struct {
		name  string
		ships int
	}
	var fleets []fleetSize
	for _, p := range players {
		if p == nil {
			continue
		}
		military := 0
		for _, ship := range p.OwnedShips {
			if ship != nil && (ship.ShipType == entities.ShipTypeFrigate ||
				ship.ShipType == entities.ShipTypeDestroyer ||
				ship.ShipType == entities.ShipTypeCruiser) {
				military++
			}
		}
		if military > 0 {
			fleets = append(fleets, fleetSize{p.Name, military})
		}
	}

	if len(fleets) == 0 {
		return "⚔️ Galaxy at peace — no military fleets deployed"
	}

	sort.Slice(fleets, func(i, j int) bool { return fleets[i].ships > fleets[j].ships })

	return fmt.Sprintf("⚔️ Military: %s leads with %d warships", fleets[0].name, fleets[0].ships)
}

func (gns *GalacticNewspaperSystem) societyStory(players []*entities.Player, systems []*entities.System) string {
	totalPop := int64(0)
	happiestPlanet := ""
	happiestScore := 0.0
	happiestOwner := ""

	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				totalPop += planet.Population
				if planet.Happiness > happiestScore && planet.Population > 100 {
					happiestScore = planet.Happiness
					happiestPlanet = planet.Name
					happiestOwner = planet.Owner
				}
			}
		}
	}

	if happiestPlanet != "" {
		return fmt.Sprintf("👥 Pop: %d total | Happiest: %s (%s, %.0f%%)",
			totalPop, happiestPlanet, happiestOwner, happiestScore*100)
	}
	return fmt.Sprintf("👥 Galactic population: %d", totalPop)
}

func (gns *GalacticNewspaperSystem) scienceStory(systems []*entities.System) string {
	bestTech := 0.0
	bestPlanet := ""
	bestOwner := ""

	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" && planet.TechLevel > bestTech {
				bestTech = planet.TechLevel
				bestPlanet = planet.Name
				bestOwner = planet.Owner
			}
		}
	}

	if bestPlanet != "" {
		return fmt.Sprintf("🔬 Science: %s (%s) leads at tech %.1f", bestPlanet, bestOwner, bestTech)
	}
	return ""
}
