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

// StandingOrderProvider gives access to standing orders and trade execution.
type StandingOrderProvider interface {
	GetStandingOrderInfos() []StandingOrderInfo
	ExecuteStandingOrderTrade(order StandingOrderInfo, player *entities.Player) error
}

func (sos *StandingOrderSystem) OnTick(tick int64) {
	if tick%30 != 0 {
		return
	}

	ctx := sos.GetContext()
	if ctx == nil {
		return
	}

	gameObj := ctx.GetGame()
	if gameObj == nil {
		return
	}

	sop, ok := gameObj.(StandingOrderProvider)
	if !ok {
		return
	}

	playersIface := ctx.GetPlayers()
	if playersIface == nil {
		return
	}
	players, ok := playersIface.([]*entities.Player)
	if !ok {
		return
	}

	logger, _ := gameObj.(EventLogger)

	playerByName := make(map[string]*entities.Player)
	for _, p := range players {
		if p != nil {
			playerByName[p.Name] = p
		}
	}

	orders := sop.GetStandingOrderInfos()
	for _, order := range orders {
		if !order.Active {
			continue
		}
		player := playerByName[order.Player]
		if player == nil {
			continue
		}

		planet := findPlanetByID(player, order.PlanetID)
		if planet == nil {
			continue
		}

		stock := planet.GetStoredAmount(order.Resource)

		if order.Action == "sell" && stock <= order.Threshold {
			continue
		}
		if order.Action == "buy" && stock >= order.Threshold {
			continue
		}

		if err := sop.ExecuteStandingOrderTrade(order, player); err == nil {
			if logger != nil {
				logger.LogEvent("trade", order.Player,
					fmt.Sprintf("[Auto] %s %s %d %s (order #%d)",
						order.Player, order.Action, order.Quantity, order.Resource, order.ID))
			}
		}
	}
}

func findPlanetByID(player *entities.Player, planetID int) *entities.Planet {
	for _, p := range player.OwnedPlanets {
		if p != nil && p.GetID() == planetID {
			return p
		}
	}
	return nil
}
