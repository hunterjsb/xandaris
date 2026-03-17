package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&CommodityFuturesSystem{
		BaseSystem: NewBaseSystem("CommodityFutures", 68),
	})
}

// CommodityFuturesSystem lets factions speculate on resource prices.
// A futures contract locks in today's price for delivery later.
//
// Every ~5000 ticks, the system generates futures opportunities:
//   "Iron futures at 15cr/unit for delivery in 8000 ticks"
//
// A faction can buy a futures contract:
//   - Pay a deposit (10% of contract value)
//   - If at maturity the market price is HIGHER → profit (price diff * qty)
//   - If at maturity the market price is LOWER → loss (deposit forfeited)
//
// This creates a financial layer:
//   - Factions who understand supply/demand cycles can profit
//   - Hedging: lock in prices before a predicted scarcity
//   - Risk: if you're wrong about the market direction, lose deposit
//
// Futures contracts are auto-resolved at maturity. No physical delivery.
type CommodityFuturesSystem struct {
	*BaseSystem
	futures    []*FuturesContract
	nextOffer  int64
}

// FuturesContract represents a price bet.
type FuturesContract struct {
	ID           int
	Resource     string
	Quantity     int
	LockedPrice  float64 // price at time of purchase
	Deposit      int
	MaturityTick int64
	Buyer        string
	Settled      bool
	Profit       int // positive = profit, negative = loss
}

func (cfs *CommodityFuturesSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := cfs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if cfs.nextOffer == 0 {
		cfs.nextOffer = tick + 3000 + int64(rand.Intn(5000))
	}

	players := ctx.GetPlayers()
	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	// Settle mature contracts
	for _, f := range cfs.futures {
		if f.Settled || f.Buyer == "" {
			continue
		}
		if tick < f.MaturityTick {
			continue
		}

		// Settlement: compare current price to locked price
		currentPrice := market.GetSellPrice(f.Resource)
		priceDiff := currentPrice - f.LockedPrice
		f.Profit = int(priceDiff * float64(f.Quantity))
		f.Settled = true

		for _, p := range players {
			if p == nil || p.Name != f.Buyer {
				continue
			}

			if f.Profit > 0 {
				// Winner: return deposit + profit
				p.Credits += f.Deposit + f.Profit
				game.LogEvent("trade", f.Buyer,
					fmt.Sprintf("📈 Futures contract settled! %s %s — price went from %.0f to %.0f. Profit: %dcr (+%dcr deposit)",
						f.Resource, f.Buyer, f.LockedPrice, currentPrice, f.Profit, f.Deposit))
			} else {
				// Loser: deposit forfeited
				game.LogEvent("trade", f.Buyer,
					fmt.Sprintf("📉 Futures contract expired! %s %s — price went from %.0f to %.0f. Lost %dcr deposit",
						f.Resource, f.Buyer, f.LockedPrice, currentPrice, f.Deposit))
			}
			break
		}
	}

	// Generate new futures offerings
	if tick >= cfs.nextOffer {
		cfs.nextOffer = tick + 5000 + int64(rand.Intn(8000))
		cfs.generateOffering(tick, market, game)
	}

	// Auto-buy for AI factions that can afford it
	for _, f := range cfs.futures {
		if f.Settled || f.Buyer != "" {
			continue
		}
		if tick > f.MaturityTick-2000 {
			continue // too close to maturity
		}
		// 5% chance per tick per AI faction to buy
		for _, p := range players {
			if p == nil || p.Credits < f.Deposit*3 {
				continue
			}
			if rand.Intn(20) != 0 {
				continue
			}
			f.Buyer = p.Name
			p.Credits -= f.Deposit
			game.LogEvent("trade", p.Name,
				fmt.Sprintf("📊 %s bought %s futures contract: %d units at %.0fcr, matures in ~%d min. Deposit: %dcr",
					p.Name, f.Resource, f.Quantity, f.LockedPrice, (f.MaturityTick-tick)/600, f.Deposit))
			break
		}
	}
}

func (cfs *CommodityFuturesSystem) generateOffering(tick int64, market interface{ GetSellPrice(string) float64 }, game GameProvider) {
	resources := []string{entities.ResIron, entities.ResOil, entities.ResWater,
		entities.ResRareMetals, entities.ResHelium3}
	res := resources[rand.Intn(len(resources))]

	price := market.GetSellPrice(res)
	qty := 100 + rand.Intn(400)
	deposit := int(price * float64(qty) * 0.1) // 10% deposit
	if deposit < 100 {
		deposit = 100
	}
	maturity := tick + 5000 + int64(rand.Intn(8000))

	f := &FuturesContract{
		ID:           len(cfs.futures) + 1,
		Resource:     res,
		Quantity:     qty,
		LockedPrice:  price,
		Deposit:      deposit,
		MaturityTick: maturity,
	}
	cfs.futures = append(cfs.futures, f)

	game.LogEvent("trade", "",
		fmt.Sprintf("📊 FUTURES AVAILABLE: %d %s at %.0fcr/unit, matures in ~%d min. Deposit: %dcr. Will the price go up?",
			qty, res, price, (maturity-tick)/600, deposit))
}
