//go:build js

package server

import (
	"fmt"

	"github.com/hunterjsb/xandaris/game"
)

func (gs *GameServer) forwardTradeToRemote(cmd game.GameCommand) {
	sendResult(cmd, fmt.Errorf("remote trading not available in browser"))
}
