package core

import (
	"fmt"
	"time"

	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/ui"
	"github.com/hunterjsb/xandaris/utils"
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
		fmt.Println("Failed to load custom key bindings:", err)
	}

	// Initialize tick manager (needed for game loading, even if not actively ticking in menu)
	a.tickManager = systems.NewTickManager(10.0)

	// Initialize view system
	a.viewManager = views.NewViewManager()

	// Initialize and register all views
	a.initializeViews()

	// Start with main menu
	a.viewManager.SwitchTo(views.ViewTypeMainMenu)

	return nil
}

// InitializeForGame initializes the app with the state for a new game
func (a *App) initializeGameComponents() {
	// Initialize fleet command executor
	a.fleetCmdExecutor = game.NewFleetCommandExecutor(a.state.Systems, a.state.Hyperlanes)

	// Initialize fleet management system
	a.fleetMgmtSystem = game.NewFleetManagementSystem(a.state)

	// Create UI components
	buildMenu := ui.NewBuildMenu(a)
	constructionQueue := ui.NewConstructionQueueUI(a)
	resourceStorage := ui.NewResourceStorageUI(a)
	shipyardUI := ui.NewShipyardUI(a)
	fleetInfoUI := ui.NewFleetInfoUI(a)

	// Initialize and register all game views with actual UI components
	a.initializeGameViews(buildMenu, constructionQueue, resourceStorage, shipyardUI, fleetInfoUI)
}

// InitializeNewGame initializes a new game with the given player name
func (a *App) InitializeNewGame(playerName string) error {
	// Reset game state
	a.state.Reset()
	a.state.Seed = time.Now().UnixNano()

	// Initialize game-specific components
	a.initializeGameComponents()

	// Reset tick manager
	a.tickManager.Reset()

	// Generate galaxy data
	galaxyGen := game.NewGalaxyGenerator(a.screenWidth, a.screenHeight)
	a.state.Systems = galaxyGen.GenerateSystems(a.state.Seed)
	a.state.Hyperlanes = galaxyGen.GenerateHyperlanes(a.state.Systems)

	// Create human player
	playerColor := utils.PlayerGreen
	a.state.HumanPlayer = entities.NewPlayer(0, playerName, playerColor, entities.PlayerTypeHuman)
	a.state.Players = append(a.state.Players, a.state.HumanPlayer)

	// Initialize player with starting planet
	entities.InitializePlayer(a.state.HumanPlayer, a.state.Systems)

	// Seed AI factions to populate the market
	a.initializeAIPlayers()

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
	handler := game.NewConstructionHandler(a.state.Systems, a.state.Players, a.tickManager)
	constructionSystem := tickable.GetSystemByName("Construction")
	if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
		cs.RegisterCompletionHandler(handler.HandleConstructionComplete)
	}
}
