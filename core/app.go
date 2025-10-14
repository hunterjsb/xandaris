package core

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/views"
)

// App orchestrates all game components and implements ebiten.Game
type App struct {
	state            *game.State
	viewManager      *views.ViewManager
	tickManager      *systems.TickManager
	keyBindings      *systems.KeyBindings
	fleetManager     *systems.FleetManager
	fleetCmdExecutor *game.FleetCommandExecutor

	// Screen dimensions
	screenWidth  int
	screenHeight int
}

// New creates a new App instance
func New(screenWidth, screenHeight int) *App {
	return &App{
		state:        game.NewState(),
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
	}
}

// GetState returns the game state
func (a *App) GetState() *game.State {
	return a.state
}

// GetViewManager returns the view manager
func (a *App) GetViewManager() views.ViewManagerInterface {
	return a.viewManager
}

// GetTickManager returns the tick manager
func (a *App) GetTickManager() views.TickManagerInterface {
	return a.tickManager
}

// GetKeyBindings returns the key bindings
func (a *App) GetKeyBindings() views.KeyBindingsInterface {
	return a.keyBindings
}

// GetFleetManager returns the fleet manager
func (a *App) GetFleetManager() views.FleetManagerInterface {
	return a.fleetManager
}

// GetFleetCommander returns the fleet command interface (App implements it)
func (a *App) GetFleetCommander() views.FleetCommandInterface {
	return a
}

// SaveKeyBindings saves the current key bindings to config file
func (a *App) SaveKeyBindings() error {
	if a.keyBindings == nil {
		return fmt.Errorf("key bindings not initialized")
	}
	return a.keyBindings.SaveToFile(systems.GetKeyBindingsConfigPath())
}

// Update updates the app state (implements ebiten.Game)
func (a *App) Update() error {
	// Handle global keyboard shortcuts
	a.handleGlobalInput()

	// Update tick system (this will also update tickable systems)
	if a.tickManager != nil {
		a.tickManager.Update()
	}

	// Update current view
	return a.viewManager.Update()
}

// Draw draws the game screen (implements ebiten.Game)
func (a *App) Draw(screen *ebiten.Image) {
	a.viewManager.Draw(screen)

	// Draw tick info overlay
	a.drawTickInfo(screen)
}

// Layout returns the game's screen size (implements ebiten.Game)
func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return a.screenWidth, a.screenHeight
}
