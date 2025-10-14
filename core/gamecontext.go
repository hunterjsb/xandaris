package core

import (
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/views"
)

// GameContext interface implementation for views

// GetSystems returns the game's systems (implements GameContext)
func (a *App) GetSystems() []*entities.System {
	return a.state.GetSystems()
}

// GetHyperlanes returns all hyperlanes
func (a *App) GetHyperlanes() []entities.Hyperlane {
	return a.state.GetHyperlanes()
}

// GetPlayers returns the game's players
func (a *App) GetPlayers() []*entities.Player {
	return a.state.GetPlayers()
}

// GetHumanPlayer returns the human player
func (a *App) GetHumanPlayer() *entities.Player {
	return a.state.GetHumanPlayer()
}

// GetSeed returns the game seed
func (a *App) GetSeed() int64 {
	return a.state.GetSeed()
}

// GetSaveLoad returns the save/load interface
func (a *App) GetSaveLoad() views.SaveLoadInterface {
	return a // App itself will implement SaveLoadInterface
}

// GetSystemsMap returns a map of systems indexed by ID (internal use)
func (a *App) GetSystemsMap() map[int]*entities.System {
	return a.state.GetSystemsMap()
}
