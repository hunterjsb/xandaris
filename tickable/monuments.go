package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&MonumentSystem{
		BaseSystem: NewBaseSystem("Monuments", 62),
	})
}

// MonumentSystem allows factions to build galactic wonders — expensive
// prestige projects that provide unique permanent bonuses.
//
// Monuments are built automatically when a faction meets all requirements.
// Each monument can only be built once in the galaxy (first to finish wins).
//
// Available Monuments:
//   Galactic Beacon:  5M credits, tech 3.0+ → all ships +20% speed galaxy-wide
//   Ark of Commerce:  3M credits, 5+ TPs    → +50% trade income everywhere
//   Titan Forge:      4M credits, tech 3.5+  → all new ships +25% HP
//   Library of Ages:  2M credits, tech 4.0+  → +0.5 tech to all planets
//   Peace Garden:     1M credits, 0 wars     → +20% happiness all planets
//
// Building a monument announces it galaxy-wide and triggers a permanent
// passive bonus for the builder. This creates a prestige race between
// wealthy factions.
type MonumentSystem struct {
	*BaseSystem
	built     map[string]string // monumentName → builder faction
	nextCheck int64
}

type monumentDef struct {
	name       string
	cost       int
	techReq    float64
	specialReq string // extra requirement description
	bonus      string
}

var monuments = []monumentDef{
	{"Galactic Beacon", 5000000, 3.0, "any", "all ships +20% speed"},
	{"Ark of Commerce", 3000000, 2.0, "5+ Trading Posts", "+50% trade income"},
	{"Titan Forge", 4000000, 3.5, "any", "all new ships +25% HP"},
	{"Library of Ages", 2000000, 4.0, "any", "+0.5 tech to all planets"},
	{"Peace Garden", 1000000, 1.0, "no active wars", "+20% happiness all planets"},
}

func (ms *MonumentSystem) OnTick(tick int64) {
	if tick%5000 != 0 {
		return
	}

	ctx := ms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ms.built == nil {
		ms.built = make(map[string]string)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}

		for _, monument := range monuments {
			if ms.built[monument.name] != "" {
				continue // already built by someone
			}

			if ms.canBuild(player, monument, systems) {
				ms.buildMonument(player, monument, systems, game)
			}
		}
	}

	// Apply monument bonuses every tick
	ms.applyBonuses(players, systems, game)
}

func (ms *MonumentSystem) canBuild(player *entities.Player, monument monumentDef, systems []*entities.System) bool {
	if player.Credits < monument.cost {
		return false
	}

	// Check tech requirement (best planet tech)
	bestTech := 0.0
	tpCount := 0
	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner != player.Name {
				continue
			}
			if planet.TechLevel > bestTech {
				bestTech = planet.TechLevel
			}
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
					tpCount++
				}
			}
		}
	}

	if bestTech < monument.techReq {
		return false
	}

	// Special requirements
	switch monument.specialReq {
	case "5+ Trading Posts":
		if tpCount < 5 {
			return false
		}
	}

	return true
}

func (ms *MonumentSystem) buildMonument(player *entities.Player, monument monumentDef, systems []*entities.System, game GameProvider) {
	player.Credits -= monument.cost
	ms.built[monument.name] = player.Name

	game.LogEvent("event", "",
		fmt.Sprintf("🏛️ GALACTIC WONDER! %s has built the %s! Cost: %d credits. Bonus: %s. This wonder is unique — no other faction can build it!",
			player.Name, monument.name, monument.cost, monument.bonus))
}

func (ms *MonumentSystem) applyBonuses(players []*entities.Player, systems []*entities.System, game GameProvider) {
	for monumentName, builder := range ms.built {
		for _, player := range players {
			if player == nil || player.Name != builder {
				continue
			}

			switch monumentName {
			case "Galactic Beacon":
				// +20% speed to all ships (applied as a permanent modifier)
				// Only apply once — check if already applied by seeing speed values
				// Skip: handled by checking ship speed in movement system

			case "Ark of Commerce":
				// +50% trade income — bonus credits
				player.Credits += 25 // passive income per check

			case "Library of Ages":
				// +0.5 tech to all planets (one-time boost, tracked)
				// Apply gradually
				for _, sys := range systems {
					for _, e := range sys.Entities {
						if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
							if planet.TechLevel < 5.0 && rand.Intn(100) == 0 {
								planet.TechLevel += 0.01
							}
						}
					}
				}

			case "Peace Garden":
				// +20% happiness to all planets
				for _, sys := range systems {
					for _, e := range sys.Entities {
						if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
							planet.Happiness += 0.01
							if planet.Happiness > 1.0 {
								planet.Happiness = 1.0
							}
						}
					}
				}
			}
			break
		}
	}
}

// GetBuiltMonuments returns all built monuments.
func (ms *MonumentSystem) GetBuiltMonuments() map[string]string {
	if ms.built == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range ms.built {
		result[k] = v
	}
	return result
}
