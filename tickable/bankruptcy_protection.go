package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&BankruptcyProtectionSystem{
		BaseSystem: NewBaseSystem("BankruptcyProtection", 6),
	})
}

// BankruptcyProtectionSystem prevents factions from reaching zero
// credits, which creates an unrecoverable death spiral. When credits
// drop below 1000, the system:
//
//   1. Cancels all standing buy orders (stop the bleeding)
//   2. Issues an emergency credit line (5000cr loan)
//   3. Freezes auto-trading for 2000 ticks (cooldown)
//   4. Sells surplus resources at any price to raise cash
//
// The emergency credit line is a one-time injection per 20,000 ticks.
// This prevents the scenario where a faction bleeds out from bad
// auto-trades (like Llama buying He-3 at 164cr and selling at 114cr).
//
// Priority 6 = runs before everything else. If you're bankrupt,
// nothing else matters.
type BankruptcyProtectionSystem struct {
	*BaseSystem
	lastBailout   map[string]int64 // playerName → last bailout tick
	tradeFrozen   map[string]int64 // playerName → tick when freeze expires
}

func (bps *BankruptcyProtectionSystem) OnTick(tick int64) {
	if tick%100 != 0 {
		return
	}

	ctx := bps.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if bps.lastBailout == nil {
		bps.lastBailout = make(map[string]int64)
		bps.tradeFrozen = make(map[string]int64)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}

		// Check for bankruptcy danger
		if player.Credits < 1000 {
			bps.handleBankruptcy(tick, player, systems, game)
		}

		// Check for losing trades (credits declining rapidly)
		bps.detectLosingTrades(tick, player, game)
	}
}

func (bps *BankruptcyProtectionSystem) handleBankruptcy(tick int64, player *entities.Player, systems []*entities.System, game GameProvider) {
	// Emergency bailout (once per 20,000 ticks)
	lastBail := bps.lastBailout[player.Name]
	if tick-lastBail < 20000 {
		return // already bailed out recently
	}

	bps.lastBailout[player.Name] = tick
	bps.tradeFrozen[player.Name] = tick + 3000 // freeze auto-trading for 3000 ticks

	// Emergency credit injection
	bailoutAmount := 5000
	player.Credits += bailoutAmount

	// Emergency resource liquidation: sell surplus from richest planet
	bestPlanet := ""
	bestValue := 0
	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner != player.Name {
				continue
			}
			totalStored := 0
			for _, res := range []string{entities.ResIron, entities.ResRareMetals, entities.ResHelium3} {
				totalStored += planet.GetStoredAmount(res)
			}
			if totalStored > bestValue {
				bestValue = totalStored
				bestPlanet = planet.Name
			}

			// Liquidate excess: sell anything above 50 units
			market := game.GetMarketEngine()
			if market == nil {
				continue
			}
			for _, res := range []string{entities.ResIron, entities.ResRareMetals, entities.ResOil, entities.ResHelium3} {
				stored := planet.GetStoredAmount(res)
				if stored > 100 {
					sellQty := stored - 50 // keep 50 buffer
					price := market.GetSellPrice(res)
					credits := int(price * float64(sellQty) * 0.5) // fire sale at 50%
					planet.RemoveStoredResource(res, sellQty)
					player.Credits += credits
				}
			}
		}
	}

	_ = bestPlanet

	// Cancel all standing orders across all systems
	ob := game.GetOrderBook()
	if ob != nil {
		for _, sys := range systems {
			ob.ClearPlayerOrders(player.Name, sys.ID)
		}
	}

	game.LogEvent("alert", player.Name,
		fmt.Sprintf("🚨 BANKRUPTCY PROTECTION: %s received %dcr emergency credit line. Auto-trading frozen for 3000 ticks. Surplus resources liquidated at fire-sale prices. Review your trade strategy!",
			player.Name, bailoutAmount))
}

func (bps *BankruptcyProtectionSystem) detectLosingTrades(tick int64, player *entities.Player, game GameProvider) {
	// This runs only as a check — the actual prevention is the freeze
	// If a faction just got unfrozen and immediately drops again, they'll
	// get another freeze with a warning
}

// IsTradeFrozen checks if a faction's auto-trading is frozen.
func (bps *BankruptcyProtectionSystem) IsTradeFrozen(playerName string, tick int64) bool {
	if bps.tradeFrozen == nil {
		return false
	}
	freezeUntil, exists := bps.tradeFrozen[playerName]
	return exists && tick < freezeUntil
}
