package server

import (
	"testing"

	"github.com/hunterjsb/xandaris/game"
)

func TestCommandRegistryDispatch(t *testing.T) {
	cr := NewCommandRegistry()
	called := false

	cr.Register(game.CmdTogglePause, func(cmd game.GameCommand) {
		called = true
	})

	err := cr.Execute(game.GameCommand{Type: game.CmdTogglePause})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
}

func TestCommandRegistryUnknown(t *testing.T) {
	cr := NewCommandRegistry()

	err := cr.Execute(game.GameCommand{Type: "nonexistent"})
	if err == nil {
		t.Error("expected error for unknown command type")
	}
}

func TestCommandRegistryAllTypes(t *testing.T) {
	// Verify all command type constants are distinct
	types := []game.CommandType{
		game.CmdSave, game.CmdSetSpeed, game.CmdTogglePause,
		game.CmdTrade, game.CmdCargoLoad, game.CmdCargoUnload,
		game.CmdBuild, game.CmdBuildShip, game.CmdMoveShip,
		game.CmdUpgrade, game.CmdRefuel, game.CmdColonize,
		game.CmdRegisterPlayer, game.CmdWorkforceAssign,
		game.CmdCancelConstruction, game.CmdStandingOrder,
		game.CmdCancelOrder, game.CmdFleetMove, game.CmdFleetCreate,
		game.CmdFleetDisband, game.CmdFleetAddShip, game.CmdFleetRemoveShip,
	}

	seen := make(map[game.CommandType]bool)
	for _, ct := range types {
		if seen[ct] {
			t.Errorf("duplicate command type: %s", ct)
		}
		seen[ct] = true
	}

	if len(seen) != 22 {
		t.Errorf("expected 22 unique command types, got %d", len(seen))
	}
}
