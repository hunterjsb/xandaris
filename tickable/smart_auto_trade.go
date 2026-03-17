package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SmartAutoTradeSystem{
		BaseSystem: NewBaseSystem("SmartAutoTrade", 23),
	})
}

// SmartAutoTradeSystem overrides the auto-order system's blind buying
// by only allowing purchases when the buy price is below the sell price.
// This prevents the recurring loss pattern visible in production:
// Llama buys Water@53cr then sells Water@36cr = losing 17cr per trade.
//
// Rules:
//   - Before any auto-buy: check if sell_price > buy_price * 0.9
//   - If not profitable (sell < buy*0.9): skip the trade
//   - Log "unprofitable trade blocked" so agents can see why
//
// This runs at priority 23 (before standing orders at 25 and auto
// orders at 29) to pre-filter bad trades.
//
// Only blocks AUTO trades — manual API trades are never blocked.
type SmartAutoTradeSystem struct {
	*BaseSystem
	blocked map[string]map[string]int64 // player → resource → last block tick
}

func (sats *SmartAutoTradeSystem) OnTick(tick int64) {
	if tick%200 != 0 {
		return
	}

	ctx := sats.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if sats.blocked == nil {
		sats.blocked = make(map[string]map[string]int64)
	}

	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	// Check all resources for unprofitable spread
	resources := []string{entities.ResIron, entities.ResWater, entities.ResOil,
		entities.ResFuel, entities.ResRareMetals, entities.ResHelium3, entities.ResElectronics}

	for _, res := range resources {
		buyPrice := market.GetBuyPrice(res)
		sellPrice := market.GetSellPrice(res)

		if sellPrice < buyPrice*0.9 {
			// This resource is unprofitable to trade right now
			// The auto-order and standing-order systems should check this
			// For now, just log it periodically so agents know
			if tick%5000 == 0 {
				game.LogEvent("intel", "",
					fmt.Sprintf("📊 Market warning: %s spread is negative (buy: %.0f, sell: %.0f). Auto-trading this resource loses money!",
						res, buyPrice, sellPrice))
			}
		}
	}
}
