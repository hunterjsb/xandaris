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
	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/tickable"
)

const (
	saveDirectory = "saves"
	saveExtension = ".xsave"
	// SaveVersion — bump this when save format is incompatible.
	// The autosave loader will discard saves with a different version.
	SaveVersion = "2.2.0"
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
	}

	if err := gob.NewDecoder(file).Decode(&saveData); err != nil {
		return fmt.Errorf("failed to decode save data: %w", err)
	}

	// Version check — reject incompatible saves
	if saveData.Version != SaveVersion {
		return fmt.Errorf("save version mismatch: got %q, need %q", saveData.Version, SaveVersion)
	}

	// Restore state
	gs.State.Systems = saveData.Systems
	gs.State.Hyperlanes = saveData.Hyperlanes
	gs.State.Seed = saveData.Seed
	gs.State.Players = saveData.Players

	gs.State.Market = economy.RestoreMarket(saveData.MarketSnapshot)
	gs.State.TradeExec = economy.NewTradeExecutor(gs.State.Market)

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

	// Rebuild planet owner references
	for _, player := range gs.State.Players {
		for _, planet := range player.OwnedPlanets {
			planet.Owner = player.Name
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

	// Start API
	api.StartServer(gs)

	// Reconcile registered accounts that don't have in-game players
	gs.reconcileRegisteredPlayers()

	fmt.Printf("[Server] Game loaded: %s, tick %d\n", saveData.PlayerName, saveData.Tick)
	return nil
}
