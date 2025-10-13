package views

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/entities"
)

// GameContext provides the interface that views need to interact with the game
// This avoids circular dependencies by not importing the main package
type GameContext interface {
	// Game state access
	GetSystems() []*entities.System
	GetHyperlanes() []entities.Hyperlane
	GetPlayers() []*entities.Player
	GetHumanPlayer() *entities.Player
	GetSeed() int64

	// View management
	GetViewManager() *ViewManager

	// Tick management
	GetTickManager() TickManager

	// Key bindings
	GetKeyBindings() KeyBindings

	// Save/Load
	SaveGame(playerName string) error

	// Construction handler
	RegisterConstructionHandler()
}

// TickManager interface for views to interact with game time
type TickManager interface {
	GetCurrentTick() int64
	GetGameTimeFormatted() string
	GetSpeed() int
	SetSpeed(speed int)
	TogglePause()
	IsPaused() bool
}

// KeyBindings interface for views to handle input
type KeyBindings interface {
	IsActionJustPressed(action string) bool
	GetAllActions() []string
	SetKey(action string, key ebiten.Key)
	GetKey(action string) (ebiten.Key, bool)
	SaveToFile(filename string) error
	LoadFromFile(filename string) error
}
