package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&HappinessInterventionSystem{
		BaseSystem: NewBaseSystem("HappinessIntervention", 133),
	})
}

// HappinessInterventionSystem takes emergency action on planets
// with critically low happiness (<15%) to prevent rebellion and
// population collapse.
//
// Interventions:
//   1. Emergency Water ration: if Water is 0 and happiness <15%,
//      inject 10 Water (humanitarian aid)
//   2. Emergency entertainment: spend 500cr to boost happiness +5%
//      (bread and circuses)
//   3. Building repair: if happiness is low because buildings are
//      offline, auto-repair the cheapest one
//
// Only triggers when happiness is truly critical. Normal unhappiness
// is the player's problem to solve.
type HappinessInterventionSystem struct {
	*BaseSystem
	lastIntervention map[int]int64
}

func (his *HappinessInterventionSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := his.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if his.lastIntervention == nil {
		his.lastIntervention = make(map[int]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.Population < 500 {
				continue
			}

			if planet.Happiness >= 0.15 {
				continue
			}

			pid := planet.GetID()
			if tick-his.lastIntervention[pid] < 3000 {
				continue
			}

			// Find player
			var player *entities.Player
			for _, p := range players {
				if p != nil && p.Name == planet.Owner {
					player = p
					break
				}
			}
			if player == nil {
				continue
			}

			his.lastIntervention[pid] = tick
			intervened := false

			// Emergency Water
			if planet.GetStoredAmount(entities.ResWater) == 0 {
				planet.AddStoredResource(entities.ResWater, 10)
				intervened = true
			}

			// Repair a broken building
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && !b.IsOperational {
					b.IsOperational = true
					intervened = true
					break // repair one at a time
				}
			}

			// Emergency entertainment (spend credits for happiness)
			if player.Credits > 1000 && planet.Happiness < 0.10 {
				player.Credits -= 500
				planet.Happiness += 0.05
				intervened = true
			}

			if intervened {
				game.LogEvent("alert", planet.Owner,
					fmt.Sprintf("🆘 Emergency intervention on %s (%.0f%% happy, %d pop). Humanitarian aid deployed!",
						planet.Name, planet.Happiness*100, planet.Population))
			}
		}
	}
}
