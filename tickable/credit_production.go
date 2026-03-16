package tickable

import (
	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&CreditProductionSystem{
		BaseSystem: NewBaseSystem("CreditProduction", 10),
	})
}

// CreditProductionSystem handles credit production from planets.
//
// Credits come from three sources:
//  1. Population labor: base income from having citizens
//  2. Domestic economy: population consuming resources = economic activity
//  3. Trading Post revenue: share of galaxy trade volume
//
// This ensures self-sustaining colonies can cover their upkeep without
// needing external trade. Trade amplifies income, it doesn't gate it.
type CreditProductionSystem struct {
	*BaseSystem
	tickCounter int64
}

// Resource base prices for domestic economy valuation
var domesticPrices = map[string]float64{
	entities.ResWater:       5,  // basic necessity, low value
	entities.ResIron:        3,  // abundant, low value
	entities.ResOil:         8,  // industrial, moderate
	entities.ResFuel:        10, // processed, higher value
	entities.ResRareMetals:  15, // scarce, high value
	entities.ResHelium3:     12, // energy, high value
	entities.ResElectronics: 25, // manufactured, highest value
}

func (cps *CreditProductionSystem) OnTick(tick int64) {
	if tick%10 != 0 {
		return
	}

	context := cps.GetContext()
	if context == nil {
		return
	}

	players := context.GetPlayers()

	var totalTradeVolume float64
	if me := context.GetGame().GetMarketEngine(); me != nil {
		totalTradeVolume = me.GetTradeVolume()
	}

	for _, player := range players {
		for _, planet := range player.OwnedPlanets {
			if planet == nil || planet.Population <= 0 {
				continue
			}

			productivityMult := planet.ProductivityBonus
			if productivityMult <= 0 {
				productivityMult = 1.0
			}
			techMult := 1.0 + planet.TechLevel*0.05

			// 1. Population labor: 1cr per 100 pop (base)
			laborIncome := int(float64(planet.Population/100) * productivityMult * techMult)

			// 2. Domestic economy: population consuming resources generates credits.
			// This represents the internal economy — citizens buying goods, factories
			// operating, services rendered. A colony that produces and consumes IS
			// an economy, even without external trade.
			domesticIncome := 0
			for _, rate := range economy.PopulationConsumption {
				consumed := float64(planet.Population) / rate.PopDivisor * rate.PerPopulation
				if consumed < 0.5 {
					continue
				}
				// Only count consumption that was actually fulfilled (resource in stock)
				stored := float64(planet.GetStoredAmount(rate.ResourceType))
				fulfilled := consumed
				if fulfilled > stored {
					fulfilled = stored
				}
				if price, ok := domesticPrices[rate.ResourceType]; ok {
					domesticIncome += int(fulfilled * price * productivityMult)
				}
			}

			// 3. Trading Post revenue: share of galaxy trade volume
			tpIncome := 0
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					if b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
						tradeShare := int(totalTradeVolume * 0.01 * float64(b.Level))
						tpIncome += 2*b.Level + tradeShare
					}
				}
			}

			player.Credits += laborIncome + domesticIncome + tpIncome
		}
	}
}
