package server

import (
	"fmt"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
)

func (gs *GameServer) handleTradeCommand(cmd game.GameCommand) {
	td, ok := cmd.Data.(game.TradeCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid trade data"))
		return
	}
	exec := gs.State.TradeExec
	human := gs.resolvePlayer(cmd)
	if exec == nil || human == nil {
		sendResult(cmd, fmt.Errorf("game not initialized"))
		return
	}

	// Resolve optional planet
	var tradePlanet *entities.Planet
	if td.PlanetID > 0 && gs.CargoCommander != nil {
		tradePlanet = gs.CargoCommander.FindPlanetByID(td.PlanetID)
	}

	var result interface{}
	var err error
	if td.Buy {
		result, err = exec.Buy(human, gs.State.Players, td.Resource, td.Quantity, tradePlanet)
	} else {
		result, err = exec.Sell(human, gs.State.Players, td.Resource, td.Quantity, tradePlanet)
	}

	if cmd.Result != nil {
		if err != nil {
			cmd.Result <- err
		} else {
			cmd.Result <- result
			if record, ok := result.(economy.TradeRecord); ok && gs.Events != nil {
				action := "bought"
				if record.Action == "sell" {
					action = "sold"
				}
				_, gt, _, _ := gs.GetTickInfo()
				gs.Events.Addf(gs.TickManager.GetCurrentTick(), gt, game.EventTrade, record.Player,
					"%s %s %d %s @ %.0f = %dcr", record.Player, action, record.Quantity, record.Resource, record.UnitPrice, record.Total)
			}
		}
		close(cmd.Result)
	}
}

func (gs *GameServer) handleCargoCommand(cmd game.GameCommand) {
	cd, ok := cmd.Data.(game.CargoCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid cargo data"))
		return
	}
	if gs.CargoCommander == nil {
		sendResult(cmd, fmt.Errorf("cargo system not initialized"))
		return
	}

	ship := game.FindShipByID(gs.State.Players, cd.ShipID)
	if ship == nil {
		sendResult(cmd, fmt.Errorf("ship not found"))
		return
	}
	planet := gs.CargoCommander.FindPlanetByID(cd.PlanetID)
	if planet == nil {
		sendResult(cmd, fmt.Errorf("planet not found"))
		return
	}

	var qty int
	var err error
	if cd.Load {
		qty, err = gs.CargoCommander.LoadCargo(ship, planet, cd.Resource, cd.Quantity)
	} else {
		qty, err = gs.CargoCommander.UnloadCargo(ship, planet, cd.Resource, cd.Quantity)
	}

	if err != nil {
		sendResult(cmd, err)
	} else {
		sendSuccess(cmd, qty)
	}
}

func (gs *GameServer) handleStandingOrderCommand(cmd game.GameCommand) {
	data, ok := cmd.Data.(game.StandingOrderCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid standing order data"))
		return
	}

	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}

	// Verify the authenticated player owns this planet
	ownsIt := false
	for _, planet := range human.OwnedPlanets {
		if planet != nil && planet.GetID() == data.PlanetID {
			ownsIt = true
			break
		}
	}
	if !ownsIt {
		sendResult(cmd, fmt.Errorf("not your planet"))
		return
	}

	order := &game.StandingOrder{
		Player:    human.Name,
		PlanetID:  data.PlanetID,
		Resource:  data.Resource,
		Action:    data.Action,
		Quantity:  data.Quantity,
		Threshold: data.Threshold,
		MaxPrice:  data.MaxPrice,
		MinPrice:  data.MinPrice,
	}

	id := gs.State.AddStandingOrder(order)
	fmt.Printf("[Server] Standing order #%d: %s %s %d %s on planet %d (threshold %d)\n",
		id, order.Player, order.Action, order.Quantity, order.Resource, order.PlanetID, order.Threshold)

	sendSuccess(cmd, map[string]interface{}{
		"order_id": id,
		"action":   order.Action,
		"resource": order.Resource,
		"quantity": order.Quantity,
	})
}

func (gs *GameServer) handleCancelOrderCommand(cmd game.GameCommand) {
	data, ok := cmd.Data.(game.CancelOrderCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid cancel order data"))
		return
	}

	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}

	// Verify the order belongs to the authenticated player
	for _, order := range gs.State.StandingOrders {
		if order.ID == data.OrderID && order.Player != human.Name {
			sendResult(cmd, fmt.Errorf("not your order"))
			return
		}
	}

	if gs.State.RemoveStandingOrder(data.OrderID) {
		sendSuccess(cmd, map[string]interface{}{"cancelled": data.OrderID})
	} else {
		sendResult(cmd, fmt.Errorf("order #%d not found", data.OrderID))
	}
}
