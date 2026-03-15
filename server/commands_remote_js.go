//go:build js

package server

import (
	"fmt"

	"github.com/hunterjsb/xandaris/game"
)

func (gs *GameServer) forwardTradeToRemote(cmd game.GameCommand) {
	sendResult(cmd, fmt.Errorf("remote not available in browser"))
}

func (gs *GameServer) forwardCommandToRemote(cmd game.GameCommand) {
	sendResult(cmd, fmt.Errorf("remote not available in browser"))
}
