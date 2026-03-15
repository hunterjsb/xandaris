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
	Registry         *game.PlayerRegistry
	DeliveryMgr      *economy.DeliveryManager
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
func (gs *GameServer) GetRegistry() *game.PlayerRegistry {
	return gs.Registry
}
func (gs *GameServer) GetCommandChannel() chan game.GameCommand {
	return gs.State.Commands
}
func (gs *GameServer) GetStandingOrders(player string) []*game.StandingOrder {
	return gs.State.GetStandingOrders(player)
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

// NewGameWithSeed initializes a game using a specific seed (for remote sync).
func (gs *GameServer) NewGameWithSeed(playerName string, seed int64) error {
	gs.State.Reset()
	gs.State.Seed = seed

	gs.TickManager.Reset()

	galaxyGen := game.NewGalaxyGenerator(gs.screenWidth, gs.screenHeight)
	gs.State.Systems = galaxyGen.GenerateSystems(gs.State.Seed)
	gs.State.Hyperlanes = galaxyGen.GenerateHyperlanes(gs.State.Systems)

	playerColor := utils.PlayerGreen
	gs.State.HumanPlayer = entities.NewPlayer(0, playerName, playerColor, entities.PlayerTypeHuman)
	gs.State.Players = append(gs.State.Players, gs.State.HumanPlayer)

	entities.InitializePlayer(gs.State.HumanPlayer, gs.State.Systems)
	game.PrepareHomeworld(gs.State.HumanPlayer, false)

	if gs.State.HumanPlayer.HomePlanet != nil {
		gs.State.HumanPlayer.HomePlanet.AddStoredResource("Fuel", 200)
		gs.State.HumanPlayer.HomePlanet.AddStoredResource("Oil", 150)
	}

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

// LogEvent implements tickable.EventLogger for game event tracking.
func (gs *GameServer) LogEvent(eventType string, player string, message string) {
	if gs.Events != nil {
		gs.Events.Add(gs.TickManager.GetCurrentTick(), gs.TickManager.GetGameTimeFormatted(),
			game.EventType(eventType), player, message)
	}
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

// --- Standing order support ---

// GetStandingOrderInfos implements tickable.StandingOrderProvider.
func (gs *GameServer) GetStandingOrderInfos() []tickable.StandingOrderInfo {
	result := make([]tickable.StandingOrderInfo, 0, len(gs.State.StandingOrders))
	for _, o := range gs.State.StandingOrders {
		result = append(result, tickable.StandingOrderInfo{
			ID: o.ID, Player: o.Player, PlanetID: o.PlanetID,
			Resource: o.Resource, Action: o.Action, Quantity: o.Quantity,
			Threshold: o.Threshold, Active: o.Active,
		})
	}
	return result
}

// ExecuteStandingOrderTrade implements tickable.StandingOrderProvider.
func (gs *GameServer) ExecuteStandingOrderTrade(order tickable.StandingOrderInfo, player *entities.Player) error {
	if gs.State.TradeExec == nil || gs.State.Market == nil {
		return fmt.Errorf("market not available")
	}

	// Price check from original order (if set)
	for _, o := range gs.State.StandingOrders {
		if o.ID == order.ID {
			if o.Action == "buy" && o.MaxPrice > 0 {
				price := gs.State.Market.GetBuyPrice(order.Resource)
				if int(price) > o.MaxPrice {
					return fmt.Errorf("price too high")
				}
			}
			if o.Action == "sell" && o.MinPrice > 0 {
				price := gs.State.Market.GetSellPrice(order.Resource)
				if int(price) < o.MinPrice {
					return fmt.Errorf("price too low")
				}
			}
			break
		}
	}

	// Find the planet
	var planet *entities.Planet
	for _, p := range player.OwnedPlanets {
		if p != nil && p.GetID() == order.PlanetID {
			planet = p
			break
		}
	}
	if planet == nil {
		return fmt.Errorf("planet not found")
	}

	var err error
	if order.Action == "buy" {
		_, err = gs.State.TradeExec.Buy(player, gs.State.Players, order.Resource, order.Quantity, planet)
	} else {
		_, err = gs.State.TradeExec.Sell(player, gs.State.Players, order.Resource, order.Quantity, planet)
	}
	return err
}

// --- economy.ShipDispatcher implementation ---

// FindAvailableCargoShip finds an idle cargo ship owned by the player, preferring the given system.
func (gs *GameServer) FindAvailableCargoShip(owner string, systemID int) *entities.Ship {
	for _, p := range gs.State.Players {
		if p == nil || p.Name != owner {
			continue
		}
		// Prefer ships in the requested system
		for _, ship := range p.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeCargo {
				continue
			}
			if ship.Status == entities.ShipStatusMoving || ship.DeliveryID != 0 {
				continue
			}
			if ship.CurrentSystem == systemID {
				return ship
			}
		}
		// Fallback: any idle cargo ship
		for _, ship := range p.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeCargo {
				continue
			}
			if ship.Status == entities.ShipStatusMoving || ship.DeliveryID != 0 {
				continue
			}
			return ship
		}
	}
	return nil
}

// DispatchShipToSystem sends a ship to a target system via hyperlane.
func (gs *GameServer) DispatchShipToSystem(ship *entities.Ship, targetSystemID int) bool {
	return gs.StartShipJourney(ship, targetSystemID)
}

// AreSystemsConnected checks if there's a hyperlane path between two systems.
func (gs *GameServer) AreSystemsConnected(fromID, toID int) bool {
	helper := tickable.NewShipMovementHelper(gs.GetSystemsMap(), gs.State.Hyperlanes)
	return helper.AreSystemsConnected(fromID, toID)
}

// FindPath returns the multi-hop route between two systems.
func (gs *GameServer) FindPath(fromID, toID int) []int {
	helper := tickable.NewShipMovementHelper(gs.GetSystemsMap(), gs.State.Hyperlanes)
	return helper.FindPath(fromID, toID)
}

// GetDeliveryManager returns the delivery manager (for tickable systems).
func (gs *GameServer) GetDeliveryManager() *economy.DeliveryManager {
	return gs.DeliveryMgr
}

// --- serverSystemContext implements tickable.SystemContext ---

type serverSystemContext struct {
	server *GameServer
}

func (ssc *serverSystemContext) GetGame() interface{}    { return ssc.server }
func (ssc *serverSystemContext) GetPlayers() interface{} { return ssc.server.State.Players }
func (ssc *serverSystemContext) GetTick() int64          { return ssc.server.TickManager.GetCurrentTick() }
