package ui

import (
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/views"
)

// UIContext defines what UI components need from the application
// This interface breaks the circular dependency between ui and core packages
type UIContext interface {
	// State access
	GetState() *game.State
	GetSystemsMap() map[int]*entities.System
	GetHyperlanes() []entities.Hyperlane

	// System managers (reuse interfaces from views package to avoid duplication)
	GetTickManager() views.TickManagerInterface
	GetFleetCommander() views.FleetCommandInterface
	GetFleetManagementSystem() *game.FleetManagementSystem
	GetKeyBindings() views.KeyBindingsInterface
}
