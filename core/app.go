package core

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/server"
	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/ui"
	"github.com/hunterjsb/xandaris/views"
)

// App is the Ebiten GUI client. It embeds a GameServer for simulation
// and adds rendering, input handling, and view management on top.
type App struct {
	Server *server.GameServer

	viewManager *views.ViewManager
	keyBindings *systems.KeyBindings
	commandBar  *ui.CommandBar

	// Screen dimensions
	screenWidth  int
	screenHeight int

	// Empire panel click regions (planet ID → y range)
	empirePlanetHits []empirePlanetHit

	// Remote connection details (stored for UI components)
	remoteServerURL string
	remoteAPIKey    string

	// Toast notifications for important events
	notifications *notificationOverlay

	// Construction cache for remote mode
	constructionCacheMu *constructionCacheMu
}

type empirePlanetHit struct {
	PlanetID int
	Y1, Y2   int
	X1, X2   int
}

// New creates a new App instance with an in-process server.
func New(screenWidth, screenHeight int) *App {
	return &App{
		Server:       server.New(screenWidth, screenHeight),
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
	}
}

// GetState returns the game state (delegates to server).
func (a *App) GetState() *game.State {
	return a.Server.State
}

// GetViewManager returns the view manager.
func (a *App) GetViewManager() views.ViewManagerInterface {
	return a.viewManager
}

// GetTickManager returns the tick manager (delegates to server).
func (a *App) GetTickManager() views.TickManagerInterface {
	return a.Server.TickManager
}

// GetKeyBindings returns the key bindings.
func (a *App) GetKeyBindings() views.KeyBindingsInterface {
	return a.keyBindings
}

// IsRemote returns true if connected to a remote server (multiplayer).
func (a *App) IsRemote() bool {
	return a.Server.IsRemote()
}


// GetFleetCommander returns the fleet command interface (delegates to server).
func (a *App) GetFleetCommander() views.FleetCommandInterface {
	return a.Server
}

// GetFleetManagementSystem returns the fleet management system (delegates to server).
func (a *App) GetFleetManagementSystem() *game.FleetManagementSystem {
	return a.Server.FleetMgmtSystem
}

// SaveKeyBindings saves the current key bindings to config file.
func (a *App) SaveKeyBindings() error {
	if a.keyBindings == nil {
		return fmt.Errorf("key bindings not initialized")
	}
	return a.keyBindings.SaveToFile(systems.GetKeyBindingsConfigPath())
}

// Update updates the app state (implements ebiten.Game).
func (a *App) Update() error {
	// Command bar toggle (before other input so it can capture keys)
	if a.commandBar != nil && a.commandBar.IsOpen() {
		a.commandBar.Update()
	} else {
		// Handle global keyboard shortcuts (client-side input)
		a.handleGlobalInput()
		// Empire panel planet clicks
		a.handleEmpirePanelClick()
	}

	// Toggle command bar with T
	if a.commandBar != nil && a.keyBindings.IsActionJustPressed(views.ActionOpenCommandBar) {
		a.commandBar.Toggle()
	}

	// Drain commands and advance simulation (in-process server)
	a.Server.DrainCommands()
	if a.Server.TickManager != nil {
		a.Server.TickManager.Update()
	}

	// Skip view input when command bar is open (prevents hotkeys while typing)
	// The simulation still runs, and Draw still renders animations
	if a.commandBar != nil && a.commandBar.IsOpen() {
		return nil
	}
	return a.viewManager.Update()
}

// Draw draws the game screen (implements ebiten.Game).
func (a *App) Draw(screen *ebiten.Image) {
	a.viewManager.Draw(screen)
	a.drawTickInfo(screen)

	// Toast notifications (above command bar, below nothing)
	if a.notifications != nil {
		a.notifications.draw(screen, a.screenWidth)
	}

	// Command bar draws on top of everything
	if a.commandBar != nil {
		a.commandBar.Draw(screen)
	}
}

// Layout returns the game's logical screen size (implements ebiten.Game).
// We keep a fixed logical resolution and let Ebiten handle scaling to the window.
func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return a.screenWidth, a.screenHeight
}
