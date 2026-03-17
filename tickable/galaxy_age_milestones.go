package tickable

import (
	"fmt"
)

func init() {
	RegisterSystem(&GalaxyAgeMilestoneSystem{
		BaseSystem: NewBaseSystem("GalaxyAgeMilestones", 144),
	})
}

// GalaxyAgeMilestoneSystem marks the passage of galactic time with
// milestone celebrations. Every significant time threshold triggers
// an announcement and small galaxy-wide bonus.
//
// Milestones:
//   1 hour:    "The galaxy's first hour" — +100cr to all factions
//   2 hours:   "A young galaxy" — +200cr to all
//   6 hours:   "The galaxy matures" — +500cr to all
//   12 hours:  "Half a day of civilization" — +1000cr to all
//   24 hours:  "One full day!" — +2000cr to all, galactic celebration
//   48 hours:  "A veteran galaxy" — +5000cr to all
//
// Creates a sense of shared history and rewards persistence.
type GalaxyAgeMilestoneSystem struct {
	*BaseSystem
	announced map[int64]bool
}

var ageMilestones = []struct {
	ticks   int64
	label   string
	bonus   int
}{
	{36000, "1 hour", 100},
	{72000, "2 hours", 200},
	{216000, "6 hours", 500},
	{432000, "12 hours", 1000},
	{864000, "24 hours", 2000},
	{1728000, "48 hours", 5000},
}

func (gams *GalaxyAgeMilestoneSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := gams.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if gams.announced == nil {
		gams.announced = make(map[int64]bool)
	}

	for _, m := range ageMilestones {
		if tick >= m.ticks && !gams.announced[m.ticks] {
			gams.announced[m.ticks] = true

			players := game.GetPlayers()
			for _, p := range players {
				if p != nil {
					p.Credits += m.bonus
				}
			}

			game.LogEvent("event", "",
				fmt.Sprintf("🎂 GALAXY MILESTONE: %s of civilization! All factions receive +%dcr celebration bonus!",
					m.label, m.bonus))
		}
	}
}
