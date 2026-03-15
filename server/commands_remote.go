//go:build !js

package server

import (
	"encoding/json"
	"fmt"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/game"
)

// forwardTradeToRemote sends a trade command to the remote server.
func (gs *GameServer) forwardTradeToRemote(cmd game.GameCommand) {
	remote, ok := gs.remoteSync.(*RemoteSync)
	if !ok || remote == nil {
		sendResult(cmd, fmt.Errorf("remote sync not available"))
		return
	}

	td, ok := cmd.Data.(game.TradeCommandData)
	if !ok {
		sendResult(cmd, fmt.Errorf("invalid trade data"))
		return
	}

	data, err := remote.ForwardTrade(td.Resource, td.Quantity, td.Buy)
	if err != nil {
		sendResult(cmd, err)
		return
	}

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
		var trData struct {
			Resource string `json:"resource"`
			Quantity int    `json:"quantity"`
			Action   string `json:"action"`
			Total    int    `json:"total"`
		}
		json.Unmarshal(resp.Data, &trData)
		cmd.Result <- economy.TradeRecord{
			Resource: trData.Resource,
			Quantity: trData.Quantity,
			Action:   trData.Action,
			Total:    trData.Total,
		}
		close(cmd.Result)
	}
}
