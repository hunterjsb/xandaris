package tickable

import (
	"github.com/hunterjsb/xandaris/economy"
)

func init() {
	RegisterSystem(&MarketSystem{
		BaseSystem: NewBaseSystem("Market", 25),
	})
}

// MarketSystem drives the economy engine: consumption, price updates, AI trading.
type MarketSystem struct {
	*BaseSystem
}

func (ms *MarketSystem) OnTick(tick int64) {
	ctx := ms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	players := ctx.GetPlayers()

	// Every 10 ticks: consumption + price update (with per-system supply tracking)
	systems := game.GetSystems()
	if tick%10 == 0 {
		result := economy.ProcessConsumption(players, systems)
		for resType, d := range result.Demand {
			market.SetDemand(resType, d)
		}
		market.UpdatePricesWithSystems(players, systems)
	}

	// Every 30 ticks: AI trader (more frequent than before)
	if tick%30 == 0 {
		executor := game.GetTradeExecutor()
		if executor == nil {
			return
		}
		executor.SetTick(tick)
		economy.RunAITrader(executor, players)
	}
}
