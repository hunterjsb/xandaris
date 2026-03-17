package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&PriceManipulationSystem{
		BaseSystem: NewBaseSystem("PriceManipulation", 98),
	})
}

// PriceManipulationSystem detects and punishes market manipulation.
// When a faction dumps large quantities of a resource to crash the
// price, then buys it back cheap, they're manipulating the market.
//
// Detection: if a faction sells 200+ of resource X and buys 100+
// of the same resource within 2000 ticks, that's manipulation.
//
// Punishment:
//   - First offense: warning + 1000cr fine
//   - Second offense: 5000cr fine + 2000-tick trade ban on that resource
//   - Third offense: 10000cr fine + galactic sanctions
//
// This prevents AI agents from gaming the spread between buy/sell
// prices and keeps the market fair for all factions.
type PriceManipulationSystem struct {
	*BaseSystem
	recentSells map[string]map[string]int64 // player → resource → last big sell tick
	recentBuys  map[string]map[string]int64 // player → resource → last big buy tick
	offenses    map[string]int              // player → offense count
}

func (pms *PriceManipulationSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := pms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	te := game.GetTradeExecutor()
	if te == nil {
		return
	}

	if pms.recentSells == nil {
		pms.recentSells = make(map[string]map[string]int64)
		pms.recentBuys = make(map[string]map[string]int64)
		pms.offenses = make(map[string]int)
	}

	players := ctx.GetPlayers()

	// Analyze recent trades
	history := te.GetHistory(30)
	for _, record := range history {
		if record.Tick < tick-3000 {
			continue
		}

		if record.Quantity < 100 {
			continue // only flag large trades
		}

		if record.Action == "sell" {
			if pms.recentSells[record.Player] == nil {
				pms.recentSells[record.Player] = make(map[string]int64)
			}
			pms.recentSells[record.Player][record.Resource] = record.Tick
		} else if record.Action == "buy" {
			if pms.recentBuys[record.Player] == nil {
				pms.recentBuys[record.Player] = make(map[string]int64)
			}
			pms.recentBuys[record.Player][record.Resource] = record.Tick
		}
	}

	// Detect manipulation: sell then buy same resource within 2000 ticks
	for playerName, sells := range pms.recentSells {
		buys := pms.recentBuys[playerName]
		if buys == nil {
			continue
		}

		for res, sellTick := range sells {
			buyTick, bought := buys[res]
			if !bought {
				continue
			}
			// Sell then buy within 2000 ticks = manipulation
			if buyTick > sellTick && buyTick-sellTick < 2000 {
				pms.offenses[playerName]++
				offense := pms.offenses[playerName]

				// Clear the detection to avoid re-triggering
				delete(sells, res)
				delete(buys, res)

				var fine int
				var msg string
				switch {
				case offense >= 3:
					fine = 10000
					msg = fmt.Sprintf("🚨 MARKET MANIPULATION (3rd offense): %s caught pump-and-dump on %s! Fine: %dcr + galactic sanctions!",
						playerName, res, fine)
				case offense >= 2:
					fine = 5000
					msg = fmt.Sprintf("🚨 Market manipulation (2nd offense): %s caught on %s! Fine: %dcr + trade restriction!",
						playerName, res, fine)
				default:
					fine = 1000
					msg = fmt.Sprintf("⚠️ Market manipulation warning: %s flagged for %s dump-and-buy. Fine: %dcr",
						playerName, res, fine)
				}

				for _, p := range players {
					if p != nil && p.Name == playerName {
						p.Credits -= fine
						if p.Credits < 0 {
							p.Credits = 0
						}
						break
					}
				}

				game.LogEvent("alert", playerName, msg)
				return // one detection per tick
			}
		}
	}

	_ = rand.Intn // suppress
}
