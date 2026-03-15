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

	players := context.GetPlayers()

	// Get trade volume from market for Trading Post revenue
	var totalTradeVolume float64
	if me := context.GetGame().GetMarketEngine(); me != nil {
		totalTradeVolume = me.GetTradeVolume()
	}

	for _, player := range players {
		for _, planet := range player.OwnedPlanets {
			// Base production: 1 credit per 100 population per interval, scaled by happiness
			productivityMult := planet.ProductivityBonus
			if productivityMult <= 0 {
				productivityMult = 1.0
			}
			production := int(float64(planet.Population/100) * productivityMult)

			// Trading Post revenue: earns from galaxy trade volume
			// Each TP level captures a share of total trade activity
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					if b.BuildingType == "Trading Post" && b.IsOperational {
						// Base: 2cr per level + share of trade volume
						tradeShare := int(totalTradeVolume * 0.01 * float64(b.Level))
						production += 2*b.Level + tradeShare
					}
				}
			}

			player.Credits += production
		}

		// No subsistence bailout — credits must come from population labor
		// (1cr per 100 pop) or Trading Post trade revenue. If credits hit 0,
		// buildings shut down (consumption.go) and drain stops, allowing
		// natural recovery from population-based income.
	}
}
