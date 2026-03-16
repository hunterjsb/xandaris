package server

import (
	"fmt"
	"os"
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
	Chat             *game.ChatLog
	Registry         *game.PlayerRegistry
	DeliveryMgr      *economy.DeliveryManager
	ShippingMgr      *game.ShippingManager
	cmdRegistry      *CommandRegistry
	// Remote is set when connected to a remote server (desktop only, not WASM)
	remoteSync interface{}

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
		gs.State.HumanPlayer.HomePlanet.AddStoredResource(entities.ResFuel, 200)
		gs.State.HumanPlayer.HomePlanet.AddStoredResource(entities.ResOil, 150)
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

	// Reconcile: create Player objects for registered accounts that don't have one
	gs.reconcileRegisteredPlayers()

	fmt.Printf("[Server] New game started for %s (%d systems, %d players)\n",
		playerName, len(gs.State.Systems), len(gs.State.Players))

	return nil
}

// reconcileRegisteredPlayers creates in-game Player objects for any registered
// accounts (from accounts.json) that don't already have a matching player.
func (gs *GameServer) reconcileRegisteredPlayers() {
	if gs.Registry == nil {
		return
	}

	accounts := gs.Registry.GetAllAccounts()
	for _, acc := range accounts {
		// Check if player already exists in game
		exists := false
		for _, p := range gs.State.Players {
			if p != nil && p.Name == acc.Name {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		// Create a new player for this account
		playerID := len(gs.State.Players)
		colors := utils.GetAIPlayerColors()
		playerColor := colors[playerID%len(colors)]
		newPlayer := entities.NewPlayer(playerID, acc.Name, playerColor, entities.PlayerTypeHuman)

		entities.InitializePlayer(newPlayer, gs.State.Systems)
		if newPlayer.HomePlanet == nil {
			fmt.Printf("[Server] WARNING: Could not find homeworld for %s\n", acc.Name)
			continue
		}

		game.PrepareHomeworld(newPlayer, true) // auto-mines + refinery + generator
		if newPlayer.HomePlanet != nil {
			newPlayer.HomePlanet.AddStoredResource(entities.ResFuel, 200)
			newPlayer.HomePlanet.AddStoredResource(entities.ResOil, 150)
		}

		gs.State.Players = append(gs.State.Players, newPlayer)
		acc.PlayerID = playerID
		gs.Registry.Save()

		fmt.Printf("[Server] Reconciled player: %s (id=%d, planet=%s)\n",
			acc.Name, playerID, newPlayer.HomePlanet.Name)
	}
}

// initSimulation sets up fleet/cargo commanders, tickable systems, and construction handler.
func (gs *GameServer) initSimulation() {
	gs.initCommandRegistry()
	gs.Events = game.NewEventLog(100)
	gs.Chat = game.NewChatLog(50)
	if gs.Registry == nil {
		gs.Registry = game.NewPlayerRegistry(os.Getenv("XANDARIS_API_KEY"))
	}

	// Wire trade event logging
	if gs.State.TradeExec != nil {
		gs.State.TradeExec.OnTrade = func(r economy.TradeRecord) {
			action := "bought"
			if r.Action == "sell" {
				action = "sold"
			}
			gs.Events.Addf(r.Tick, gs.TickManager.GetGameTimeFormatted(), game.EventTrade, r.Player,
				"%s %s %d %s @ %.0fcr", r.Player, action, r.Quantity, r.Resource, r.UnitPrice)
		}
	}

	gs.FleetCmdExecutor = game.NewFleetCommandExecutor(gs.State.Systems, gs.State.Hyperlanes)
	gs.FleetMgmtSystem = game.NewFleetManagementSystem(gs.State)
	gs.CargoCommander = game.NewCargoCommandExecutor(gs.State.Systems)

	// Wire delivery system for cargo-based trade
	gs.DeliveryMgr = economy.NewDeliveryManager()
	gs.ShippingMgr = game.NewShippingManager()

	if gs.State.TradeExec != nil {
		gs.State.TradeExec.SetSystems(gs.State.Systems)
		gs.State.TradeExec.Deliveries = gs.DeliveryMgr
		gs.State.TradeExec.Dispatcher = gs
	}

	// Initialize tickable systems
	ctx := &serverSystemContext{server: gs}
	tickable.InitializeAllSystems(ctx)

	// Register construction handler
	handler := game.NewConstructionHandler(gs.State.Systems, gs.State.Players, gs.TickManager)
	if cs := tickable.GetConstructionSystem(); cs != nil {
		cs.RegisterCompletionHandler(handler.HandleConstructionComplete)
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

// Interface implementations (GameProvider, GameStateProvider, ShipDispatcher)
// are in providers.go to keep server.go focused on lifecycle.

// NewGameWithSeed initializes a game using a specific seed (for remote sync).
func (gs *GameServer) NewGameWithSeed(playerName string, seed int64) error {
	gs.State.Reset()
	gs.State.Seed = seed

	gs.TickManager.Reset()

	galaxyGen := game.NewGalaxyGenerator(gs.screenWidth, gs.screenHeight)
	gs.State.Systems = galaxyGen.GenerateSystems(gs.State.Seed)
	gs.State.Hyperlanes = galaxyGen.GenerateHyperlanes(gs.State.Systems)

	// Create local player — home planet will be set by remote sync
	// (don't call InitializePlayer which picks a random planet)
	playerColor := utils.PlayerGreen
	gs.State.HumanPlayer = entities.NewPlayer(0, playerName, playerColor, entities.PlayerTypeHuman)
	gs.State.Players = append(gs.State.Players, gs.State.HumanPlayer)

	gs.State.Market = economy.NewMarket()
	gs.State.TradeExec = economy.NewTradeExecutor(gs.State.Market)

	// Don't seed AI factions — they exist on the remote server
	gs.initSimulation()

	fmt.Printf("[Server] Game initialized with remote seed %d (%d systems)\n",
		seed, len(gs.State.Systems))

	return nil
}

// SetRemoteSync sets the remote sync client (for --connect mode).
func (gs *GameServer) SetRemoteSync(rs interface{}) {
	gs.remoteSync = rs
}

// IsRemote returns true if this server is connected to a remote server.
func (gs *GameServer) IsRemote() bool {
	return gs.remoteSync != nil
}
