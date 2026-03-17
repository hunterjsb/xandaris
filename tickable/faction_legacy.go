package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&FactionLegacySystem{
		BaseSystem: NewBaseSystem("FactionLegacy", 75),
	})
}

// FactionLegacySystem tracks long-term faction accomplishments and
// generates a "legacy score" that reflects overall galactic impact.
// Unlike credits (which fluctuate), legacy only grows.
//
// Legacy sources:
//   - First to colonize a new system: +100 legacy
//   - First to reach tech level milestones: +200 legacy
//   - Completing freight contracts: +50 legacy each
//   - Winning battles: +75 legacy per victory
//   - Building monuments: +500 legacy
//   - Recovering relics: +300 legacy
//   - Golden age on a planet: +25 legacy per interval
//   - Longest trade route: +10 legacy per trip
//
// Legacy milestones grant titles:
//   100:  Pioneer
//   500:  Luminary
//   1000: Titan
//   2500: Legend
//   5000: Eternal
//
// Titles are announced galaxy-wide and displayed on the leaderboard.
type FactionLegacySystem struct {
	*BaseSystem
	legacy     map[string]int    // playerName → legacy score
	titles     map[string]string // playerName → current title
	nextUpdate int64
}

var legacyTitles = []struct {
	threshold int
	title     string
}{
	{5000, "Eternal"},
	{2500, "Legend"},
	{1000, "Titan"},
	{500, "Luminary"},
	{100, "Pioneer"},
}

func (fls *FactionLegacySystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := fls.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if fls.legacy == nil {
		fls.legacy = make(map[string]int)
		fls.titles = make(map[string]string)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}

		fls.accumulateLegacy(player, systems, game)
		fls.checkTitleChange(player, game)
	}
}

func (fls *FactionLegacySystem) accumulateLegacy(player *entities.Player, systems []*entities.System, game GameProvider) {
	points := 0

	// Planet count legacy
	planetCount := 0
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
				planetCount++
				// High-tech planets
				if planet.TechLevel >= 3.0 {
					points += 2
				}
				// Prosperous planets
				if planet.Happiness > 0.8 {
					points += 1
				}
			}
		}
	}
	points += planetCount

	// Fleet size
	shipCount := 0
	for _, ship := range player.OwnedShips {
		if ship != nil {
			shipCount++
		}
	}
	points += shipCount / 5

	// Wealth (logarithmic)
	if player.Credits > 100000 {
		points += 5
	}
	if player.Credits > 1000000 {
		points += 10
	}

	// Shipping routes
	routes := game.GetShippingRoutes()
	for _, route := range routes {
		if route.Owner == player.Name && route.TripsComplete > 0 {
			points += route.TripsComplete
		}
	}

	// Small random factor
	if rand.Intn(3) == 0 {
		points += rand.Intn(3)
	}

	fls.legacy[player.Name] += points
}

func (fls *FactionLegacySystem) checkTitleChange(player *entities.Player, game GameProvider) {
	score := fls.legacy[player.Name]
	currentTitle := fls.titles[player.Name]

	newTitle := ""
	for _, lt := range legacyTitles {
		if score >= lt.threshold {
			newTitle = lt.title
			break
		}
	}

	if newTitle != "" && newTitle != currentTitle {
		fls.titles[player.Name] = newTitle
		game.LogEvent("event", player.Name,
			fmt.Sprintf("👑 %s has earned the title of \"%s\"! Legacy score: %d",
				player.Name, newTitle, score))
	}
}

// GetLegacy returns the legacy score for a faction.
func (fls *FactionLegacySystem) GetLegacy(playerName string) int {
	if fls.legacy == nil {
		return 0
	}
	return fls.legacy[playerName]
}

// GetTitle returns the legacy title for a faction.
func (fls *FactionLegacySystem) GetTitle(playerName string) string {
	if fls.titles == nil {
		return ""
	}
	return fls.titles[playerName]
}
