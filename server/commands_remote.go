package server

import (
	"encoding/json"
	"fmt"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/game"
)

// commandEndpoints maps command types to their remote API endpoints and body builders.
var commandEndpoints = map[game.CommandType]string{
	game.CmdTrade:              "/api/market/trade",
	game.CmdBuild:              "/api/build",
	game.CmdBuildShip:          "/api/ships/build",
	game.CmdMoveShip:           "/api/ships/move",
	game.CmdUpgrade:            "/api/upgrade",
	game.CmdRefuel:             "/api/ships/refuel",
	game.CmdCargoLoad:          "/api/cargo/load",
	game.CmdCargoUnload:        "/api/cargo/unload",
	game.CmdColonize:           "/api/colonize",
	game.CmdFleetMove:          "/api/fleets/move",
	game.CmdFleetCreate:        "/api/fleets/create",
	game.CmdFleetDisband:       "/api/fleets/disband",
	game.CmdFleetAddShip:       "/api/fleets/add-ship",
	game.CmdFleetRemoveShip:    "/api/fleets/remove-ship",
	game.CmdWorkforceAssign:    "/api/workforce/assign",
	game.CmdCancelConstruction: "/api/construction/cancel",
	game.CmdDemolish:           "/api/demolish",
	game.CmdTransferFuel:       "/api/ships/transfer-fuel",
}

// convertCommandToAPI converts a game command's data to API-compatible JSON.
func convertCommandToAPI(cmd game.GameCommand) ([]byte, error) {
	switch d := cmd.Data.(type) {
	case game.TradeCommandData:
		action := "sell"
		if d.Buy {
			action = "buy"
		}
		return json.Marshal(map[string]interface{}{
			"resource":  d.Resource,
			"quantity":  d.Quantity,
			"action":    action,
			"planet_id": d.PlanetID,
		})
	case game.BuildCommandData:
		return json.Marshal(map[string]interface{}{
			"planet_id":     d.PlanetID,
			"building_type": d.BuildingType,
			"resource_id":   d.ResourceID,
		})
	case game.ShipBuildCommandData:
		return json.Marshal(map[string]interface{}{
			"planet_id": d.PlanetID,
			"ship_type": d.ShipType,
		})
	case game.ShipMoveCommandData:
		return json.Marshal(map[string]interface{}{
			"ship_id":          d.ShipID,
			"target_system_id": d.TargetSystemID,
		})
	case game.UpgradeCommandData:
		return json.Marshal(map[string]interface{}{
			"planet_id":      d.PlanetID,
			"building_index": d.BuildingIndex,
		})
	case game.CargoCommandData:
		return json.Marshal(map[string]interface{}{
			"ship_id":   d.ShipID,
			"planet_id": d.PlanetID,
			"resource":  d.Resource,
			"quantity":  d.Quantity,
		})
	case game.FleetMoveCommandData:
		return json.Marshal(map[string]interface{}{
			"fleet_id":         d.FleetID,
			"target_system_id": d.TargetSystemID,
		})
	case game.FleetCreateCommandData:
		return json.Marshal(map[string]interface{}{"ship_id": d.ShipID})
	case game.FleetDisbandCommandData:
		return json.Marshal(map[string]interface{}{"fleet_id": d.FleetID})
	case game.FleetAddShipCommandData:
		return json.Marshal(map[string]interface{}{"ship_id": d.ShipID, "fleet_id": d.FleetID})
	case game.FleetRemoveShipCommandData:
		return json.Marshal(map[string]interface{}{"ship_id": d.ShipID, "fleet_id": d.FleetID})
	case game.WorkforceAssignCommandData:
		return json.Marshal(map[string]interface{}{
			"planet_id":      d.PlanetID,
			"building_index": d.BuildingIndex,
			"workers":        d.Workers,
		})
	case game.CancelConstructionCommandData:
		return json.Marshal(map[string]interface{}{"construction_id": d.ConstructionID})
	case game.DemolishCommandData:
		return json.Marshal(map[string]interface{}{
			"planet_id":      d.PlanetID,
			"building_index": d.BuildingIndex,
		})
	case game.TransferFuelCommandData:
		return json.Marshal(map[string]interface{}{
			"from_ship_id": d.FromShipID,
			"to_ship_id":   d.ToShipID,
			"amount":       d.Amount,
		})
	case game.ShipRefuelCommandData:
		return json.Marshal(map[string]interface{}{
			"ship_id":   d.ShipID,
			"planet_id": d.PlanetID,
			"amount":    d.Amount,
		})
	case game.ColonizeCommandData:
		return json.Marshal(map[string]interface{}{
			"ship_id":   d.ShipID,
			"planet_id": d.PlanetID,
		})
	default:
		// Try generic marshal as fallback
		return json.Marshal(cmd.Data)
	}
}

// forwardCommandToRemote sends any game command to the remote server.
func (gs *GameServer) forwardCommandToRemote(cmd game.GameCommand) {
	remote, ok := gs.remoteSync.(*RemoteSync)
	if !ok || remote == nil {
		sendResult(cmd, fmt.Errorf("remote sync not available"))
		return
	}

	endpoint, exists := commandEndpoints[cmd.Type]
	if !exists {
		sendResult(cmd, fmt.Errorf("unknown remote command: %s", cmd.Type))
		return
	}

	// Convert command data to API-compatible JSON
	body, err := convertCommandToAPI(cmd)
	if err != nil {
		sendResult(cmd, err)
		return
	}

	// Send to remote
	data, err := remote.apiPost(endpoint, string(body))
	if err != nil {
		sendResult(cmd, fmt.Errorf("remote request failed: %w", err))
		return
	}

	// Parse response
	var resp struct {
		OK    bool            `json:"ok"`
		Error string          `json:"error"`
		Data  json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		sendResult(cmd, fmt.Errorf("failed to parse response"))
		return
	}
	if !resp.OK {
		sendResult(cmd, fmt.Errorf("%s", resp.Error))
		return
	}

	if cmd.Result != nil {
		// For trades, parse as TradeRecord for the UI
		if cmd.Type == "trade" {
			var tr struct {
				Resource string `json:"resource"`
				Quantity int    `json:"quantity"`
				Action   string `json:"action"`
				Total    int    `json:"total"`
			}
			json.Unmarshal(resp.Data, &tr)
			cmd.Result <- economy.TradeRecord{
				Resource: tr.Resource,
				Quantity: tr.Quantity,
				Action:   tr.Action,
				Total:    tr.Total,
			}
		} else {
			// For other commands, return the raw response data
			var result interface{}
			json.Unmarshal(resp.Data, &result)
			cmd.Result <- result
		}
		close(cmd.Result)
	}
}
