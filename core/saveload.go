package core

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/views"
)

const (
	saveDirectory = "saves"
	saveExtension = ".xsave"
)

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

// SaveFileInfo contains metadata about a save file
type SaveFileInfo struct {
	Filename   string
	Path       string
	PlayerName string
	GameTime   string
	SavedAt    time.Time
	ModTime    time.Time
}

// SaveGameToFile saves the current game state to a file using gob
func (a *App) SaveGameToFile(playerName string) error {
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
		GameTime:           a.tickManager.GetGameTimeFormatted(),
		Tick:               a.tickManager.GetCurrentTick(),
		Seed:               a.state.Seed,
		TickSpeed:          a.tickManager.GetSpeed().(systems.TickSpeed),
		Systems:            a.state.Systems,
		Hyperlanes:         a.state.Hyperlanes,
		Players:            a.state.Players,
		ConstructionQueues: constructionQueues,
	}

	// Encode entire game state
	if err := encoder.Encode(saveData); err != nil {
		return fmt.Errorf("failed to encode save data: %w", err)
	}

	fmt.Printf("[SaveSystem] Game saved to: %s\n", filename)
	fmt.Printf("[SaveSystem] Saved %d systems, %d players, tick %d\n",
		len(a.state.Systems), len(a.state.Players), a.tickManager.GetCurrentTick())
	return nil
}

// LoadGameFromPath loads a game from the given path
func (a *App) LoadGameFromPath(path string) error {
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
	a.state.Systems = saveData.Systems
	a.state.Hyperlanes = saveData.Hyperlanes
	a.state.Seed = saveData.Seed
	a.state.Players = saveData.Players

	// Update tick manager with saved state
	a.tickManager.SetSpeed(saveData.TickSpeed)
	a.tickManager.SetCurrentTick(saveData.Tick)

	// Find human player
	a.state.HumanPlayer = nil
	for _, player := range a.state.Players {
		if player.Type == entities.PlayerTypeHuman {
			a.state.HumanPlayer = player
			break
		}
	}

	// Rebuild planet owner references
	for _, player := range a.state.Players {
		for _, planet := range player.OwnedPlanets {
			planet.Owner = player.Name
		}
	}

	// Initialize tickable systems
	context := &GameSystemContext{app: a}
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

	a.registerConstructionHandler()

	// Update fleet command executor with loaded systems/hyperlanes
	a.fleetCmdExecutor = game.NewFleetCommandExecutor(a.state.Systems, a.state.Hyperlanes)

	// Initialize game-specific components
	a.initializeGameComponents()

	fmt.Printf("[SaveSystem] Game loaded successfully\n")
	return nil
}

// ListSaveFiles returns all save files (implements SaveLoadInterface)
func (a *App) ListSaveFiles() ([]views.SaveFileInfo, error) {
	files, err := listSaveFiles()
	if err != nil {
		return nil, err
	}

	// Convert from app.SaveFileInfo to views.SaveFileInfo
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

// GetSaveFileInfo gets info about a specific save file (implements SaveLoadInterface)
func (a *App) GetSaveFileInfo(path string) (views.SaveFileInfo, error) {
	info, err := getSaveFileInfo(path)
	if err != nil {
		return views.SaveFileInfo{}, err
	}

	// Convert from app.SaveFileInfo to views.SaveFileInfo
	return views.SaveFileInfo{
		Filename:   info.Filename,
		Path:       info.Path,
		PlayerName: info.PlayerName,
		GameTime:   info.GameTime,
		SavedAt:    info.SavedAt,
		ModTime:    info.ModTime,
	}, nil
}

// DeleteSaveFile deletes a save file by path (implements SaveLoadInterface)
func (a *App) DeleteSaveFile(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete save file: %w", err)
	}
	fmt.Printf("[SaveSystem] Save file deleted: %s\n", path)
	return nil
}

// RenameSaveFile renames a save file (implements SaveLoadInterface)
func (a *App) RenameSaveFile(oldPath string, newFilename string) error {
	// Validate new filename doesn't already exist
	newPath := filepath.Join(saveDirectory, newFilename)
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("save file already exists: %s", newFilename)
	}

	// Rename the file
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename save file: %w", err)
	}
	fmt.Printf("[SaveSystem] Save file renamed from %s to %s\n", oldPath, newPath)
	return nil
}

// listSaveFiles returns a list of all save files (internal helper)
func listSaveFiles() ([]SaveFileInfo, error) {
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
			info, err := getSaveFileInfo(fullPath)
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

// getSaveFileInfo reads metadata from a save file without loading the entire game (internal helper)
func getSaveFileInfo(filepath string) (SaveFileInfo, error) {
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
