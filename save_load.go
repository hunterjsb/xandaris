package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/views"
)

const (
	saveDirectory = "saves"
	saveExtension = ".xsave"
)

// SaveFileMetadata contains basic info about a save file
type SaveFileMetadata struct {
	Version    string    `json:"version"`
	SavedAt    time.Time `json:"saved_at"`
	PlayerName string    `json:"player_name"`
	GameTime   string    `json:"game_time"`
	Tick       int64     `json:"tick"`
}

// init registers all types that need to be serialized with gob
func init() {
	// Register all entity types so gob can handle the Entity interface
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
}

// SaveGameToFile saves the current game state to a file using gob
func (g *Game) SaveGameToFile(playerName string) error {
	// Create saves directory if it doesn't exist
	if err := os.MkdirAll(saveDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	// Generate filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := filepath.Join(saveDirectory, fmt.Sprintf("%s_%s%s", playerName, timestamp, saveExtension))

	// Create file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create save file: %w", err)
	}
	defer file.Close()

	// Create gob encoder
	encoder := gob.NewEncoder(file)

	// Get construction queues
	var constructionQueues map[string][]*tickable.ConstructionItem
	if constructionSystem := tickable.GetSystemByName("Construction"); constructionSystem != nil {
		if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
			constructionQueues = cs.GetAllQueues()
		}
	}

	// Create save data structure
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
	}{
		Version:            "2.0.0-gob",
		SavedAt:            time.Now(),
		PlayerName:         playerName,
		GameTime:           g.tickManager.GetGameTimeFormatted(),
		Tick:               g.tickManager.GetCurrentTick(),
		Seed:               g.seed,
		TickSpeed:          g.tickManager.GetSpeed().(systems.TickSpeed),
		Systems:            g.systems,
		Hyperlanes:         g.hyperlanes,
		Players:            g.players,
		ConstructionQueues: constructionQueues,
	}

	// Encode entire game state
	if err := encoder.Encode(saveData); err != nil {
		return fmt.Errorf("failed to encode save data: %w", err)
	}

	fmt.Printf("[SaveSystem] Game saved to: %s\n", filename)
	fmt.Printf("[SaveSystem] Saved %d systems, %d players, tick %d\n", len(g.systems), len(g.players), g.tickManager.GetCurrentTick())
	return nil
}

// LoadGameFromFile loads a game state from a file using gob
func LoadGameFromFile(filename string) (*Game, error) {
	fmt.Printf("[SaveSystem] Loading game from: %s\n", filename)

	// Open file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open save file: %w", err)
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
		return nil, fmt.Errorf("failed to decode save data: %w", err)
	}

	fmt.Printf("[SaveSystem] Decoded save data: version=%s, player=%s, systems=%d\n",
		saveData.Version, saveData.PlayerName, len(saveData.Systems))

	// Create new game instance
	g := &Game{
		systems:    saveData.Systems,
		hyperlanes: saveData.Hyperlanes,
		seed:       saveData.Seed,
		players:    saveData.Players,
	}

	// Initialize key bindings
	g.keyBindings = systems.NewKeyBindings()
	// Try to load custom key bindings from config
	if err := g.keyBindings.LoadFromFile(systems.GetKeyBindingsConfigPath()); err != nil {
		// Silently use defaults if config doesn't exist
	}

	// Initialize tick manager with saved state
	g.tickManager = systems.NewTickManager(10.0)
	g.tickManager.SetSpeed(saveData.TickSpeed)
	g.tickManager.SetCurrentTick(saveData.Tick)

	// Find human player
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

	// Initialize fleet manager
	g.fleetManager = systems.NewFleetManager(g)

	// Initialize view system
	g.viewManager = views.NewViewManager()

	// Create UI components (stay in main package)
	buildMenu := NewBuildMenu(g)
	constructionQueue := NewConstructionQueueUI(g)
	resourceStorage := NewResourceStorageUI(g)
	shipyardUI := NewShipyardUI(g)
	fleetInfoUI := NewFleetInfoUI(g)

	// Create and register views
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

	// Start with galaxy view
	g.viewManager.SwitchTo(views.ViewTypeGalaxy)

	fmt.Printf("[SaveSystem] Game successfully loaded: %d systems, %d players, tick %d\n",
		len(g.systems), len(g.players), g.tickManager.GetCurrentTick())
	return g, nil
}

// ListSaveFiles returns a list of all save files
func ListSaveFiles() ([]SaveFileInfo, error) {
	// Create saves directory if it doesn't exist
	if err := os.MkdirAll(saveDirectory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create save directory: %w", err)
	}

	// Read directory
	entries, err := os.ReadDir(saveDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to read save directory: %w", err)
	}

	saveFiles := make([]SaveFileInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) == saveExtension {
			fullPath := filepath.Join(saveDirectory, entry.Name())

			// Try to read basic info from the save file
			info, err := GetSaveFileInfo(fullPath)
			if err != nil {
				// If we can't read it, just add basic info from file system
				fileInfo, _ := entry.Info()
				saveFiles = append(saveFiles, SaveFileInfo{
					Filename:   entry.Name(),
					Path:       fullPath,
					ModTime:    fileInfo.ModTime(),
					PlayerName: "Unknown",
					GameTime:   "Unknown",
					SavedAt:    fileInfo.ModTime(),
				})
				continue
			}
			saveFiles = append(saveFiles, info)
		}
	}

	return saveFiles, nil
}

// SaveFileInfo contains metadata about a save file
type SaveFileInfo struct {
	Filename   string
	Path       string
	PlayerName string
	GameTime   string
	SavedAt    time.Time
	ModTime    time.Time
}

// GetSaveFileInfo reads metadata from a save file without loading the entire game
func GetSaveFileInfo(filepath string) (SaveFileInfo, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return SaveFileInfo{}, err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)

	// Decode only the metadata we need
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
		return SaveFileInfo{}, err
	}

	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return SaveFileInfo{}, err
	}

	return SaveFileInfo{
		Filename:   filepath,
		Path:       filepath,
		PlayerName: saveData.PlayerName,
		GameTime:   saveData.GameTime,
		SavedAt:    saveData.SavedAt,
		ModTime:    fileInfo.ModTime(),
	}, nil
}
