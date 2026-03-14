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

		// Minimum credit floor: players below 1000cr get a small subsidy.
		// Prevents permanent poverty spirals where AI can never invest.
		if player.Credits < 1000 {
			player.Credits += 5
		}
	}
}
