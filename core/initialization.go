package core

import (
	"image/color"
	"time"

	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/ui"
	"github.com/hunterjsb/xandaris/views"
)

// GameSystemContext implements tickable.SystemContext
type GameSystemContext struct {
	app *App
}

func (gsc *GameSystemContext) GetGame() interface{} {
	return gsc.app
}

func (gsc *GameSystemContext) GetPlayers() interface{} {
	return gsc.app.state.Players
}

func (gsc *GameSystemContext) GetTick() int64 {
	return gsc.app.tickManager.GetCurrentTick()
}

// InitializeForMenu initializes the app with minimal state for the main menu
func (a *App) InitializeForMenu() error {
	// Initialize key bindings (needed for menu navigation)
	a.keyBindings = systems.NewKeyBindings()
	// Try to load custom key bindings from config
	if err := a.keyBindings.LoadFromFile(systems.GetKeyBindingsConfigPath()); err != nil {
		// Silently use defaults if config doesn't exist
	}

	// Initialize tick manager for menu (though it won't really be used)
	a.tickManager = systems.NewTickManager(10.0)

	// Initialize fleet manager (empty for menu)
	a.fleetManager = systems.NewFleetManager(a)

	// Initialize fleet command executor (empty for menu)
	a.fleetCmdExecutor = game.NewFleetCommandExecutor(a.state.Systems, a.state.Hyperlanes)

	// Initialize view system
	a.viewManager = views.NewViewManager()

	// Create UI components
	buildMenu := ui.NewBuildMenu(a)
	constructionQueue := ui.NewConstructionQueueUI(a)
	resourceStorage := ui.NewResourceStorageUI(a)
	shipyardUI := ui.NewShipyardUI(a)
	fleetInfoUI := ui.NewFleetInfoUI(a)

	// Initialize and register all views
	a.initializeViews(buildMenu, constructionQueue, resourceStorage, shipyardUI, fleetInfoUI)

	// Start with main menu
	a.viewManager.SwitchTo(views.ViewTypeMainMenu)

	return nil
}

// InitializeNewGame initializes a new game with the given player name
func (a *App) InitializeNewGame(playerName string) error {
	// Reset game state
	a.state.Reset()
	a.state.Seed = time.Now().UnixNano()

	// Reset tick manager
	a.tickManager.Reset()

	// Generate galaxy data
	galaxyGen := game.NewGalaxyGenerator(a.screenWidth, a.screenHeight)
	a.state.Systems = galaxyGen.GenerateSystems(a.state.Seed)
	a.state.Hyperlanes = galaxyGen.GenerateHyperlanes(a.state.Systems)

	// Create human player
	playerColor := color.RGBA{100, 200, 100, 255} // Green for player
	a.state.HumanPlayer = entities.NewPlayer(0, playerName, playerColor, entities.PlayerTypeHuman)
	a.state.Players = append(a.state.Players, a.state.HumanPlayer)

	// Initialize player with starting planet
	entities.InitializePlayer(a.state.HumanPlayer, a.state.Systems)

	// Initialize tickable systems
	context := &GameSystemContext{app: a}
	tickable.InitializeAllSystems(context)

	// Register construction completion handler
	a.registerConstructionHandler()

	// Update fleet command executor with new systems/hyperlanes
	a.fleetCmdExecutor = game.NewFleetCommandExecutor(a.state.Systems, a.state.Hyperlanes)

	// Switch to galaxy view after game initialization
	a.viewManager.SwitchTo(views.ViewTypeGalaxy)

	return nil
}

// registerConstructionHandler sets up handler for completed constructions
func (a *App) registerConstructionHandler() {
	handler := game.NewConstructionHandler(a.state.Systems, a.state.Players, a.tickManager, a)
	constructionSystem := tickable.GetSystemByName("Construction")
	if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
		cs.RegisterCompletionHandler(handler.HandleConstructionComplete)
	}
}

// RefreshPlanetViewIfActive refreshes planet view if the given planet is currently displayed
func (a *App) RefreshPlanetViewIfActive(planet *entities.Planet) {
	// TODO: Re-implement once PlanetView is fully ported
	// if a.viewManager.GetCurrentView().GetType() == views.ViewTypePlanet {
	// 	if planetView, ok := a.viewManager.GetCurrentView().(*views.PlanetView); ok {
	// 		if planetView.planet == planet {
	// 			planetView.RefreshPlanet()
	// 		}
	// 	}
	// }
}
