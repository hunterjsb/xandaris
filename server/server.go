package server

import (
	"fmt"
	"time"

	"github.com/hunterjsb/xandaris/api"
	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/utils"
)

// GameServer is a headless game simulation server.
// It owns all game state and runs the tick loop without any rendering.
// Both the GUI client (core.App) and headless mode use this.
type GameServer struct {
	State            *game.State
	TickManager      *systems.TickManager
	FleetCmdExecutor *game.FleetCommandExecutor
	FleetMgmtSystem  *game.FleetManagementSystem
	CargoCommander   *game.CargoCommandExecutor
	Events           *game.EventLog

	screenWidth  int
	screenHeight int

	stopCh chan struct{}
}

// New creates a new GameServer.
func New(screenWidth, screenHeight int) *GameServer {
	return &GameServer{
		State:        game.NewState(),
		TickManager:  systems.NewTickManager(10.0),
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		stopCh:       make(chan struct{}),
	}
}

// NewGame initializes a new game with the given player name.
func (gs *GameServer) NewGame(playerName string) error {
	gs.State.Reset()
	gs.State.Seed = time.Now().UnixNano()

	// Reset tick manager
	gs.TickManager.Reset()

	// Generate galaxy
	galaxyGen := game.NewGalaxyGenerator(gs.screenWidth, gs.screenHeight)
	gs.State.Systems = galaxyGen.GenerateSystems(gs.State.Seed)
	gs.State.Hyperlanes = galaxyGen.GenerateHyperlanes(gs.State.Systems)

	// Create human player
	playerColor := utils.PlayerGreen
	gs.State.HumanPlayer = entities.NewPlayer(0, playerName, playerColor, entities.PlayerTypeHuman)
	gs.State.Players = append(gs.State.Players, gs.State.HumanPlayer)

	// Initialize player with starting planet
	entities.InitializePlayer(gs.State.HumanPlayer, gs.State.Systems)

	// Prepare human homeworld (Trading Post + seeded commodities, no auto-mines)
	game.PrepareHomeworld(gs.State.HumanPlayer, false)

	// Extra starting resources for human player — enough for early infrastructure
	if gs.State.HumanPlayer.HomePlanet != nil {
		gs.State.HumanPlayer.HomePlanet.AddStoredResource("Fuel", 200)
		gs.State.HumanPlayer.HomePlanet.AddStoredResource("Oil", 150)
	}

	// Create economy
	gs.State.Market = economy.NewMarket()
	gs.State.TradeExec = economy.NewTradeExecutor(gs.State.Market)

	// Seed AI factions
	game.InitializeAIPlayers(gs.State)

	// Initialize simulation components
	gs.initSimulation()

	// Start API server
	api.StartServer(gs)

	fmt.Printf("[Server] New game started for %s (%d systems, %d players)\n",
		playerName, len(gs.State.Systems), len(gs.State.Players))

	return nil
}

// initSimulation sets up fleet/cargo commanders, tickable systems, and construction handler.
func (gs *GameServer) initSimulation() {
	gs.Events = game.NewEventLog(100)
	gs.FleetCmdExecutor = game.NewFleetCommandExecutor(gs.State.Systems, gs.State.Hyperlanes)
	gs.FleetMgmtSystem = game.NewFleetManagementSystem(gs.State)
	gs.CargoCommander = game.NewCargoCommandExecutor(gs.State.Systems)

	if gs.State.TradeExec != nil {
		gs.State.TradeExec.SetSystems(gs.State.Systems)
	}

	// Initialize tickable systems
	ctx := &serverSystemContext{server: gs}
	tickable.InitializeAllSystems(ctx)

	// Register construction handler
	handler := game.NewConstructionHandler(gs.State.Systems, gs.State.Players, gs.TickManager)
	if constructionSystem := tickable.GetSystemByName("Construction"); constructionSystem != nil {
		if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
			cs.RegisterCompletionHandler(handler.HandleConstructionComplete)
		}
	}
}

// Run starts the headless game loop. Blocks until Stop() is called.
func (gs *GameServer) Run() {
	ticker := time.NewTicker(16 * time.Millisecond) // ~60fps simulation
	defer ticker.Stop()

	fmt.Println("[Server] Simulation loop started")
	for {
		select {
		case <-gs.stopCh:
			fmt.Println("[Server] Simulation loop stopped")
			return
		case <-ticker.C:
			gs.DrainCommands()
			gs.TickManager.Update()
		}
	}
}

// Stop signals the server to stop its simulation loop.
func (gs *GameServer) Stop() {
	close(gs.stopCh)
}

// DrainCommands processes all pending commands from the command channel.
func (gs *GameServer) DrainCommands() {
	if gs.State == nil || gs.State.Commands == nil {
		return
	}
	for {
		select {
		case cmd := <-gs.State.Commands:
			gs.executeCommand(cmd)
		default:
			return
		}
	}
}

// AIBuildOnPlanet implements tickable.BuildingAdder — lets AI build infrastructure.
func (gs *GameServer) AIBuildOnPlanet(planet *entities.Planet, buildingType string, owner string, systemID int) {
	game.AddBuildingToPlanet(planet, buildingType, owner, systemID)
}

// --- api.GameStateProvider implementation ---

func (gs *GameServer) GetSystems() []*entities.System     { return gs.State.Systems }
func (gs *GameServer) GetHyperlanes() []entities.Hyperlane { return gs.State.Hyperlanes }
func (gs *GameServer) GetPlayers() []*entities.Player      { return gs.State.Players }
func (gs *GameServer) GetHumanPlayer() *entities.Player    { return gs.State.HumanPlayer }
func (gs *GameServer) GetSeed() int64                      { return gs.State.Seed }
func (gs *GameServer) GetMarket() *economy.Market          { return gs.State.Market }
func (gs *GameServer) GetTradeExecutor() *economy.TradeExecutor {
	return gs.State.TradeExec
}
func (gs *GameServer) GetCargoCommander() *game.CargoCommandExecutor {
	return gs.CargoCommander
}
func (gs *GameServer) GetFleetManagementSystem() *game.FleetManagementSystem {
	return gs.FleetMgmtSystem
}
func (gs *GameServer) GetEventLog() *game.EventLog {
	return gs.Events
}
func (gs *GameServer) GetCommandChannel() chan game.GameCommand {
	return gs.State.Commands
}
func (gs *GameServer) GetTickInfo() (tick int64, gameTime string, speed string, paused bool) {
	if gs.TickManager == nil {
		return 0, "0:00", "1x", false
	}
	return gs.TickManager.GetCurrentTick(),
		gs.TickManager.GetGameTimeFormatted(),
		gs.TickManager.GetSpeedString(),
		gs.TickManager.IsPaused()
}

// --- Tickable system provider interfaces ---

// GetMarketEngine implements tickable.MarketProvider
func (gs *GameServer) GetMarketEngine() *economy.Market { return gs.State.Market }

// LoadCargo implements tickable.CargoOperator
func (gs *GameServer) LoadCargo(ship *entities.Ship, planet *entities.Planet, resource string, qty int) (int, error) {
	if gs.CargoCommander == nil {
		return 0, fmt.Errorf("cargo system not initialized")
	}
	return gs.CargoCommander.LoadCargo(ship, planet, resource, qty)
}

// UnloadCargo implements tickable.CargoOperator
func (gs *GameServer) UnloadCargo(ship *entities.Ship, planet *entities.Planet, resource string, qty int) (int, error) {
	if gs.CargoCommander == nil {
		return 0, fmt.Errorf("cargo system not initialized")
	}
	return gs.CargoCommander.UnloadCargo(ship, planet, resource, qty)
}

// --- Fleet commands ---

func (gs *GameServer) MoveFleetToSystem(fleet *entities.Fleet, targetSystemID int) (int, int) {
	return gs.FleetCmdExecutor.MoveFleetToSystem(fleet, targetSystemID)
}
func (gs *GameServer) MoveFleetToPlanet(fleet *entities.Fleet, targetPlanet *entities.Planet) (int, int) {
	return gs.FleetCmdExecutor.MoveFleetToPlanet(fleet, targetPlanet)
}
func (gs *GameServer) MoveFleetToStar(fleet *entities.Fleet) (int, int) {
	return gs.FleetCmdExecutor.MoveFleetToStar(fleet)
}
func (gs *GameServer) GetConnectedSystems(fromSystemID int) []int {
	return gs.FleetCmdExecutor.GetConnectedSystems(fromSystemID)
}
func (gs *GameServer) GetSystemByID(systemID int) *entities.System {
	return gs.FleetCmdExecutor.GetSystemByID(systemID)
}

// StartShipJourney moves a single ship to a target system (for AI logistics).
func (gs *GameServer) StartShipJourney(ship *entities.Ship, targetSystemID int) bool {
	helper := tickable.NewShipMovementHelper(gs.GetSystemsMap(), gs.State.Hyperlanes)
	return helper.StartJourney(ship, targetSystemID)
}

// GetSystemsMap returns systems indexed by ID.
func (gs *GameServer) GetSystemsMap() map[int]*entities.System {
	return gs.State.GetSystemsMap()
}

// --- serverSystemContext implements tickable.SystemContext ---

type serverSystemContext struct {
	server *GameServer
}

func (ssc *serverSystemContext) GetGame() interface{}    { return ssc.server }
func (ssc *serverSystemContext) GetPlayers() interface{} { return ssc.server.State.Players }
func (ssc *serverSystemContext) GetTick() int64          { return ssc.server.TickManager.GetCurrentTick() }
