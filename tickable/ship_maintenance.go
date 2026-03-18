package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ShipMaintenanceSystem{
		BaseSystem: NewBaseSystem("ShipMaintenance", 30),
	})
}

// ShipMaintenanceSystem deducts credit upkeep for each ship a player owns.
// This creates natural pressure against fleet bloat — maintaining a 300-ship
// armada should cost serious money.
//
// Costs per ship per interval:
//
//	Scout: 2cr, Cargo: 5cr, Colony: 8cr
//	Frigate: 10cr, Destroyer: 20cr, Cruiser: 40cr
//
// If a player can't afford maintenance, the most expensive ships
// are mothballed (set to Idle with a warning).
type ShipMaintenanceSystem struct {
	*BaseSystem
	lastWarning map[string]int64 // player → last warning tick
}

var shipMaintenanceCost = map[entities.ShipType]int{
	entities.ShipTypeScout:     2,
	entities.ShipTypeCargo:     5,
	entities.ShipTypeColony:    8,
	entities.ShipTypeFrigate:   10,
	entities.ShipTypeDestroyer: 20,
	entities.ShipTypeCruiser:   40,
}

func (sms *ShipMaintenanceSystem) OnTick(tick int64) {
	if tick%300 != 0 {
		return
	}

	ctx := sms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if sms.lastWarning == nil {
		sms.lastWarning = make(map[string]int64)
	}

	for _, player := range ctx.GetPlayers() {
		if player == nil || len(player.OwnedShips) == 0 {
			continue
		}

		totalCost := 0
		for _, ship := range player.OwnedShips {
			if ship == nil {
				continue
			}
			cost := shipMaintenanceCost[ship.ShipType]
			if cost == 0 {
				cost = 5
			}
			totalCost += cost
		}

		if totalCost <= 0 {
			continue
		}

		if player.Credits >= totalCost {
			player.Credits -= totalCost
		} else {
			// Can't afford full maintenance — deduct what we can and warn
			player.Credits = 0
			if tick-sms.lastWarning[player.Name] > 10000 {
				sms.lastWarning[player.Name] = tick
				game.LogEvent("alert", player.Name,
					fmt.Sprintf("⚠️ %s can't afford fleet maintenance (%d cr/interval for %d ships)! Consider scrapping excess ships.",
						player.Name, totalCost, len(player.OwnedShips)))
			}
		}
	}
}
