package server

import (
	"fmt"

	"github.com/hunterjsb/xandaris/game"
)

func (gs *GameServer) handleFleetMoveCommand(cmd game.GameCommand) {
	fd, ok := cmd.Data.(game.FleetMoveCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid fleet move data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	fleet, owner := game.FindFleetByID(gs.State.Players, fd.FleetID)
	if fleet == nil || owner != human {
		sendResult(cmd, fmt.Errorf("fleet not found or not owned"))
		return
	}
	success, fail := gs.FleetCmdExecutor.MoveFleetToSystem(fleet, fd.TargetSystemID)
	if success == 0 {
		sendResult(cmd, fmt.Errorf("no ships could move (no route or insufficient fuel)"))
		return
	}
	sendSuccess(cmd, map[string]interface{}{
		"fleet_id": fd.FleetID,
		"target":   fd.TargetSystemID,
		"moved":    success,
		"failed":   fail,
	})
}

func (gs *GameServer) handleFleetCreateCommand(cmd game.GameCommand) {
	fd, ok := cmd.Data.(game.FleetCreateCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid fleet create data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	ship := game.FindShipByID(gs.State.Players, fd.ShipID)
	if ship == nil || ship.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("ship not found or not owned"))
		return
	}
	fleet, err := gs.FleetMgmtSystem.CreateFleetFromShip(ship, human)
	if err != nil {
		sendResult(cmd, err)
		return
	}
	sendSuccess(cmd, map[string]interface{}{
		"fleet_id": fleet.ID,
		"ship_id":  fd.ShipID,
		"size":     fleet.Size(),
	})
}

func (gs *GameServer) handleFleetDisbandCommand(cmd game.GameCommand) {
	fd, ok := cmd.Data.(game.FleetDisbandCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid fleet disband data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	fleet, owner := game.FindFleetByID(gs.State.Players, fd.FleetID)
	if fleet == nil || owner != human {
		sendResult(cmd, fmt.Errorf("fleet not found or not owned"))
		return
	}
	shipCount := len(fleet.Ships)
	err := gs.FleetMgmtSystem.DisbandFleet(fleet, human)
	if err != nil {
		sendResult(cmd, err)
		return
	}
	sendSuccess(cmd, map[string]interface{}{
		"fleet_id":       fd.FleetID,
		"ships_released": shipCount,
	})
}

func (gs *GameServer) handleFleetAddShipCommand(cmd game.GameCommand) {
	fd, ok := cmd.Data.(game.FleetAddShipCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid fleet add ship data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	ship := game.FindShipByID(gs.State.Players, fd.ShipID)
	if ship == nil || ship.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("ship not found or not owned"))
		return
	}
	fleet, owner := game.FindFleetByID(gs.State.Players, fd.FleetID)
	if fleet == nil || owner != human {
		sendResult(cmd, fmt.Errorf("fleet not found or not owned"))
		return
	}
	err := gs.FleetMgmtSystem.AddShipToFleet(ship, fleet, human)
	if err != nil {
		sendResult(cmd, err)
		return
	}
	sendSuccess(cmd, map[string]interface{}{
		"fleet_id": fd.FleetID,
		"ship_id":  fd.ShipID,
		"size":     fleet.Size(),
	})
}

func (gs *GameServer) handleFleetRemoveShipCommand(cmd game.GameCommand) {
	fd, ok := cmd.Data.(game.FleetRemoveShipCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid fleet remove ship data"))
		return
	}
	human := gs.resolvePlayer(cmd)
	if human == nil {
		sendResult(cmd, fmt.Errorf("no player"))
		return
	}
	ship := game.FindShipByID(gs.State.Players, fd.ShipID)
	if ship == nil || ship.Owner != human.Name {
		sendResult(cmd, fmt.Errorf("ship not found or not owned"))
		return
	}
	fleet, owner := game.FindFleetByID(gs.State.Players, fd.FleetID)
	if fleet == nil || owner != human {
		sendResult(cmd, fmt.Errorf("fleet not found or not owned"))
		return
	}
	err := gs.FleetMgmtSystem.RemoveShipFromFleet(ship, fleet, human)
	if err != nil {
		sendResult(cmd, err)
		return
	}
	sendSuccess(cmd, map[string]interface{}{
		"fleet_id": fd.FleetID,
		"ship_id":  fd.ShipID,
	})
}
