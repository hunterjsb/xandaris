package economy

import (
	"github.com/hunterjsb/xandaris/entities"
)

const (
	aiSurplusThreshold  = 0.30 // Sell when above 30% capacity (300 units at cap 1000)
	aiShortageThreshold = 0.10 // Buy when below 10% capacity (100 units at cap 1000)
	aiMaxTradeQty       = 50   // Max units per AI trade
)

// RunAITrader executes trading logic for all AI players using the trade executor.
func RunAITrader(executor *TradeExecutor, players []*entities.Player) {
	for _, player := range players {
		if player == nil || player.IsHuman() {
			continue
		}
		processAIPlayer(executor, player, players)
	}
}

func processAIPlayer(executor *TradeExecutor, player *entities.Player, allPlayers []*entities.Player) {
	for _, planet := range player.OwnedPlanets {
		if planet == nil {
			continue
		}
		for resType, storage := range planet.StoredResources {
			if storage == nil || storage.Capacity <= 0 {
				continue
			}

			ratio := float64(storage.Amount) / float64(storage.Capacity)

			if ratio > aiSurplusThreshold {
				// Sell surplus from THIS planet
				excess := storage.Amount - int(float64(storage.Capacity)*aiSurplusThreshold)
				qty := excess
				if qty > aiMaxTradeQty {
					qty = aiMaxTradeQty
				}
				if qty <= 0 {
					continue
				}

				// Price-aware: sell more aggressively when price is above base
				market := executor.market
				if market != nil {
					sellPrice := market.GetSellPrice(resType)
					basePrice := GetBasePrice(resType)
					if sellPrice > basePrice*1.2 {
						qty = qty * 3 / 2
						if qty > aiMaxTradeQty*2 {
							qty = aiMaxTradeQty * 2
						}
					}
				}

				if _, err := executor.Sell(player, allPlayers, resType, qty, planet); err != nil {
					// Silently skip — resource might not be sellable right now
					_ = err
				}

			} else if ratio < aiShortageThreshold {
				// Buy shortage — deliver to THIS planet
				deficit := int(float64(storage.Capacity)*aiShortageThreshold) - storage.Amount
				qty := deficit
				if qty > aiMaxTradeQty {
					qty = aiMaxTradeQty
				}
				if qty <= 0 {
					continue
				}

				// Price-aware: skip buying if too expensive
				market := executor.market
				if market != nil {
					buyPrice := market.GetBuyPrice(resType)
					basePrice := GetBasePrice(resType)
					if buyPrice > basePrice*2.0 {
						// Too expensive — buy much less
						qty = qty / 5
						if qty <= 0 {
							continue
						}
					} else if buyPrice > basePrice*1.5 {
						qty = qty / 2
						if qty <= 0 {
							continue
						}
					}
				}

				// Check credits
				if player.Credits < 100 {
					continue // Don't trade when broke
				}

				if _, err := executor.Buy(player, allPlayers, resType, qty, planet); err != nil {
					// Normal — might not be enough stock available
					if qty > 10 {
						// Try smaller amount
						executor.Buy(player, allPlayers, resType, 10, planet)
					}
				}
			}
		}

		// Speculative trading: sell when price is very high (even below surplus threshold)
		// This creates more dynamic markets — AI acts as price stabilizer
		if player.Credits > 500 {
			for resType, storage := range planet.StoredResources {
				if storage == nil || storage.Amount < 50 {
					continue // keep minimum buffer
				}
				market := executor.market
				if market == nil {
					continue
				}
				sellPrice := market.GetSellPrice(resType)
				basePrice := GetBasePrice(resType)

				// Sell if price is > 2x base and we have stock to spare
				if sellPrice > basePrice*2.0 && storage.Amount > 100 {
					qty := storage.Amount / 4 // sell 25% of stock
					if qty > aiMaxTradeQty {
						qty = aiMaxTradeQty
					}
					if qty > 0 {
						executor.Sell(player, allPlayers, resType, qty, planet)
					}
				}

				// Buy if price is < 50% of base (bargain) and below surplus threshold
				ratio := float64(storage.Amount) / float64(storage.Capacity)
				if ratio < aiSurplusThreshold {
					buyPrice := market.GetBuyPrice(resType)
					if buyPrice < basePrice*0.5 && player.Credits > int(buyPrice)*20 {
						qty := 20
						if _, err := executor.Buy(player, allPlayers, resType, qty, planet); err != nil {
							_ = err
						}
					}
				}
			}
		}

		// Also try to buy resources that aren't in storage yet
		// (consumption creates demand for resources the planet doesn't produce)
		for _, resType := range []string{"Water", "Iron", "Oil", "Fuel"} {
			if planet.GetStoredAmount(resType) > 0 {
				continue // Already in storage, handled above
			}
			if player.Credits < 200 {
				continue
			}
			// Buy a small amount to get started
			qty := 20
			market := executor.market
			if market != nil {
				buyPrice := market.GetBuyPrice(resType)
				basePrice := GetBasePrice(resType)
				if buyPrice > basePrice*2.5 {
					continue // Way too expensive
				}
			}
			if _, err := executor.Buy(player, allPlayers, resType, qty, planet); err != nil {
				_ = err
			}
		}
	}
}

