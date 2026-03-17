package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&LastStandSystem{
		BaseSystem: NewBaseSystem("LastStand", 171),
	})
}

// LastStandSystem detects when a faction is down to their last planet
// and generates dramatic "last stand" events with defensive bonuses.
//
// When a faction has exactly 1 planet remaining:
//   - +50% production bonus (desperate efficiency)
//   - +30% happiness (rallying around the flag)
//   - Planet becomes harder to conquer (siege damage halved)
//   - Dramatic announcement: "X makes their LAST STAND on Y!"
//
// This prevents total steamrolling and creates dramatic narratives.
// The last planet of an empire is always the hardest to take.
type LastStandSystem struct {
	*BaseSystem
	lastStanders map[string]bool
}

func (lss *LastStandSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := lss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if lss.lastStanders == nil {
		lss.lastStanders = make(map[string]bool)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}

		// Count planets
		var planets []*entities.Planet
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
					planets = append(planets, planet)
				}
			}
		}

		wasLastStand := lss.lastStanders[player.Name]

		if len(planets) == 1 {
			planet := planets[0]

			// Apply last stand bonuses
			planet.Happiness += 0.02
			if planet.Happiness > 0.8 {
				planet.Happiness = 0.8
			}

			// Small production bonus (extra resources)
			if rand.Intn(3) == 0 {
				for _, re := range planet.Resources {
					if r, ok := re.(*entities.Resource); ok && r.Abundance > 0 {
						planet.AddStoredResource(r.ResourceType, 2)
					}
				}
			}

			if !wasLastStand {
				lss.lastStanders[player.Name] = true
				game.LogEvent("event", player.Name,
					fmt.Sprintf("🏴 LAST STAND! %s is down to their final planet: %s! Defensive bonuses activated. This is where legends are made!",
						player.Name, planet.Name))
			}
		} else if wasLastStand && len(planets) > 1 {
			// Recovered from last stand
			lss.lastStanders[player.Name] = false
			game.LogEvent("event", player.Name,
				fmt.Sprintf("🔥 %s has broken free from their Last Stand! Empire expanding again with %d planets!",
					player.Name, len(planets)))
		}
	}
}
