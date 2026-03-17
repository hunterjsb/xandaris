package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticSummarySystem{
		BaseSystem: NewBaseSystem("GalacticSummary", 152),
	})
}

// GalacticSummarySystem generates a compact galaxy status every
// ~3000 ticks that aggregates the key numbers into one line.
// This is the heartbeat event that shows the game is alive.
//
// Format: "Galaxy: 8 factions | 52 planets | 530 ships | 45K pop |
//          158 routes (42 delivering) | Season: Winter | GMI: 105"
//
// Always visible, always useful, never spammy.
type GalacticSummarySystem struct {
	*BaseSystem
	nextSummary int64
}

func (gss *GalacticSummarySystem) OnTick(tick int64) {
	ctx := gss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if gss.nextSummary == 0 {
		gss.nextSummary = tick + 2000 + int64(rand.Intn(2000))
	}
	if tick < gss.nextSummary {
		return
	}
	gss.nextSummary = tick + 3000 + int64(rand.Intn(2000))

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	routes := game.GetShippingRoutes()

	factionCount := 0
	totalPlanets := 0
	totalShips := 0
	totalPop := int64(0)
	totalCredits := 0

	for _, p := range players {
		if p == nil {
			continue
		}
		factionCount++
		totalShips += len(p.OwnedShips)
		totalCredits += p.Credits
	}

	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				totalPlanets++
				totalPop += planet.Population
			}
		}
	}

	activeRoutes := 0
	delivering := 0
	for _, r := range routes {
		if r.Active {
			activeRoutes++
			if r.TripsComplete > 0 {
				delivering++
			}
		}
	}

	// Season
	cyclePos := tick % 20000
	season := "Spring"
	switch {
	case cyclePos < 5000:
		season = "🌱Spring"
	case cyclePos < 10000:
		season = "☀️Summer"
	case cyclePos < 15000:
		season = "🍂Autumn"
	default:
		season = "❄️Winter"
	}

	// Game time
	hours := tick / 36000
	minutes := (tick % 36000) / 600

	popStr := fmt.Sprintf("%d", totalPop)
	if totalPop > 1000 {
		popStr = fmt.Sprintf("%.1fK", float64(totalPop)/1000)
	}

	creditStr := fmt.Sprintf("%d", totalCredits)
	if totalCredits > 1000000 {
		creditStr = fmt.Sprintf("%.1fM", float64(totalCredits)/1000000)
	} else if totalCredits > 1000 {
		creditStr = fmt.Sprintf("%.0fK", float64(totalCredits)/1000)
	}

	game.LogEvent("intel", "",
		fmt.Sprintf("🌌 Galaxy [%dh%02dm]: %d factions | %d planets | %d ships | %s pop | %scr | %d routes(%d active) | %s",
			hours, minutes, factionCount, totalPlanets, totalShips, popStr, creditStr,
			activeRoutes, delivering, season))
}
