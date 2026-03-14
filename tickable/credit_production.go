package tickable

import (
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&CreditProductionSystem{
		BaseSystem: NewBaseSystem("CreditProduction", 10),
	})
}

// CreditProductionSystem handles credit production from planets
type CreditProductionSystem struct {
	*BaseSystem
	tickCounter int64
}

// OnTick processes credit production each tick
func (cps *CreditProductionSystem) OnTick(tick int64) {
	cps.tickCounter++

	interval := int64(cps.GetPriority())
	if interval <= 0 {
		interval = 1
	}

	if cps.tickCounter%interval != 0 {
		return
	}

	context := cps.GetContext()
	if context == nil {
		return
	}

	// Get players from context
	playersInterface := context.GetPlayers()
	if playersInterface == nil {
		return
	}

	players, ok := playersInterface.([]*entities.Player)
	if !ok {
		return
	}

	for _, player := range players {
		for _, planet := range player.OwnedPlanets {
			// Base production: 1 credit per 100 population per interval
			production := int(planet.Population / 100)

			// Trading Post bonus: 3 credits per level per interval
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					if b.BuildingType == "Trading Post" && b.IsOperational {
						production += 3 * b.Level
					}
				}
			}

			player.Credits += production
		}

		// Subsistence income: when broke, population generates minimal credits
		// from barter/labor — scales with population, not a fixed handout.
		if player.Credits < 500 {
			for _, planet := range player.OwnedPlanets {
				subsistence := int(planet.Population / 500) // 1cr per 500 pop
				if subsistence < 1 {
					subsistence = 1
				}
				player.Credits += subsistence
			}
		}
	}
}
