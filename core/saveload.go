package core

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/views"
)

const (
	saveDirectory = "saves"
	saveExtension = ".xsave"
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

// SaveFileInfo contains metadata about a save file (internal type)
type SaveFileInfo struct {
	Filename   string
	Path       string
	PlayerName string
	GameTime   string
	SavedAt    time.Time
	ModTime    time.Time
}

// SaveGameToFile delegates to the server's save function.
func (a *App) SaveGameToFile(playerName string) error {
	return a.Server.SaveGame(playerName)
}

// ListSaveFiles returns all save files (implements SaveLoadInterface)
func (a *App) ListSaveFiles() ([]views.SaveFileInfo, error) {
	files, err := listSaveFiles()
	if err != nil {
		return nil, err
	}

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
	return os.Remove(path)
}

// RenameSaveFile renames a save file (implements SaveLoadInterface)
func (a *App) RenameSaveFile(oldPath string, newFilename string) error {
	newPath := filepath.Join(saveDirectory, newFilename)
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("save file already exists: %s", newFilename)
	}
	return os.Rename(oldPath, newPath)
}

// listSaveFiles returns a list of all save files (internal helper)
func listSaveFiles() ([]SaveFileInfo, error) {
	if err := os.MkdirAll(saveDirectory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create save directory: %w", err)
	}

	entries, err := os.ReadDir(saveDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to read save directory: %w", err)
	}

	saveFiles := make([]SaveFileInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != saveExtension {
			continue
		}
		fullPath := filepath.Join(saveDirectory, entry.Name())
		info, err := getSaveFileInfo(fullPath)
		if err != nil {
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
	return saveFiles, nil
}

// getSaveFileInfo reads metadata from a save file
func getSaveFileInfo(fpath string) (SaveFileInfo, error) {
	file, err := os.Open(fpath)
	if err != nil {
		return SaveFileInfo{}, err
	}
	defer file.Close()

	var saveData struct {
		Version    string
		SavedAt    time.Time
		PlayerName string
		GameTime   string
		Tick       int64
		Seed       int64
		TickSpeed  systems.TickSpeed
		Systems    []*entities.System
		Hyperlanes []entities.Hyperlane
		Players    []*entities.Player
	}

	if err := gob.NewDecoder(file).Decode(&saveData); err != nil {
		return SaveFileInfo{}, err
	}

	fileInfo, err := os.Stat(fpath)
	if err != nil {
		return SaveFileInfo{}, err
	}

	return SaveFileInfo{
		Filename:   fpath,
		Path:       fpath,
		PlayerName: saveData.PlayerName,
		GameTime:   saveData.GameTime,
		SavedAt:    saveData.SavedAt,
		ModTime:    fileInfo.ModTime(),
	}, nil
}
