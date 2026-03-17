package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&OrderMatchingSystem{
		BaseSystem: NewBaseSystem("OrderMatching", 27),
	})
}

// OrderMatchingSystem matches buy and sell limit orders within each system.
// When a buy order's price >= a sell order's price, the trade executes:
// - Resources transfer from seller's planet to buyer's planet
// - Credits transfer from buyer to seller at the matched price
// - Both orders update their filled quantities
type OrderMatchingSystem struct {
	*BaseSystem
}

func (oms *OrderMatchingSystem) OnTick(tick int64) {
	if tick%50 != 0 {
		return
	}

	ctx := oms.GetContext()
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

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	systemsMap := game.GetSystemsMap()

	playerByName := make(map[string]*entities.Player)
	for _, p := range players {
		if p != nil {
			playerByName[p.Name] = p
		}
	}

	// Process each system
	for _, sys := range systems {
		matches := ob.FindMatches(sys.ID)
		for _, m := range matches {
			buyer := playerByName[m.BuyOrder.Player]
			seller := playerByName[m.SellOrder.Player]
			if buyer == nil || seller == nil {
				continue
			}

			total := m.Price * m.Quantity
			if buyer.Credits < total {
				continue
			}

			// Find planets
			buyerPlanet := findPlanetByID(systemsMap, m.BuyOrder.PlanetID)
			sellerPlanet := findPlanetByID(systemsMap, m.SellOrder.PlanetID)
			if sellerPlanet == nil || buyerPlanet == nil {
				continue
			}

			// Check seller has stock
			if sellerPlanet.GetStoredAmount(m.BuyOrder.Resource) < m.Quantity {
				continue
			}

			// Execute
			sellerPlanet.RemoveStoredResource(m.BuyOrder.Resource, m.Quantity)
			buyerPlanet.AddStoredResource(m.BuyOrder.Resource, m.Quantity)
			buyer.Credits -= total
			seller.Credits += total

			// Log
			game.LogEvent("trade", seller.Name,
				fmt.Sprintf("%s sold %d %s @ %dcr to %s (limit order)",
					seller.Name, m.Quantity, m.BuyOrder.Resource, m.Price, buyer.Name))

			if me := game.GetMarketEngine(); me != nil {
				me.AddTradeVolume(m.BuyOrder.Resource, m.Quantity, true)
			}
		}
	}
}
