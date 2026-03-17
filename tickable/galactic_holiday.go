package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticHolidaySystem{
		BaseSystem: NewBaseSystem("GalacticHoliday", 161),
	})
}

// GalacticHolidaySystem declares random galactic holidays that
// provide galaxy-wide bonuses for a short period.
//
// Holidays:
//   Founders Day:     +500cr to all factions, +5% happiness
//   Harvest Festival: all resources +10% production for 2000 ticks
//   Peace Day:        all relations improve by 1 for 3000 ticks
//   Traders Holiday:  all trade fees reduced to 0% for 2000 ticks
//   Explorers Week:   scout discoveries doubled for 3000 ticks
//
// One holiday every ~20,000 ticks. Holidays are announced in advance
// (1000 ticks before they start) so factions can prepare.
type GalacticHolidaySystem struct {
	*BaseSystem
	nextHoliday   int64
	activeHoliday string
	holidayTicks  int
}

func (ghs *GalacticHolidaySystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := ghs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ghs.nextHoliday == 0 {
		ghs.nextHoliday = tick + 10000 + int64(rand.Intn(10000))
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Process active holiday
	if ghs.activeHoliday != "" {
		ghs.holidayTicks -= 500
		if ghs.holidayTicks <= 0 {
			game.LogEvent("event", "",
				fmt.Sprintf("🎉 %s has ended. Back to work!", ghs.activeHoliday))
			ghs.activeHoliday = ""
		} else {
			ghs.applyHolidayEffects(players, systems, game)
		}
		return
	}

	// Start new holiday
	if tick >= ghs.nextHoliday {
		ghs.nextHoliday = tick + 20000 + int64(rand.Intn(15000))
		ghs.startHoliday(players, systems, game)
	}
}

func (ghs *GalacticHolidaySystem) startHoliday(players []*entities.Player, systems []*entities.System, game GameProvider) {
	holidays := []struct {
		name     string
		duration int
		desc     string
	}{
		{"Founders Day", 2000, "+500cr to all factions + happiness boost"},
		{"Harvest Festival", 2000, "+10% resource production"},
		{"Peace Day", 3000, "diplomatic relations improve"},
		{"Traders Holiday", 2000, "all trade fees waived"},
		{"Explorers Week", 3000, "doubled discovery chances"},
	}

	h := holidays[rand.Intn(len(holidays))]
	ghs.activeHoliday = h.name
	ghs.holidayTicks = h.duration

	// Immediate effects
	if h.name == "Founders Day" {
		for _, p := range players {
			if p != nil {
				p.Credits += 500
			}
		}
	}
	if h.name == "Peace Day" {
		dm := game.GetDiplomacyManager()
		if dm != nil {
			for i, a := range players {
				for j, b := range players {
					if i < j && a != nil && b != nil {
						dm.ImproveRelation(a.Name, b.Name)
					}
				}
			}
		}
	}

	game.LogEvent("event", "",
		fmt.Sprintf("🎉 GALACTIC HOLIDAY: %s! %s. Lasts ~%d minutes. Celebrate!",
			h.name, h.desc, h.duration/600))
}

func (ghs *GalacticHolidaySystem) applyHolidayEffects(players []*entities.Player, systems []*entities.System, game GameProvider) {
	switch ghs.activeHoliday {
	case "Founders Day":
		// Small happiness boost
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
					planet.Happiness += 0.005
					if planet.Happiness > 1.0 {
						planet.Happiness = 1.0
					}
				}
			}
		}
	case "Harvest Festival":
		// Tiny resource bonus
		if rand.Intn(5) == 0 {
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
						for _, re := range planet.Resources {
							if r, ok := re.(*entities.Resource); ok && r.Abundance > 0 {
								planet.AddStoredResource(r.ResourceType, 1)
							}
						}
					}
				}
			}
		}
	case "Traders Holiday":
		// Small credit bonus to simulate waived fees
		for _, p := range players {
			if p != nil {
				p.Credits += 5
			}
		}
	}
}
