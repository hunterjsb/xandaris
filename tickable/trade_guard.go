package tickable

import (
	"fmt"
)

func init() {
	RegisterSystem(&TradeGuardSystem{
		BaseSystem: NewBaseSystem("TradeGuard", 24),
	})
}

// TradeGuardSystem monitors trade patterns and blocks obviously
// unprofitable auto-trades. It prevents the death spiral where
// AI agents repeatedly buy resources at high prices and sell at
// low prices (like Llama buying He-3@164cr and selling@114cr).
//
// Rules:
//   1. If a faction buys resource X and sells resource X within
//      1000 ticks, and the sell price < buy price, log a warning
//   2. If the same losing pattern repeats 3+ times, block auto-trades
//      of that resource for 5000 ticks
//   3. Never block manual (API) trades — only standing orders and
//      auto-order system trades
//
// This runs at priority 24 (before standing orders at 25 and
// auto orders at 29) so it can intervene before bad trades execute.
type TradeGuardSystem struct {
	*BaseSystem
	recentTrades map[string][]tradeEntry // playerName → recent trades
	blocked      map[string]map[string]int64 // playerName → resource → unblock tick
	lossCount    map[string]map[string]int   // playerName → resource → consecutive losses
}

type tradeEntry struct {
	resource string
	action   string // "buy" or "sell"
	price    float64
	tick     int64
}

func (tgs *TradeGuardSystem) OnTick(tick int64) {
	if tick%200 != 0 {
		return
	}

	ctx := tgs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tgs.recentTrades == nil {
		tgs.recentTrades = make(map[string][]tradeEntry)
		tgs.blocked = make(map[string]map[string]int64)
		tgs.lossCount = make(map[string]map[string]int)
	}

	te := game.GetTradeExecutor()
	if te == nil {
		return
	}

	// Analyze recent trade history for losing patterns
	history := te.GetHistory(50)
	for _, record := range history {
		if record.Tick < tick-2000 {
			continue // only recent trades
		}

		playerTrades := tgs.recentTrades[record.Player]

		// Check for buy-then-sell-at-loss
		if record.Action == "sell" {
			for _, prev := range playerTrades {
				if prev.resource == record.Resource && prev.action == "buy" &&
					record.Tick-prev.tick < 1000 {
					if record.UnitPrice < prev.price {
						// LOSING TRADE DETECTED
						loss := prev.price - record.UnitPrice
						if tgs.lossCount[record.Player] == nil {
							tgs.lossCount[record.Player] = make(map[string]int)
						}
						tgs.lossCount[record.Player][record.Resource]++
						count := tgs.lossCount[record.Player][record.Resource]

						if count >= 3 {
							// Block this resource for this player
							if tgs.blocked[record.Player] == nil {
								tgs.blocked[record.Player] = make(map[string]int64)
							}
							tgs.blocked[record.Player][record.Resource] = tick + 5000

							game.LogEvent("alert", record.Player,
								fmt.Sprintf("🛑 Trade guard: %s auto-trading of %s BLOCKED! Detected %d consecutive losing trades (buying@%.0f, selling@%.0f, loss: %.0f/unit). Review strategy!",
									record.Player, record.Resource, count, prev.price, record.UnitPrice, loss))

							tgs.lossCount[record.Player][record.Resource] = 0
						}
					}
				}
			}
		}

		// Track this trade
		tgs.recentTrades[record.Player] = append(tgs.recentTrades[record.Player], tradeEntry{
			resource: record.Resource,
			action:   record.Action,
			price:    record.UnitPrice,
			tick:     record.Tick,
		})

		// Keep only last 20 trades per player
		if len(tgs.recentTrades[record.Player]) > 20 {
			tgs.recentTrades[record.Player] = tgs.recentTrades[record.Player][1:]
		}
	}

	// Expire blocks
	for player, blocks := range tgs.blocked {
		for res, unblockTick := range blocks {
			if tick >= unblockTick {
				delete(tgs.blocked[player], res)
				game.LogEvent("logistics", player,
					fmt.Sprintf("✅ Trade guard: %s auto-trading of %s unblocked. Trade carefully!",
						player, res))
			}
		}
	}
}

// IsBlocked checks if auto-trading of a resource is blocked for a player.
func (tgs *TradeGuardSystem) IsBlocked(playerName, resource string, tick int64) bool {
	if tgs.blocked == nil {
		return false
	}
	blocks := tgs.blocked[playerName]
	if blocks == nil {
		return false
	}
	unblockTick, exists := blocks[resource]
	return exists && tick < unblockTick
}
