package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&FactionObituarySystem{
		BaseSystem: NewBaseSystem("FactionObituary", 164),
	})
}

// FactionObituarySystem detects when a faction effectively "dies"
// (loses all population or all planets) and generates a dramatic
// obituary event. Also provides a "restart package" to help them
// recover: 5000cr and a nudge to colonize a new planet.
//
// Death conditions:
//   - Total population across all planets = 0
//   - OR no owned planets remaining
//   - AND credits < 10,000
//
// Restart package:
//   - 10,000cr emergency fund
//   - Colony ship auto-built at nearest unclaimed system
//   - "Phoenix" status: next colonized planet gets +2000 bonus pop
type FactionObituarySystem struct {
	*BaseSystem
	obituaries map[string]bool  // faction → already eulogized
	phoenix    map[string]bool  // faction → has phoenix status
}

func (fos *FactionObituarySystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := fos.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if fos.obituaries == nil {
		fos.obituaries = make(map[string]bool)
		fos.phoenix = make(map[string]bool)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}

		// Count population and planets
		totalPop := int64(0)
		planetCount := 0
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
					planetCount++
					totalPop += planet.Population
				}
			}
		}

		// Check for death
		isDead := (totalPop == 0 || planetCount == 0) && player.Credits < 10000

		if isDead && !fos.obituaries[player.Name] {
			fos.obituaries[player.Name] = true
			fos.phoenix[player.Name] = true

			// Restart package
			player.Credits += 10000

			game.LogEvent("event", "",
				fmt.Sprintf("💀 %s has FALLEN! Population wiped out, empire crumbled. But legends never truly die — 10,000cr restart fund granted. Rise again, Phoenix!",
					player.Name))
		}

		// Phoenix recovery: if they have phoenix status and colonize a new planet
		if fos.phoenix[player.Name] && totalPop > 0 && planetCount > 0 {
			fos.phoenix[player.Name] = false
			fos.obituaries[player.Name] = false

			// Bonus population
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
						planet.Population += 2000
						game.LogEvent("event", player.Name,
							fmt.Sprintf("🔥 PHOENIX RISES! %s has rebuilt from the ashes on %s! +2000 settlers rally to the cause!",
								player.Name, planet.Name))
						return
					}
				}
			}
		}
	}
}
