package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AchievementSystem{
		BaseSystem: NewBaseSystem("Achievements", 46),
	})
}

// AchievementSystem tracks milestones and grants permanent bonuses.
// Achievements are checked every 1000 ticks and announced when earned.
type AchievementSystem struct {
	*BaseSystem
	earned map[string]map[string]bool // player → achievement → earned
}

type achievement struct {
	name   string
	desc   string
	check  func(p *entities.Player, systems []*entities.System) bool
	reward string
}

var achievements = []achievement{
	{
		name: "First Steps",
		desc: "Own your first planet",
		check: func(p *entities.Player, systems []*entities.System) bool {
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if pl, ok := e.(*entities.Planet); ok && pl.Owner == p.Name {
						return true
					}
				}
			}
			return false
		},
		reward: "+500 starting credits",
	},
	{
		name: "Merchant Prince",
		desc: "Accumulate 100,000 credits",
		check: func(p *entities.Player, _ []*entities.System) bool {
			return p.Credits >= 100_000
		},
		reward: "+5% trade revenue",
	},
	{
		name: "Industrial Giant",
		desc: "Own 5+ mines across all planets",
		check: func(p *entities.Player, systems []*entities.System) bool {
			mines := 0
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if pl, ok := e.(*entities.Planet); ok && pl.Owner == p.Name {
						for _, be := range pl.Buildings {
							if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingMine {
								mines++
							}
						}
					}
				}
			}
			return mines >= 5
		},
		reward: "+10% mine output",
	},
	{
		name: "Empire Builder",
		desc: "Control 5 planets",
		check: func(p *entities.Player, systems []*entities.System) bool {
			count := 0
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if pl, ok := e.(*entities.Planet); ok && pl.Owner == p.Name {
						count++
					}
				}
			}
			return count >= 5
		},
		reward: "+15% colony growth",
	},
	{
		name: "Tycoon",
		desc: "Accumulate 1,000,000 credits",
		check: func(p *entities.Player, _ []*entities.System) bool {
			return p.Credits >= 1_000_000
		},
		reward: "+10% all income",
	},
	{
		name: "Galactic Power",
		desc: "Control 10 planets",
		check: func(p *entities.Player, systems []*entities.System) bool {
			count := 0
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if pl, ok := e.(*entities.Planet); ok && pl.Owner == p.Name {
						count++
					}
				}
			}
			return count >= 10
		},
		reward: "+20% all production",
	},
	{
		name: "Tech Pioneer",
		desc: "Reach tech level 3.0",
		check: func(p *entities.Player, systems []*entities.System) bool {
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if pl, ok := e.(*entities.Planet); ok && pl.Owner == p.Name && pl.TechLevel >= 3.0 {
						return true
					}
				}
			}
			return false
		},
		reward: "+25% research speed",
	},
	{
		name: "Diverse Economy",
		desc: "Stock all 7 resource types on one planet",
		check: func(p *entities.Player, systems []*entities.System) bool {
			resources := []string{"Water", "Iron", "Oil", "Fuel", "Rare Metals", "Helium-3", "Electronics"}
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if pl, ok := e.(*entities.Planet); ok && pl.Owner == p.Name {
						all := true
						for _, r := range resources {
							if pl.GetStoredAmount(r) <= 0 {
								all = false
								break
							}
						}
						if all {
							return true
						}
					}
				}
			}
			return false
		},
		reward: "Permanent 3x diversity bonus",
	},
}

func (as *AchievementSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := as.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if as.earned == nil {
		as.earned = make(map[string]map[string]bool)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}

		if as.earned[player.Name] == nil {
			as.earned[player.Name] = make(map[string]bool)
		}

		for _, ach := range achievements {
			if as.earned[player.Name][ach.name] {
				continue
			}

			if ach.check(player, systems) {
				as.earned[player.Name][ach.name] = true
				game.LogEvent("achievement", player.Name,
					fmt.Sprintf("🏅 %s earned: %s — \"%s\" (Reward: %s)",
						player.Name, ach.name, ach.desc, ach.reward))
			}
		}
	}
}
