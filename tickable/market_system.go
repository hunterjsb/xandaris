package tickable

import (
	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
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

	mp, ok := game.(MarketProvider)
	if !ok {
		return
	}
	market := mp.GetMarketEngine()
	if market == nil {
		return
	}

	playersIface := ctx.GetPlayers()
	if playersIface == nil {
		return
	}
	players, ok := playersIface.([]*entities.Player)
	if !ok {
		return
	}

	// Every 10 ticks: consumption + price update (with per-system supply tracking)
	if tick%10 == 0 {
		result := economy.ProcessConsumption(players)
		for resType, d := range result.Demand {
			market.SetDemand(resType, d)
		}
		// Use system-aware price update if systems provider is available
		if sp, ok := game.(SystemsProvider); ok {
			systems := sp.GetSystems()
			market.UpdatePricesWithSystems(players, systems)
		} else {
			market.UpdatePrices(players)
		}
	}

	// Every 30 ticks: AI trader (more frequent than before)
	if tick%30 == 0 {
		ep, ok := game.(ExecutorProvider)
		if !ok {
			return
		}
		executor := ep.GetTradeExecutor()
		if executor == nil {
			return
		}
		executor.SetTick(tick)
		economy.RunAITrader(executor, players)
	}
}

// MarketProvider is implemented by the App to give tickable access to the market.
type MarketProvider interface {
	GetMarketEngine() *economy.Market
}

// ExecutorProvider is implemented by the App to give tickable access to the trade executor.
type ExecutorProvider interface {
	GetTradeExecutor() *economy.TradeExecutor
}

// SystemsProvider is implemented by the App to give access to star systems.
type SystemsProvider interface {
	GetSystems() []*entities.System
}
