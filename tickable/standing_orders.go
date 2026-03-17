package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&StandingOrderSystem{
		BaseSystem: NewBaseSystem("StandingOrders", 25),
	})
}

// StandingOrderSystem executes active standing trade orders each cycle.
type StandingOrderSystem struct {
	*BaseSystem
}

// StandingOrderInfo is a minimal representation of a standing order (avoids importing game).
type StandingOrderInfo struct {
	ID        int
	Player    string
	PlanetID  int
	Resource  string
	Action    string // "buy" or "sell"
	Quantity  int
	Threshold int
	Active    bool
}

func (sos *StandingOrderSystem) OnTick(tick int64) {
	if tick%30 != 0 {
		return
	}

	ctx := sos.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()

	playerByName := make(map[string]*entities.Player)
	for _, p := range players {
		if p != nil {
			playerByName[p.Name] = p
		}
	}

	orders := game.GetStandingOrderInfos()
	for _, order := range orders {
		if !order.Active {
			continue
		}
		player := playerByName[order.Player]
		if player == nil {
			continue
		}

		planet := findPlanetByIDFromSystems(game.GetSystems(), order.PlanetID)
		if planet == nil || planet.Owner != order.Player {
			continue
		}

		stock := planet.GetStoredAmount(order.Resource)

		if order.Action == "sell" && stock <= order.Threshold {
			continue
		}
		if order.Action == "buy" && stock >= order.Threshold {
			continue
		}

		// Credit floor: don't auto-buy if credits are critically low
		if order.Action == "buy" && player.Credits < 1000 {
			continue
		}

		if err := game.ExecuteStandingOrderTrade(order, player); err == nil {
			game.LogEvent("trade", order.Player,
				fmt.Sprintf("[Auto] %s %s %d %s (order #%d)",
					order.Player, order.Action, order.Quantity, order.Resource, order.ID))
		}
	}
}

// findPlanetByIDFromSystems looks up a planet from authoritative system entities.
func findPlanetByIDFromSystems(systems []*entities.System, planetID int) *entities.Planet {
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if p, ok := e.(*entities.Planet); ok && p.GetID() == planetID {
				return p
			}
		}
	}
	return nil
}
