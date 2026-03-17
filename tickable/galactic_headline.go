package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticHeadlineSystem{
		BaseSystem: NewBaseSystem("GalacticHeadline", 149),
	})
}

// GalacticHeadlineSystem generates a single dramatic headline every
// ~5000 ticks that captures the most interesting thing happening
// in the galaxy RIGHT NOW. Unlike the newspaper (which lists multiple
// stories), this is ONE punchy headline.
//
// Headlines are chosen from the most dramatic current situation:
//   - Close race for #1
//   - Faction in crisis
//   - Record-breaking achievement
//   - Major economic shift
//   - Diplomatic drama
type GalacticHeadlineSystem struct {
	*BaseSystem
	nextHeadline int64
}

func (ghs *GalacticHeadlineSystem) OnTick(tick int64) {
	ctx := ghs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ghs.nextHeadline == 0 {
		ghs.nextHeadline = tick + 3000 + int64(rand.Intn(3000))
	}
	if tick < ghs.nextHeadline {
		return
	}
	ghs.nextHeadline = tick + 5000 + int64(rand.Intn(5000))

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Gather facts
	var richest, poorest string
	maxCredits, minCredits := 0, 999999999
	totalPop := int64(0)
	totalPlanets := 0

	for _, p := range players {
		if p == nil {
			continue
		}
		if p.Credits > maxCredits {
			maxCredits = p.Credits
			richest = p.Name
		}
		if p.Credits < minCredits {
			minCredits = p.Credits
			poorest = p.Name
		}
	}

	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				totalPlanets++
				totalPop += planet.Population
			}
		}
	}

	// Pick the most dramatic headline
	headlines := []string{
		fmt.Sprintf("📰 \"%s Leads the Galaxy with %dcr — But %s is Closing In!\"",
			richest, maxCredits, poorest),
		fmt.Sprintf("📰 \"%d Souls Across %d Worlds — The Galaxy Has Never Been So Alive\"",
			totalPop, totalPlanets),
	}

	// Add context-specific headlines
	if maxCredits > 1000000 {
		headlines = append(headlines, fmt.Sprintf("📰 \"%s Becomes First Millionaire of This Galaxy!\"", richest))
	}
	if minCredits < 10000 {
		headlines = append(headlines, fmt.Sprintf("📰 \"%s Teeters on the Brink — Can They Recover?\"", poorest))
	}
	if totalPop > 50000 {
		headlines = append(headlines, fmt.Sprintf("📰 \"Galactic Population Boom: %d Citizens and Counting!\"", totalPop))
	}

	routes := game.GetShippingRoutes()
	totalTrips := 0
	for _, r := range routes {
		totalTrips += r.TripsComplete
	}
	if totalTrips > 100 {
		headlines = append(headlines, fmt.Sprintf("📰 \"%d Trade Deliveries Completed — The Logistics Network Thrives!\"", totalTrips))
	}

	game.LogEvent("intel", "", headlines[rand.Intn(len(headlines))])
}
