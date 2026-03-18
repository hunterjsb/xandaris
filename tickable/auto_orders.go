package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AutoOrderSystem{
		BaseSystem: NewBaseSystem("AutoOrders", 29),
	})
}

// AutoOrderSystem automatically places limit orders for planets based on
// their resource needs. This bridges the gap between the order book system
// and agents that don't use it directly.
//
// For each planet with a Trading Post:
// - Resources with <50 stored: place a buy order (drives diversity bonus)
// - Resources with >500 stored: place a sell order (monetize surplus)
//
// Orders are refreshed every 500 ticks (~50 seconds). Old orders are
// replaced, not stacked.
type AutoOrderSystem struct {
	*BaseSystem
	lastRun int64
}

func (aos *AutoOrderSystem) OnTick(tick int64) {
	if tick-aos.lastRun < 500 {
		return
	}
	aos.lastRun = tick

	ctx := aos.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	ob := game.GetOrderBook()
	if ob == nil {
		return
	}

	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	playerByName := make(map[string]*entities.Player)
	for _, p := range players {
		if p != nil {
			playerByName[p.Name] = p
		}
	}

	resources := []string{
		entities.ResWater, entities.ResIron, entities.ResOil,
		entities.ResFuel, entities.ResRareMetals, entities.ResHelium3,
		entities.ResElectronics,
	}

	// Prune dead orders first
	ob.PruneExpired()

	// Track which players we've cleared per system (so we clear once, place fresh)
	cleared := make(map[string]bool)

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			// Clear old auto-orders for this player in this system (once)
			clearKey := fmt.Sprintf("%s:%d", planet.Owner, sys.ID)
			if !cleared[clearKey] {
				ob.ClearPlayerOrders(planet.Owner, sys.ID)
				cleared[clearKey] = true
			}

			// Must have a Trading Post
			hasTP := false
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					if b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
						hasTP = true
						break
					}
				}
			}
			if !hasTP {
				continue
			}

			player := playerByName[planet.Owner]
			if player == nil {
				continue
			}

			// Count resource types stocked for diversity awareness
			typesStocked := 0
			for _, res := range resources {
				if planet.GetStoredAmount(res) > 0 {
					typesStocked++
				}
			}

			for _, res := range resources {
				stored := planet.GetStoredAmount(res)
				buyPrice := int(market.GetBuyPrice(res))
				sellPrice := int(market.GetSellPrice(res))
				basePrice := int(economy.GetBasePrice(res))
				if buyPrice <= 0 { buyPrice = 10 }
				if sellPrice <= 0 { sellPrice = 5 }
				if basePrice <= 0 { basePrice = 100 }

				// Sell threshold considers diversity impact
				// Don't sell a resource if it would drop diversity below current level
				// AND the resource isn't very profitable to sell
				priceRatio := float64(sellPrice) / float64(basePrice)
				sellThreshold := 300
				if priceRatio > 2.0 {
					sellThreshold = 50 // very profitable, sell aggressively
				} else if priceRatio > 1.0 {
					sellThreshold = 150
				} else if typesStocked <= 5 {
					sellThreshold = 500 // cheap resource + low diversity = hoard for bonus
				}
				keepBuffer := sellThreshold / 2

				if stored < 50 && player.Credits > 500 {
					// Need this resource — buy for diversity
					// Skip luxury resources at floor prices (oversupplied)
					isEssential := res == entities.ResWater || res == entities.ResFuel || res == entities.ResIron
					if !isEssential && priceRatio < 0.3 {
						continue
					}
					maxSpend := player.Credits / 10
					qty := 50
					if buyPrice*qty > maxSpend {
						qty = maxSpend / buyPrice
					}
					if qty > 0 {
						ob.PlaceOrder(sys.ID, planet.GetID(), planet.Owner, res, "buy", qty, buyPrice)
					}
				} else if stored > sellThreshold && priceRatio > 0.3 {
					// Surplus — sell above buffer (but not at floor prices)
					surplus := stored - keepBuffer
					if surplus > 100 {
						surplus = 100
					}
					if surplus > 0 {
						ob.PlaceOrder(sys.ID, planet.GetID(), planet.Owner, res, "sell", surplus, sellPrice)
					}
				}

				// Emergency overflow: if storage > 90% capacity, dump at half price
				// Prevents resources from being wasted when storage is full
				cap := planet.GetStorageCapacity()
				if cap > 0 && stored > cap*9/10 {
					dumpQty := stored - cap*3/4 // dump down to 75%
					if dumpQty > 200 {
						dumpQty = 200
					}
					if dumpQty > 0 {
						dumpPrice := sellPrice / 2
						if dumpPrice < 1 { dumpPrice = 1 }
						ob.PlaceOrder(sys.ID, planet.GetID(), planet.Owner, res, "sell", dumpQty, dumpPrice)
					}
				}
			}
		}
	}

	// Count orders placed
	allOrders := ob.GetAllOrders()
	active := 0
	for _, o := range allOrders {
		if o.Active && o.Quantity > 0 {
			active++
		}
	}
	if active > 0 && tick%5000 == 0 {
		fmt.Printf("[AutoOrders] %d active limit orders on the book\n", active)
	}
}
