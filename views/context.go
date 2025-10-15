package views

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/entities"
)

// GameContext provides the minimal interface that views need to interact with the game
// This avoids circular dependencies by not importing the main package
type GameContext interface {
	// Game state access
	GetSystems() []*entities.System
	GetHyperlanes() []entities.Hyperlane
	GetPlayers() []*entities.Player
	GetHumanPlayer() *entities.Player
	GetSeed() int64

	// View management - returns the interface, not concrete type
	GetViewManager() ViewManagerInterface

	// Tick management - returns the interface
	GetTickManager() TickManagerInterface

	// Key bindings - returns the interface
	GetKeyBindings() KeyBindingsInterface

	// Save/Load interface
	GetSaveLoad() SaveLoadInterface

	// Fleet management interface
	GetFleetManager() FleetManagerInterface

	// Fleet command interface - for issuing commands to fleets
	GetFleetCommander() FleetCommandInterface

	// Game lifecycle methods (primarily for main menu)
	// These update the current game state in-place rather than replacing it
	InitializeNewGame(playerName string) error
	LoadGameFromPath(path string) error
}

// ViewManagerInterface defines what views need from the view manager
type ViewManagerInterface interface {
	SwitchTo(viewType ViewType)
	GetView(viewType ViewType) View
}

// TickManagerInterface defines what views need from the tick manager
type TickManagerInterface interface {
	GetCurrentTick() int64
	GetGameTimeFormatted() string
	GetSpeed() interface{}               // Returns TickSpeed type from main package
	GetSpeedFloat() float64              // Returns speed as float64 for animation calculations
	GetSpeedString() string              // Returns human-readable speed string
	GetEffectiveTicksPerSecond() float64 // Returns actual ticks per second considering speed
	SetSpeed(speed interface{})
	TogglePause()
	IsPaused() bool
	Reset()
}

// KeyBindingsInterface defines what views need from key bindings
type KeyBindingsInterface interface {
	IsActionJustPressed(action KeyAction) bool
	GetAllActions() []KeyAction
	SetKey(action KeyAction, key ebiten.Key)
	GetKey(action KeyAction) (ebiten.Key, bool)
	GetKeyName(key ebiten.Key) string
	SaveToFile(filename string) error
	LoadFromFile(filename string) error
	LoadDefaults() // Reset all bindings to defaults
}

// KeyAction represents a bindable action (views need to reference these)
type KeyAction string

// Key action constants that views will use
const (
	// Global actions
	ActionPauseToggle   KeyAction = "pause_toggle"
	ActionSpeedSlow     KeyAction = "speed_slow"
	ActionSpeedNormal   KeyAction = "speed_normal"
	ActionSpeedFast     KeyAction = "speed_fast"
	ActionSpeedVeryFast KeyAction = "speed_very_fast"
	ActionQuickSave     KeyAction = "quick_save"

	// View navigation
	ActionEscape        KeyAction = "escape"
	ActionOpenBuildMenu KeyAction = "open_build_menu"
	ActionOpenMarket    KeyAction = "open_market"
	ActionOpenPlayerDir KeyAction = "open_player_directory"
	ActionFocusHome     KeyAction = "focus_home_system"

	// Menu navigation
	ActionMenuUp      KeyAction = "menu_up"
	ActionMenuDown    KeyAction = "menu_down"
	ActionMenuConfirm KeyAction = "menu_confirm"
	ActionMenuCancel  KeyAction = "menu_cancel"
	ActionMenuDelete  KeyAction = "menu_delete"
)

// SaveFileInfo contains metadata about a save file
type SaveFileInfo struct {
	Filename   string
	Path       string
	PlayerName string
	GameTime   string
	SavedAt    time.Time
	ModTime    time.Time
}

// SaveLoadInterface defines save/load operations
type SaveLoadInterface interface {
	ListSaveFiles() ([]SaveFileInfo, error)
	GetSaveFileInfo(path string) (SaveFileInfo, error)
}
