package game

import (
	"github.com/hunterjsb/xandaris/entities"
)

// State represents the core game state (pure data, no behavior)
type State struct {
	Systems     []*entities.System
	Hyperlanes  []entities.Hyperlane
	Players     []*entities.Player
	HumanPlayer *entities.Player
	Seed        int64
}

// NewState creates a new empty game state
func NewState() *State {
	return &State{
		Systems:    make([]*entities.System, 0),
		Hyperlanes: make([]entities.Hyperlane, 0),
		Players:    make([]*entities.Player, 0),
	}
}

// GetSystems returns all star systems
func (gs *State) GetSystems() []*entities.System {
	return gs.Systems
}

// GetSystemsMap returns systems indexed by ID for efficient lookup
func (gs *State) GetSystemsMap() map[int]*entities.System {
	systemsMap := make(map[int]*entities.System)
	for _, system := range gs.Systems {
		systemsMap[system.ID] = system
	}
	return systemsMap
}

// GetHyperlanes returns all hyperlane connections
func (gs *State) GetHyperlanes() []entities.Hyperlane {
	return gs.Hyperlanes
}

// GetPlayers returns all players
func (gs *State) GetPlayers() []*entities.Player {
	return gs.Players
}

// GetHumanPlayer returns the human player
func (gs *State) GetHumanPlayer() *entities.Player {
	return gs.HumanPlayer
}

// GetSeed returns the galaxy generation seed
func (gs *State) GetSeed() int64 {
	return gs.Seed
}

// Reset clears all game state
func (gs *State) Reset() {
	gs.Systems = make([]*entities.System, 0)
	gs.Hyperlanes = make([]entities.Hyperlane, 0)
	gs.Players = make([]*entities.Player, 0)
	gs.HumanPlayer = nil
	gs.Seed = 0
}
