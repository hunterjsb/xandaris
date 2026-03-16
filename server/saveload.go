package server

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hunterjsb/xandaris/api"
	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/tickable"
)

const (
	saveDirectory = "saves"
	saveExtension = ".xsave"
	// SaveVersion — bump this when save format is incompatible.
	// The autosave loader will discard saves with a different version.
	// 3.0.0: logistics-driven trade (deliveries, credit ledger, shipping routes, docking)
	SaveVersion = "3.0.0"
)

func init() {
	gob.Register(&entities.Star{})
	gob.Register(&entities.Planet{})
	gob.Register(&entities.Resource{})
	gob.Register(&entities.Building{})
	gob.Register(&entities.Station{})
	gob.Register(&entities.Ship{})
	gob.Register(&entities.System{})
	gob.Register(&entities.Player{})
	gob.Register(&entities.Hyperlane{})
	gob.Register(&entities.ResourceStorage{})
	gob.Register(&tickable.ConstructionItem{})
	gob.Register(&economy.ResourceMarket{})
	gob.Register(map[string]*economy.ResourceMarket{})
	gob.Register(&economy.PendingDelivery{})
	gob.Register(&game.ShippingRoute{})
}

// SaveGame saves the current game state.
func (gs *GameServer) SaveGame(playerName string) error {
	if err := os.MkdirAll(saveDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := filepath.Join(saveDirectory, fmt.Sprintf("%s_%s%s", playerName, timestamp, saveExtension))

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create save file: %w", err)
	}
	defer file.Close()

	var constructionQueues map[string][]*tickable.ConstructionItem
	if cs := tickable.GetConstructionSystem(); cs != nil {
		constructionQueues = cs.GetAllQueues()
	}

	saveData := struct {
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
		MarketSnapshot     *economy.MarketSnapshot
		StandingOrders     []*game.StandingOrder
		Deliveries         []*economy.PendingDelivery
		ShippingRoutes     []*game.ShippingRoute
		CreditOutstanding  map[string]map[string]int
		CreditLimits       map[string]map[string]int
	}{
		Version:            SaveVersion,
		SavedAt:            time.Now(),
		PlayerName:         playerName,
		GameTime:           gs.TickManager.GetGameTimeFormatted(),
		Tick:               gs.TickManager.GetCurrentTick(),
		Seed:               gs.State.Seed,
		TickSpeed:          gs.TickManager.GetSpeed().(systems.TickSpeed),
		Systems:            gs.State.Systems,
		Hyperlanes:         gs.State.Hyperlanes,
		Players:            gs.State.Players,
		ConstructionQueues: constructionQueues,
		MarketSnapshot:     gs.getMarketSnapshot(),
		StandingOrders:     gs.State.StandingOrders,
		Deliveries:         gs.getDeliveries(),
		ShippingRoutes:     gs.getShippingRoutes(),
		CreditOutstanding:  gs.getCreditOutstanding(),
		CreditLimits:       gs.getCreditLimits(),
	}

	if err := gob.NewEncoder(file).Encode(saveData); err != nil {
		return fmt.Errorf("failed to encode save data: %w", err)
	}

	fmt.Printf("[Server] Game saved to: %s\n", filename)
	return nil
}

// AutoSave saves to a fixed path, overwriting the previous autosave.
func (gs *GameServer) AutoSave(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	// Write to temp file first, then rename (atomic)
	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	var constructionQueues map[string][]*tickable.ConstructionItem
	if cs := tickable.GetConstructionSystem(); cs != nil {
		constructionQueues = cs.GetAllQueues()
	}

	playerName := "Server"
	if gs.State.HumanPlayer != nil {
		playerName = gs.State.HumanPlayer.Name
	}

	saveData := struct {
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
		MarketSnapshot     *economy.MarketSnapshot
		StandingOrders     []*game.StandingOrder
		Deliveries         []*economy.PendingDelivery
		ShippingRoutes     []*game.ShippingRoute
		CreditOutstanding  map[string]map[string]int
		CreditLimits       map[string]map[string]int
	}{
		Version:            SaveVersion,
		SavedAt:            time.Now(),
		PlayerName:         playerName,
		GameTime:           gs.TickManager.GetGameTimeFormatted(),
		Tick:               gs.TickManager.GetCurrentTick(),
		Seed:               gs.State.Seed,
		TickSpeed:          gs.TickManager.GetSpeed().(systems.TickSpeed),
		Systems:            gs.State.Systems,
		Hyperlanes:         gs.State.Hyperlanes,
		Players:            gs.State.Players,
		ConstructionQueues: constructionQueues,
		MarketSnapshot:     gs.getMarketSnapshot(),
		StandingOrders:     gs.State.StandingOrders,
		Deliveries:         gs.getDeliveries(),
		ShippingRoutes:     gs.getShippingRoutes(),
		CreditOutstanding:  gs.getCreditOutstanding(),
		CreditLimits:       gs.getCreditLimits(),
	}

	if err := gob.NewEncoder(file).Encode(saveData); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to encode: %w", err)
	}
	file.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}

	fmt.Printf("[Autosave] Saved tick %d to %s\n", gs.TickManager.GetCurrentTick(), path)
	return nil
}

func (gs *GameServer) getMarketSnapshot() *economy.MarketSnapshot {
	if gs.State.Market == nil {
		return nil
	}
	snap := gs.State.Market.GetSnapshot()
	return &snap
}

func (gs *GameServer) getDeliveries() []*economy.PendingDelivery {
	if gs.DeliveryMgr == nil {
		return nil
	}
	return gs.DeliveryMgr.GetAllDeliveries()
}

func (gs *GameServer) getShippingRoutes() []*game.ShippingRoute {
	if gs.ShippingMgr == nil {
		return nil
	}
	return gs.ShippingMgr.GetAllRoutes()
}

func (gs *GameServer) getCreditOutstanding() map[string]map[string]int {
	if gs.CreditLedger == nil {
		return nil
	}
	return gs.CreditLedger.GetAllOutstanding()
}

func (gs *GameServer) getCreditLimits() map[string]map[string]int {
	if gs.CreditLedger == nil {
		return nil
	}
	return gs.CreditLedger.GetAllLimits()
}

// LoadGame loads a game from the given path.
func (gs *GameServer) LoadGame(path string) error {
	fmt.Printf("[Server] Loading game from: %s\n", path)

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open save file: %w", err)
	}
	defer file.Close()

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
		MarketSnapshot     *economy.MarketSnapshot
		StandingOrders     []*game.StandingOrder
		Deliveries         []*economy.PendingDelivery
		ShippingRoutes     []*game.ShippingRoute
		CreditOutstanding  map[string]map[string]int
		CreditLimits       map[string]map[string]int
	}

	if err := gob.NewDecoder(file).Decode(&saveData); err != nil {
		return fmt.Errorf("failed to decode save data: %w", err)
	}

	// Version check — accept compatible save versions
	compatibleVersions := map[string]bool{
		SaveVersion: true,
		"2.4.0":     true, // pre-logistics saves (missing delivery/shipping/credit fields — zeroed by gob)
	}
	if !compatibleVersions[saveData.Version] {
		return fmt.Errorf("save version mismatch: got %q, need %q", saveData.Version, SaveVersion)
	}

	// Restore state
	gs.State.Systems = saveData.Systems
	gs.State.Hyperlanes = saveData.Hyperlanes
	gs.State.Seed = saveData.Seed
	gs.State.Players = saveData.Players

	gs.State.Market = economy.RestoreMarket(saveData.MarketSnapshot)
	gs.State.TradeExec = economy.NewTradeExecutor(gs.State.Market)
	gs.State.StandingOrders = saveData.StandingOrders

	gs.TickManager.SetSpeed(saveData.TickSpeed)
	gs.TickManager.SetCurrentTick(saveData.Tick)

	// Find human player
	gs.State.HumanPlayer = nil
	for _, player := range gs.State.Players {
		if player.Type == entities.PlayerTypeHuman {
			gs.State.HumanPlayer = player
			break
		}
	}

	// Rebuild player references (HomePlanet, HomeSystem, planet owners)
	for _, player := range gs.State.Players {
		for _, planet := range player.OwnedPlanets {
			planet.Owner = player.Name
			// Restore HomePlanet to the first owned planet (best heuristic after load)
			if player.HomePlanet == nil {
				player.HomePlanet = planet
			}
		}
		// Restore HomeSystem by finding the system containing HomePlanet
		if player.HomePlanet != nil {
			for _, sys := range gs.State.Systems {
				for _, e := range sys.Entities {
					if p, ok := e.(*entities.Planet); ok && p.GetID() == player.HomePlanet.GetID() {
						player.HomeSystem = sys
						break
					}
				}
				if player.HomeSystem != nil {
					break
				}
			}
		}
	}

	// Initialize simulation
	gs.initSimulation()

	// Restore construction queues
	if saveData.ConstructionQueues != nil {
		if cs := tickable.GetConstructionSystem(); cs != nil {
			cs.RestoreQueues(saveData.ConstructionQueues)
		}
	}

	// Restore deliveries
	if saveData.Deliveries != nil && gs.DeliveryMgr != nil {
		gs.DeliveryMgr.RestoreDeliveries(saveData.Deliveries)
	}

	// Restore shipping routes
	if saveData.ShippingRoutes != nil && gs.ShippingMgr != nil {
		for _, route := range saveData.ShippingRoutes {
			gs.ShippingMgr.CreateRoute(route.Owner, route.SourcePlanet, route.DestPlanet, route.Resource, route.Quantity, route.ShipID)
		}
	}

	// Restore credit ledger
	if gs.CreditLedger != nil {
		gs.CreditLedger.RestoreLedger(saveData.CreditOutstanding, saveData.CreditLimits)
	}

	// Start API
	api.StartServer(gs)

	// Reconcile registered accounts that don't have in-game players
	gs.reconcileRegisteredPlayers()

	fmt.Printf("[Server] Game loaded: %s, tick %d\n", saveData.PlayerName, saveData.Tick)
	return nil
}
