package core

import (
	"fmt"

	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/ui"
	"github.com/hunterjsb/xandaris/views"
)

// InitializeForMenu initializes the app with minimal state for the main menu.
func (a *App) InitializeForMenu() error {
	// Initialize key bindings (needed for menu navigation)
	a.keyBindings = systems.NewKeyBindings()
	if err := a.keyBindings.LoadFromFile(systems.GetKeyBindingsConfigPath()); err != nil {
		fmt.Println("Failed to load custom key bindings:", err)
	}

	// Initialize view system
	a.viewManager = views.NewViewManager()

	// Initialize and register menu views
	a.initializeViews()

	// Start with main menu
	a.viewManager.SwitchTo(views.ViewTypeMainMenu)

	return nil
}

// initializeGameViews creates and registers all game views with UI components.
// Called after server has initialized game state.
func (a *App) initializeGameViews(buildMenu *ui.BuildMenu, constructionQueue *ui.ConstructionQueueUI,
	resourceStorage *ui.ResourceStorageUI, shipyardUI *ui.ShipyardUI, fleetInfoUI *ui.FleetInfoUI) {

	galaxyView := views.NewGalaxyView(a)
	systemView := views.NewSystemView(a, fleetInfoUI)
	planetView := views.NewPlanetView(a, buildMenu, constructionQueue, resourceStorage, shipyardUI, fleetInfoUI)
	marketView := views.NewMarketView(a)
	playerDirectory := views.NewPlayerDirectoryView(a)

	a.viewManager.RegisterView(galaxyView)
	a.viewManager.RegisterView(systemView)
	a.viewManager.RegisterView(planetView)
	a.viewManager.RegisterView(marketView)
	a.viewManager.RegisterView(playerDirectory)
}

// InitializeClientViews sets up UI components and registers game views.
// Called after the server has a game loaded/created. Exported for remote play.
func (a *App) InitializeClientViews() {
	buildMenu := ui.NewBuildMenu(a)
	constructionQueue := ui.NewConstructionQueueUI(a)
	resourceStorage := ui.NewResourceStorageUI(a)
	shipyardUI := ui.NewShipyardUI(a)
	fleetInfoUI := ui.NewFleetInfoUI(a)

	a.initializeGameViews(buildMenu, constructionQueue, resourceStorage, shipyardUI, fleetInfoUI)

	// Command bar (overlay on all views)
	a.commandBar = ui.NewCommandBar(a, a.screenWidth, a.screenHeight)
}

// SwitchToGalaxyView switches the view to the galaxy view. Used for remote play startup.
func (a *App) SwitchToGalaxyView() {
	a.viewManager.SwitchTo(views.ViewTypeGalaxy)
}

// InitializeNewGame creates a new game via the server and sets up client views.
func (a *App) InitializeNewGame(playerName string) error {
	if err := a.Server.NewGame(playerName); err != nil {
		return err
	}

	// Set up client-side views
	a.InitializeClientViews()

	// Switch to galaxy view
	a.viewManager.SwitchTo(views.ViewTypeGalaxy)

	return nil
}

// LoadGameFromPath loads a game via the server and sets up client views.
func (a *App) LoadGameFromPath(path string) error {
	if err := a.Server.LoadGame(path); err != nil {
		return err
	}

	// Set up client-side views
	a.InitializeClientViews()

	return nil
}
