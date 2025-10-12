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

	// Only produce credits every 10 ticks (once per second at 1x speed)
	if cps.tickCounter%10 != 0 {
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
		// Each planet produces credits based on population
		for _, planet := range player.OwnedPlanets {
			// Base production: 1 credit per million population per second
			production := int(planet.Population / 1000000)

			// Bonus for habitability
			habitabilityBonus := float64(planet.Habitability) / 100.0
			production = int(float64(production) * (1.0 + habitabilityBonus))

			// Add production to player credits
			player.Credits += production
		}
	}
}
