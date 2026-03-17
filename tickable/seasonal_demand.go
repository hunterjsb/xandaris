package tickable

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SeasonalDemandSystem{
		BaseSystem: NewBaseSystem("SeasonalDemand", 93),
	})
}

// SeasonalDemandSystem creates predictable demand cycles for resources.
// Unlike random events, seasons are PREDICTABLE — smart factions can
// stockpile before demand spikes and sell at the peak.
//
// Seasons (20,000 tick cycle):
//   Spring (ticks 0-5000):     Water demand +50%, population boom
//   Summer (ticks 5000-10000): Fuel demand +30%, ship activity peaks
//   Autumn (ticks 10000-15000): Iron/RM demand +40%, building season
//   Winter (ticks 15000-20000): Electronics demand +30%, research focus
//
// During each season:
//   - The in-demand resource's price naturally rises (demand pressure)
//   - Factions who supply the seasonal resource earn bonus credits
//   - Seasonal events fire (spring festivals, summer expeditions, etc.)
//
// Seasons are announced 2000 ticks before they change, giving factions
// time to prepare logistics.
type SeasonalDemandSystem struct {
	*BaseSystem
	lastSeason string
}

func (sds *SeasonalDemandSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := sds.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	// Calculate current season (20,000 tick cycle)
	cyclePos := tick % 20000
	season := sds.getSeason(cyclePos)

	// Announce season changes
	if season != sds.lastSeason && sds.lastSeason != "" {
		sds.announceSeason(season, game)
	}
	sds.lastSeason = season

	// Apply seasonal effects
	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	systems := game.GetSystems()

	switch season {
	case "Spring":
		// Water demand surge
		market.AddTradeVolume(entities.ResWater, 20, true)
		// Population bonus
		if rand.Intn(5) == 0 {
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
						cap := planet.GetTotalPopulationCapacity()
						if cap > 0 && planet.Population < cap {
							planet.Population += 50
						}
					}
				}
			}
		}

	case "Summer":
		// Fuel demand for expeditions
		market.AddTradeVolume(entities.ResFuel, 15, true)

	case "Autumn":
		// Iron + RM demand for construction
		market.AddTradeVolume(entities.ResIron, 15, true)
		market.AddTradeVolume(entities.ResRareMetals, 10, true)

	case "Winter":
		// Electronics demand for research
		market.AddTradeVolume(entities.ResElectronics, 10, true)
	}

	// Pre-announce next season 2000 ticks before change
	nextSeasonIn := sds.ticksUntilSeasonChange(cyclePos)
	if nextSeasonIn <= 2000 && nextSeasonIn > 1500 {
		nextSeason := sds.getNextSeason(season)
		resource := sds.seasonalResource(nextSeason)
		game.LogEvent("intel", "",
			fmt.Sprintf("🗓️ Season change approaching! %s begins in ~%d minutes — %s demand will surge. Stockpile now!",
				nextSeason, nextSeasonIn/600, resource))
	}
}

func (sds *SeasonalDemandSystem) getSeason(cyclePos int64) string {
	switch {
	case cyclePos < 5000:
		return "Spring"
	case cyclePos < 10000:
		return "Summer"
	case cyclePos < 15000:
		return "Autumn"
	default:
		return "Winter"
	}
}

func (sds *SeasonalDemandSystem) getNextSeason(current string) string {
	switch current {
	case "Spring":
		return "Summer"
	case "Summer":
		return "Autumn"
	case "Autumn":
		return "Winter"
	default:
		return "Spring"
	}
}

func (sds *SeasonalDemandSystem) ticksUntilSeasonChange(cyclePos int64) int64 {
	boundaries := []int64{5000, 10000, 15000, 20000}
	for _, b := range boundaries {
		if cyclePos < b {
			return b - cyclePos
		}
	}
	return 20000 - cyclePos
}

func (sds *SeasonalDemandSystem) seasonalResource(season string) string {
	switch season {
	case "Spring":
		return "Water"
	case "Summer":
		return "Fuel"
	case "Autumn":
		return "Iron & Rare Metals"
	case "Winter":
		return "Electronics"
	default:
		return "various"
	}
}

func (sds *SeasonalDemandSystem) announceSeason(season string, game GameProvider) {
	resource := sds.seasonalResource(season)
	emoji := "🌱"
	switch season {
	case "Summer":
		emoji = "☀️"
	case "Autumn":
		emoji = "🍂"
	case "Winter":
		emoji = "❄️"
	}

	game.LogEvent("event", "",
		fmt.Sprintf("%s %s has arrived! %s demand surging. Factions with stockpiles will profit!",
			emoji, season, resource))
}

// Suppress unused import warning
var _ = math.Pi
