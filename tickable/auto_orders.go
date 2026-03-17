package tickable

import (
	"fmt"

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

			for _, res := range resources {
				stored := planet.GetStoredAmount(res)
				buyPrice := int(market.GetBuyPrice(res))
				sellPrice := int(market.GetSellPrice(res))
				if buyPrice <= 0 { buyPrice = 10 }
				if sellPrice <= 0 { sellPrice = 5 }

				// Dynamic sell threshold: high-value resources sell sooner
				// Oil/Electronics at 5x+ base price → sell above 100 (not 500)
				// Cheap resources → only sell real surplus above 300
				sellThreshold := 300
				if sellPrice > 200 {
					sellThreshold = 100 // valuable resource, sell sooner
				} else if sellPrice > 50 {
					sellThreshold = 200
				}
				keepBuffer := sellThreshold / 2

				if stored < 50 && player.Credits > 500 {
					// Need this resource — place buy order
					maxSpend := player.Credits / 10
					qty := 50
					if buyPrice*qty > maxSpend {
						qty = maxSpend / buyPrice
					}
					if qty > 0 {
						ob.PlaceOrder(sys.ID, planet.GetID(), planet.Owner, res, "buy", qty, buyPrice)
					}
				} else if stored > sellThreshold {
					// Surplus — sell above buffer
					surplus := stored - keepBuffer
					if surplus > 100 {
						surplus = 100
					}
					if surplus > 0 {
						ob.PlaceOrder(sys.ID, planet.GetID(), planet.Owner, res, "sell", surplus, sellPrice)
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
