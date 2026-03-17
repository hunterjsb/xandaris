package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&MarketMakerSystem{
		BaseSystem: NewBaseSystem("MarketMaker", 27),
	})
}

// MarketMakerSystem provides liquidity when the order book has severe
// imbalances. It converts between surplus and scarce resources to
// prevent complete market failure.
//
// When a system has both:
//   - Massive surplus of cheap resources (Iron, RM at 0.2x base)
//   - Buy orders for scarce resources with NO sellers
//
// The market maker converts surplus into scarce resources at an
// unfavorable rate (simulating universal fabricators/recyclers).
//
// Conversion rates (intentionally expensive):
//   100 Iron + 50 Rare Metals → 10 Electronics
//   50 Iron + 20 Water → 15 Fuel
//   30 Iron → 10 Water (desalination/purification)
//
// This prevents the deadlock where everyone has Iron/RM but needs
// Electronics/Fuel/Water and there's no way to convert between them.
type MarketMakerSystem struct {
	*BaseSystem
}

type conversion struct {
	inputs  map[string]int
	output  string
	amount  int
}

var conversions = []conversion{
	{map[string]int{"Iron": 100, "Rare Metals": 50}, "Electronics", 10},
	{map[string]int{"Iron": 50, "Water": 20}, "Fuel", 15},
	{map[string]int{"Iron": 30}, "Water", 10},
	{map[string]int{"Rare Metals": 40, "Iron": 20}, "Helium-3", 5},
}

func (mms *MarketMakerSystem) OnTick(tick int64) {
	if tick%200 != 0 {
		return
	}

	ctx := mms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	ob := game.GetOrderBook()
	market := game.GetMarketEngine()
	if ob == nil || market == nil {
		return
	}

	systems := game.GetSystems()
	players := ctx.GetPlayers()

	playerByName := make(map[string]*entities.Player)
	for _, p := range players {
		if p != nil {
			playerByName[p.Name] = p
		}
	}

	for _, sys := range systems {
		// Check if this system has unfilled buy orders for scarce resources
		for _, conv := range conversions {
			buyOrders := ob.GetOrders(sys.ID, conv.output)
			hasDemand := false
			for _, o := range buyOrders {
				if o.Action == "buy" && o.Active && o.Quantity > 0 {
					hasDemand = true
					break
				}
			}
			if !hasDemand {
				continue
			}

			// Find a planet in this system with enough input resources
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner == "" {
					continue
				}

				// Check if planet has ALL required inputs
				canConvert := true
				for res, needed := range conv.inputs {
					if planet.GetStoredAmount(res) < needed {
						canConvert = false
						break
					}
				}
				if !canConvert {
					continue
				}

				// Check planet has a Trading Post (needed for market participation)
				hasTP := false
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
						hasTP = true
						break
					}
				}
				if !hasTP {
					continue
				}

				// Execute conversion
				for res, needed := range conv.inputs {
					planet.RemoveStoredResource(res, needed)
				}
				planet.AddStoredResource(conv.output, conv.amount)

				// Bump trade volume
				market.AddTradeVolume(conv.output, conv.amount, false)

				fmt.Printf("[MarketMaker] %s on %s: converted inputs → %d %s\n",
					planet.Owner, planet.Name, conv.amount, conv.output)

				game.LogEvent("trade", planet.Owner,
					fmt.Sprintf("%s fabricated %d %s from surplus materials (market maker)",
						planet.Owner, conv.amount, conv.output))

				break // one conversion per system per tick
			}
		}
	}
}
