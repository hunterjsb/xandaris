package main

import (
	"encoding/gob"
	"fmt"
	"image/color"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/entities"
	_ "github.com/hunterjsb/xandaris/entities/building"
	_ "github.com/hunterjsb/xandaris/entities/planet"
	_ "github.com/hunterjsb/xandaris/entities/resource"
	_ "github.com/hunterjsb/xandaris/entities/star"
	_ "github.com/hunterjsb/xandaris/entities/station"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/tickable"
	_ "github.com/hunterjsb/xandaris/tickable" // Import tickable systems for auto-registration
	"github.com/hunterjsb/xandaris/views"
)

// StartMode represents how the game was started
type StartMode int

const (
	StartModeMenu StartMode = iota
	StartModeNewGame
)

const (
	screenWidth  = 1280
	screenHeight = 720
	circleRadius = 8
)

// GameSystemContext implements tickable.SystemContext
type GameSystemContext struct {
	game *Game
}

func (gsc *GameSystemContext) GetGame() interface{} {
	return gsc.game
}

func (gsc *GameSystemContext) GetPlayers() interface{} {
	return gsc.game.players
}

func (gsc *GameSystemContext) GetTick() int64 {
	return gsc.game.tickManager.GetCurrentTick()
}

// Game implements ebiten.Game interface and views.GameContext
type Game struct {
	systems           []*entities.System
	hyperlanes        []entities.Hyperlane
	viewManager       *views.ViewManager
	seed              int64
	players           []*entities.Player
	humanPlayer       *entities.Player
	tickManager       *systems.TickManager
	keyBindings       *systems.KeyBindings
	fleetManager      *systems.FleetManager
	fleetCmdExecutor  *game.FleetCommandExecutor
}

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		systems:    make([]*entities.System, 0),
		hyperlanes: make([]entities.Hyperlane, 0),
		seed:       time.Now().UnixNano(),
		players:    make([]*entities.Player, 0),
	}

	// Initialize key bindings
	g.keyBindings = systems.NewKeyBindings()
	// Try to load custom key bindings from config
	if err := g.keyBindings.LoadFromFile(systems.GetKeyBindingsConfigPath()); err != nil {
		fmt.Printf("Warning: Could not load key bindings: %v\n", err)
		fmt.Println("Using default key bindings")
	}

	// Initialize tick system (10 ticks per second at 1x speed)
	g.tickManager = systems.NewTickManager(10.0)

	// Generate galaxy data
	galaxyGen := game.NewGalaxyGenerator(screenWidth, screenHeight)
	g.systems = galaxyGen.GenerateSystems(g.seed)
	g.hyperlanes = galaxyGen.GenerateHyperlanes(g.systems)

	// Create human player
	playerColor := color.RGBA{100, 200, 100, 255} // Green for player
	g.humanPlayer = entities.NewPlayer(0, "Player", playerColor, entities.PlayerTypeHuman)
	g.players = append(g.players, g.humanPlayer)

	// Initialize player with starting planet
	entities.InitializePlayer(g.humanPlayer, g.systems)

	// Initialize tickable systems
	context := &GameSystemContext{game: g}
	tickable.InitializeAllSystems(context)

	// Register construction completion handler
	g.registerConstructionHandler()

	// Initialize fleet manager
	g.fleetManager = systems.NewFleetManager(g)

	// Initialize fleet command executor
	g.fleetCmdExecutor = game.NewFleetCommandExecutor(g.systems, g.hyperlanes)

	// Initialize view system
	g.viewManager = views.NewViewManager()

	// Create UI components (stay in main package)
	buildMenu := NewBuildMenu(g)
	constructionQueue := NewConstructionQueueUI(g)
	resourceStorage := NewResourceStorageUI(g)
	shipyardUI := NewShipyardUI(g)
	fleetInfoUI := NewFleetInfoUI(g)

	// Create and register views (pass Game as GameContext)
	galaxyView := views.NewGalaxyView(g)
	systemView := views.NewSystemView(g, fleetInfoUI)
	planetView := views.NewPlanetView(g, buildMenu, constructionQueue, resourceStorage, shipyardUI, fleetInfoUI)
	mainMenuView := views.NewMainMenuView(g)
	settingsView := views.NewSettingsView(g)

	g.viewManager.RegisterView(galaxyView)
	g.viewManager.RegisterView(systemView)
	g.viewManager.RegisterView(planetView)
	g.viewManager.RegisterView(mainMenuView)
	g.viewManager.RegisterView(settingsView)

	// Start with galaxy view
	g.viewManager.SwitchTo(views.ViewTypeGalaxy)

	return g
}

// NewGameForMenu creates a minimal game instance for the main menu
func NewGameForMenu() *Game {
	g := &Game{
		systems:    make([]*entities.System, 0),
		hyperlanes: make([]entities.Hyperlane, 0),
		players:    make([]*entities.Player, 0),
	}

	// Initialize key bindings (needed for menu navigation)
	g.keyBindings = systems.NewKeyBindings()
	// Try to load custom key bindings from config
	if err := g.keyBindings.LoadFromFile(systems.GetKeyBindingsConfigPath()); err != nil {
		// Silently use defaults if config doesn't exist
		// (Don't print warnings in menu since game hasn't started yet)
	}

	// Initialize tick manager for menu (though it won't really be used)
	g.tickManager = systems.NewTickManager(10.0)

	// Initialize fleet manager (empty for menu)
	g.fleetManager = systems.NewFleetManager(g)

	// Initialize fleet command executor (empty for menu)
	g.fleetCmdExecutor = game.NewFleetCommandExecutor(g.systems, g.hyperlanes)

	// Initialize view system
	g.viewManager = views.NewViewManager()

	// Create UI components (stay in main package)
	buildMenu := NewBuildMenu(g)
	constructionQueue := NewConstructionQueueUI(g)
	resourceStorage := NewResourceStorageUI(g)
	shipyardUI := NewShipyardUI(g)
	fleetInfoUI := NewFleetInfoUI(g)

	// Create and register all views
	mainMenuView := views.NewMainMenuView(g)
	galaxyView := views.NewGalaxyView(g)
	systemView := views.NewSystemView(g, fleetInfoUI)
	planetView := views.NewPlanetView(g, buildMenu, constructionQueue, resourceStorage, shipyardUI, fleetInfoUI)
	settingsView := views.NewSettingsView(g)

	g.viewManager.RegisterView(mainMenuView)
	g.viewManager.RegisterView(galaxyView)
	g.viewManager.RegisterView(systemView)
	g.viewManager.RegisterView(planetView)
	g.viewManager.RegisterView(settingsView)

	// Start with main menu
	g.viewManager.SwitchTo(views.ViewTypeMainMenu)

	return g
}

// GetPlayers returns the game's players
func (g *Game) GetPlayers() []*entities.Player {
	return g.players
}

// GetSystemsMap returns a map of systems indexed by ID (internal use)
func (g *Game) GetSystemsMap() map[int]*entities.System {
	systemsMap := make(map[int]*entities.System)
	for _, system := range g.systems {
		systemsMap[system.ID] = system
	}
	return systemsMap
}

// GetHyperlanes returns all hyperlanes
func (g *Game) GetHyperlanes() []entities.Hyperlane {
	return g.hyperlanes
}

// GameContext interface implementation

// GetSystems returns the game's systems (implements GameContext)
func (g *Game) GetSystems() []*entities.System {
	return g.systems
}

// GetHumanPlayer returns the human player
func (g *Game) GetHumanPlayer() *entities.Player {
	return g.humanPlayer
}

// GetSeed returns the game seed
func (g *Game) GetSeed() int64 {
	return g.seed
}

// GetViewManager returns the view manager interface
func (g *Game) GetViewManager() views.ViewManagerInterface {
	return g.viewManager
}

// GetTickManager returns the tick manager interface
func (g *Game) GetTickManager() views.TickManagerInterface {
	return g.tickManager
}

// GetKeyBindings returns the key bindings interface
func (g *Game) GetKeyBindings() views.KeyBindingsInterface {
	return g.keyBindings
}

// GetSaveLoad returns the save/load interface
func (g *Game) GetSaveLoad() views.SaveLoadInterface {
	return g // Game itself implements SaveLoadInterface
}

// GetFleetManager returns the fleet manager interface
func (g *Game) GetFleetManager() views.FleetManagerInterface {
	return g.fleetManager
}

// GetFleetCommander returns the fleet command interface (Game implements it)
func (g *Game) GetFleetCommander() views.FleetCommandInterface {
	return g
}

// Fleet command interface implementation - delegates to FleetCommandExecutor

// MoveFleetToSystem attempts to move all ships in a fleet to another system
func (g *Game) MoveFleetToSystem(fleet *entities.Fleet, targetSystemID int) (int, int) {
	return g.fleetCmdExecutor.MoveFleetToSystem(fleet, targetSystemID)
}

// MoveFleetToPlanet moves all ships in a fleet to orbit a specific planet
func (g *Game) MoveFleetToPlanet(fleet *entities.Fleet, targetPlanet *entities.Planet) (int, int) {
	return g.fleetCmdExecutor.MoveFleetToPlanet(fleet, targetPlanet)
}

// MoveFleetToStar moves all ships in a fleet to orbit the system's star
func (g *Game) MoveFleetToStar(fleet *entities.Fleet) (int, int) {
	return g.fleetCmdExecutor.MoveFleetToStar(fleet)
}

// GetConnectedSystems returns system IDs connected to the given system via hyperlanes
func (g *Game) GetConnectedSystems(fromSystemID int) []int {
	return g.fleetCmdExecutor.GetConnectedSystems(fromSystemID)
}

// GetSystemByID returns a system by its ID
func (g *Game) GetSystemByID(systemID int) *entities.System {
	return g.fleetCmdExecutor.GetSystemByID(systemID)
}

// InitializeNewGame initializes a new game with the given player name
func (g *Game) InitializeNewGame(playerName string) error {
	// Reset game state
	g.systems = make([]*entities.System, 0)
	g.hyperlanes = make([]entities.Hyperlane, 0)
	g.seed = time.Now().UnixNano()
	g.players = make([]*entities.Player, 0)

	// Reset tick manager
	g.tickManager.Reset()

	// Generate galaxy data
	galaxyGen := game.NewGalaxyGenerator(screenWidth, screenHeight)
	g.systems = galaxyGen.GenerateSystems(g.seed)
	g.hyperlanes = galaxyGen.GenerateHyperlanes(g.systems)

	// Create human player
	playerColor := color.RGBA{100, 200, 100, 255} // Green for player
	g.humanPlayer = entities.NewPlayer(0, playerName, playerColor, entities.PlayerTypeHuman)
	g.players = append(g.players, g.humanPlayer)

	// Initialize player with starting planet
	entities.InitializePlayer(g.humanPlayer, g.systems)

	// Initialize tickable systems
	context := &GameSystemContext{game: g}
	tickable.InitializeAllSystems(context)

	// Register construction completion handler
	g.registerConstructionHandler()

	// Update fleet command executor with new systems/hyperlanes
	g.fleetCmdExecutor = game.NewFleetCommandExecutor(g.systems, g.hyperlanes)

	return nil
}

// LoadGameFromPath loads a game from the given path
func (g *Game) LoadGameFromPath(path string) error {
	fmt.Printf("[SaveSystem] Loading game from: %s\n", path)

	// Open file
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open save file: %w", err)
	}
	defer file.Close()

	// Get file info for debugging
	fileInfo, _ := file.Stat()
	fmt.Printf("[SaveSystem] Reading %d bytes from save file\n", fileInfo.Size())

	// Create gob decoder
	decoder := gob.NewDecoder(file)

	// Decode save data
	var saveData struct {
		Version            string
		SavedAt            time.Time
		PlayerName         string
		GameTime           string
		Tick               int64
		Seed               int64
		TickSpeed          systems.TickSpeed
		Systems            []*entities.System
		Hyperlanes         []entities.Hyperlane
		Players            []*entities.Player
		ConstructionQueues map[string][]*tickable.ConstructionItem
	}

	if err := decoder.Decode(&saveData); err != nil {
		return fmt.Errorf("failed to decode save data: %w", err)
	}

	fmt.Printf("[SaveSystem] Decoded save data: version=%s, player=%s, systems=%d\n",
		saveData.Version, saveData.PlayerName, len(saveData.Systems))

	// Update game state
	g.systems = saveData.Systems
	g.hyperlanes = saveData.Hyperlanes
	g.seed = saveData.Seed
	g.players = saveData.Players

	// Update tick manager with saved state
	g.tickManager.SetSpeed(saveData.TickSpeed)
	g.tickManager.SetCurrentTick(saveData.Tick)

	// Find human player
	g.humanPlayer = nil
	for _, player := range g.players {
		if player.Type == entities.PlayerTypeHuman {
			g.humanPlayer = player
			break
		}
	}

	// Rebuild planet owner references
	for _, player := range g.players {
		for _, planet := range player.OwnedPlanets {
			planet.Owner = player.Name
		}
	}

	// Initialize tickable systems
	context := &GameSystemContext{game: g}
	tickable.InitializeAllSystems(context)

	// Restore construction queues
	if saveData.ConstructionQueues != nil && len(saveData.ConstructionQueues) > 0 {
		if constructionSystem := tickable.GetSystemByName("Construction"); constructionSystem != nil {
			if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
				cs.RestoreQueues(saveData.ConstructionQueues)
				fmt.Printf("[SaveSystem] Restored %d construction queues\n", len(saveData.ConstructionQueues))
			}
		}
	}

	g.registerConstructionHandler()

	// Update fleet command executor with loaded systems/hyperlanes
	g.fleetCmdExecutor = game.NewFleetCommandExecutor(g.systems, g.hyperlanes)

	fmt.Printf("[SaveSystem] Game loaded successfully\n")
	return nil
}

// SaveLoadInterface implementation

// ListSaveFiles returns all save files
func (g *Game) ListSaveFiles() ([]views.SaveFileInfo, error) {
	files, err := ListSaveFiles()
	if err != nil {
		return nil, err
	}

	// Convert from main.SaveFileInfo to views.SaveFileInfo
	viewFiles := make([]views.SaveFileInfo, len(files))
	for i, f := range files {
		viewFiles[i] = views.SaveFileInfo{
			Filename:   f.Filename,
			Path:       f.Path,
			PlayerName: f.PlayerName,
			GameTime:   f.GameTime,
			SavedAt:    f.SavedAt,
			ModTime:    f.ModTime,
		}
	}
	return viewFiles, nil
}

// GetSaveFileInfo gets info about a specific save file
func (g *Game) GetSaveFileInfo(path string) (views.SaveFileInfo, error) {
	info, err := GetSaveFileInfo(path)
	if err != nil {
		return views.SaveFileInfo{}, err
	}

	// Convert from main.SaveFileInfo to views.SaveFileInfo
	return views.SaveFileInfo{
		Filename:   info.Filename,
		Path:       info.Path,
		PlayerName: info.PlayerName,
		GameTime:   info.GameTime,
		SavedAt:    info.SavedAt,
		ModTime:    info.ModTime,
	}, nil
}

// Update updates the game state
func (g *Game) Update() error {
	// Handle global keyboard shortcuts
	g.handleGlobalInput()

	// Update tick system (this will also update tickable systems)
	g.tickManager.Update()

	// Update current view
	return g.viewManager.Update()
}

// handleGlobalInput handles keyboard input for game-wide controls
func (g *Game) handleGlobalInput() {
	// Don't handle game controls in main menu
	if g.viewManager.GetCurrentView().GetType() == views.ViewTypeMainMenu {
		return
	}

	// Toggle pause
	if g.keyBindings.IsActionJustPressed(views.ActionPauseToggle) {
		g.tickManager.TogglePause()
	}

	// Speed control
	if g.keyBindings.IsActionJustPressed(views.ActionSpeedSlow) {
		g.tickManager.SetSpeed(systems.TickSpeed1x)
	}
	if g.keyBindings.IsActionJustPressed(views.ActionSpeedNormal) {
		g.tickManager.SetSpeed(systems.TickSpeed2x)
	}
	if g.keyBindings.IsActionJustPressed(views.ActionSpeedFast) {
		g.tickManager.SetSpeed(systems.TickSpeed4x)
	}
	if g.keyBindings.IsActionJustPressed(views.ActionSpeedVeryFast) {
		g.tickManager.SetSpeed(systems.TickSpeed8x)
	}

	// Cycle speed
	if g.keyBindings.IsActionJustPressed(views.ActionSpeedIncrease) {
		g.tickManager.CycleSpeed()
	}

	// Quick save
	if g.keyBindings.IsActionJustPressed(views.ActionQuickSave) {
		if g.humanPlayer != nil {
			err := g.SaveGameToFile(g.humanPlayer.Name)
			if err != nil {
				fmt.Printf("Failed to save game: %v\n", err)
			} else {
				fmt.Println("Game saved successfully!")
			}
		}
	}
}

// Draw draws the game screen
func (g *Game) Draw(screen *ebiten.Image) {
	g.viewManager.Draw(screen)

	// Draw tick info overlay
	g.drawTickInfo(screen)
}

// drawTickInfo draws tick information overlay
func (g *Game) drawTickInfo(screen *ebiten.Image) {
	// Don't draw in main menu
	if g.viewManager.GetCurrentView().GetType() == views.ViewTypeMainMenu {
		return
	}

	// Draw in bottom-left corner
	x := 10
	y := screenHeight - 60

	// Create small panel
	panel := NewUIPanel(x, y, 200, 50)
	panel.Draw(screen)

	// Draw tick info
	textX := x + 10
	textY := y + 15

	speedStr := g.tickManager.GetSpeedString()
	DrawText(screen, "Speed: "+speedStr, textX, textY, UITextPrimary)
	DrawText(screen, g.tickManager.GetGameTimeFormatted(), textX, textY+15, UITextSecondary)
	DrawText(screen, "[Space] Pause  [F5] Save", textX, textY+30, UITextSecondary)
}

// Layout returns the game's screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// SaveKeyBindings saves the current key bindings to config file
func (g *Game) SaveKeyBindings() error {
	if g.keyBindings == nil {
		return fmt.Errorf("key bindings not initialized")
	}
	return g.keyBindings.SaveToFile(systems.GetKeyBindingsConfigPath())
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Xandaris II - Space Trading Game")

	// Start with main menu
	game := NewGameForMenu()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// registerConstructionHandler sets up handler for completed constructions
func (g *Game) registerConstructionHandler() {
	handler := game.NewConstructionHandler(g.systems, g.players, g.tickManager, g)
	constructionSystem := tickable.GetSystemByName("Construction")
	if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
		cs.RegisterCompletionHandler(handler.HandleConstructionComplete)
	}
}

// RefreshPlanetViewIfActive refreshes planet view if the given planet is currently displayed
func (g *Game) RefreshPlanetViewIfActive(planet *entities.Planet) {
	// TODO: Re-implement once PlanetView is fully ported
	// if g.viewManager.GetCurrentView().GetType() == views.ViewTypePlanet {
	// 	if planetView, ok := g.viewManager.GetCurrentView().(*views.PlanetView); ok {
	// 		if planetView.planet == planet {
	// 			planetView.RefreshPlanet()
	// 		}
	// 	}
	// }
}

