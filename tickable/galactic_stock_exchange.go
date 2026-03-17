package tickable

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticStockExchangeSystem{
		BaseSystem: NewBaseSystem("GalacticStockExchange", 154),
	})
}

// GalacticStockExchangeSystem creates faction "stocks" that other
// factions can invest in. Each faction's stock price is based on
// their economic performance.
//
// Stock price = (credits/1000) + (planets*50) + (pop/100) + (ships*5)
//
// Every 5000 ticks, stock prices update and dividends are paid:
//   - Shareholders earn 1% of the stock's price change as dividends
//   - If stock goes up: shareholders profit
//   - If stock goes down: shareholders lose value (but never credits)
//
// Factions auto-buy stocks of their allies (diplomatic investment).
// Creates financial interconnection between friendly factions.
type GalacticStockExchangeSystem struct {
	*BaseSystem
	stockPrices map[string]float64            // faction → current price
	holdings    map[string]map[string]float64  // investor → faction → shares
	nextTick    int64
}

func (gses *GalacticStockExchangeSystem) OnTick(tick int64) {
	if tick%3000 != 0 {
		return
	}

	ctx := gses.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	dm := game.GetDiplomacyManager()

	if gses.stockPrices == nil {
		gses.stockPrices = make(map[string]float64)
		gses.holdings = make(map[string]map[string]float64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Update stock prices
	for _, p := range players {
		if p == nil {
			continue
		}

		price := float64(p.Credits) / 1000
		price += float64(len(p.OwnedShips)) * 5

		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == p.Name {
					price += 50 + float64(planet.Population)/100
				}
			}
		}

		oldPrice := gses.stockPrices[p.Name]
		gses.stockPrices[p.Name] = price

		// Pay dividends on price increase
		if oldPrice > 0 {
			change := price - oldPrice
			for investor, holdings := range gses.holdings {
				shares := holdings[p.Name]
				if shares <= 0 {
					continue
				}
				dividend := int(change * shares * 0.01)
				if dividend > 0 {
					for _, ip := range players {
						if ip != nil && ip.Name == investor {
							ip.Credits += dividend
							break
						}
					}
				}
			}
		}
	}

	// Auto-invest in allied factions
	if dm == nil {
		return
	}

	if gses.nextTick == 0 {
		gses.nextTick = tick + 10000
	}
	if tick < gses.nextTick {
		return
	}
	gses.nextTick = tick + 10000 + int64(rand.Intn(5000))

	for _, investor := range players {
		if investor == nil || investor.Credits < 50000 {
			continue
		}

		for _, target := range players {
			if target == nil || target.Name == investor.Name {
				continue
			}

			rel := dm.GetRelation(investor.Name, target.Name)
			if rel < 1 {
				continue
			}

			// Buy shares worth 1% of credits
			investment := investor.Credits / 100
			if investment > 5000 {
				investment = 5000
			}
			if investment < 500 {
				continue
			}

			investor.Credits -= investment
			if gses.holdings[investor.Name] == nil {
				gses.holdings[investor.Name] = make(map[string]float64)
			}

			price := gses.stockPrices[target.Name]
			if price <= 0 {
				price = 1
			}
			shares := float64(investment) / price
			gses.holdings[investor.Name][target.Name] += shares

			if rand.Intn(3) == 0 {
				game.LogEvent("trade", investor.Name,
					fmt.Sprintf("📈 %s bought %.1f shares of %s stock for %dcr (price: %.0f)",
						investor.Name, shares, target.Name, investment, price))
			}
			break // one investment per tick
		}
	}

	// Report top stocks
	if rand.Intn(5) == 0 {
		bestStock := ""
		bestPrice := 0.0
		for name, price := range gses.stockPrices {
			if price > bestPrice {
				bestPrice = price
				bestStock = name
			}
		}
		if bestStock != "" {
			game.LogEvent("intel", "",
				fmt.Sprintf("📈 Stock Exchange: Top stock: %s (%.0f). Invest in allied factions for dividends!",
					bestStock, bestPrice))
		}
	}

	_ = math.Abs
}
