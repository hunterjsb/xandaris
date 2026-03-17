package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&LocalExchangeSystem{
		BaseSystem: NewBaseSystem("LocalExchange", 26),
	})
}

// LocalExchangeSystem automatically facilitates trade between planets in the
// same system. If planet A has surplus Water and planet B needs Water, the
// Trading Posts handle the exchange automatically (with a small fee).
//
// This simulates the natural commerce that happens when civilizations share
// a star system — you don't need to manually sell/buy every resource.
//
// Rules:
// - Both planets must have operational Trading Posts
// - Only trades surplus (above 200 units) to planets with shortage (below 50)
// - Seller gets credits at local sell price
// - Buyer's planet gets the resources
// - TP processing fee applies
type LocalExchangeSystem struct {
	*BaseSystem
}

func (les *LocalExchangeSystem) OnTick(tick int64) {
	// Run every 100 ticks (~10 seconds)
	if tick%100 != 0 {
		return
	}

	ctx := les.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// For each system, find planets with TPs and match surplus to shortage
	for _, sys := range systems {
		les.processSystem(sys, players, market, game)
	}
}

type planetInfo struct {
	planet *entities.Planet
	player *entities.Player
	tpLevel int
}

func (les *LocalExchangeSystem) processSystem(sys *entities.System, players []*entities.Player, market *economy.Market, game GameProvider) {
	// Find all owned planets with Trading Posts in this system
	var planets []planetInfo
	for _, e := range sys.Entities {
		planet, ok := e.(*entities.Planet)
		if !ok || planet.Owner == "" {
			continue
		}

		// Find the player
		var owner *entities.Player
		for _, p := range players {
			if p != nil && p.Name == planet.Owner {
				owner = p
				break
			}
		}
		if owner == nil {
			continue
		}

		// Check for operational Trading Post
		tpLevel := 0
		for _, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok {
				if b.BuildingType == entities.BuildingTradingPost && b.IsOperational && b.GetStaffingRatio() > 0 {
					tpLevel = b.Level
					break
				}
			}
		}
		if tpLevel == 0 {
			continue
		}

		planets = append(planets, planetInfo{planet: planet, player: owner, tpLevel: tpLevel})
	}

	if len(planets) < 2 {
		return // need at least 2 planets with TPs for local exchange
	}

	// For each resource, match surplus sellers to shortage buyers
	resources := []string{
		entities.ResWater, entities.ResIron, entities.ResOil,
		entities.ResFuel, entities.ResRareMetals, entities.ResHelium3,
		entities.ResElectronics,
	}

	for _, res := range resources {
		for i, seller := range planets {
			sellerStock := seller.planet.GetStoredAmount(res)
			// Surplus: keep 100 as buffer, trade anything above that
			if sellerStock <= 100 {
				continue
			}
			surplus := sellerStock - 100

			for j, buyer := range planets {
				if i == j {
					continue
				}
				if seller.player == buyer.player {
					continue
				}
				buyerStock := buyer.planet.GetStoredAmount(res)
				// Buyer wants it if they have less than 200 (was 50 — too restrictive)
				if buyerStock >= 200 {
					continue
				}

				// Trade batch: up to 50 units per tick (was 20), or whatever surplus allows
				qty := 50
				if qty > surplus {
					qty = surplus
				}
				// Don't oversupply the buyer
				maxNeed := 200 - buyerStock
				if qty > maxNeed {
					qty = maxNeed
				}

				// Price: local sell price
				price := market.GetSellPrice(res)
				total := int(price * float64(qty))
				if total <= 0 {
					continue
				}

				// TP fee adjusted by diplomatic relations
				feeRate := economy.TradingPostFee(seller.tpLevel)
				// Check diplomacy — allies get reduced fees
				if dm := game.GetDiplomacyManager(); dm != nil {
					relation := dm.GetRelation(seller.player.Name, buyer.player.Name)
					feeRate *= economy.DockingFeeMultiplier(relation)
				}
				fee := int(float64(total) * feeRate)
				sellerGets := total - fee

				// Check buyer can afford
				if buyer.player.Credits < total {
					continue
				}

				// Execute exchange
				seller.planet.RemoveStoredResource(res, qty)
				buyer.planet.AddStoredResource(res, qty)
				seller.player.Credits += sellerGets
				buyer.player.Credits -= total

				// Bump market trade volume
				market.AddTradeVolume(res, qty, false)

				surplus -= qty
				if surplus <= 0 {
					break
				}

				game.LogEvent("trade", seller.player.Name,
					fmt.Sprintf("%s sold %d %s @ %.0fcr to %s (auto-exchange)",
						seller.player.Name, qty, res, price, buyer.player.Name))
			}
		}
	}
}
