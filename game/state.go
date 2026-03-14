package game

import (
	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
)

// State represents the core game state (pure data, no behavior)
type State struct {
	Systems     []*entities.System
	Hyperlanes  []entities.Hyperlane
	Players     []*entities.Player
	HumanPlayer *entities.Player
	Seed        int64
	Market        *economy.Market
	TradeExec     *economy.TradeExecutor
	Commands      chan GameCommand
}

// GameCommand represents a command to be executed on the main goroutine.
type GameCommand struct {
	Type   string
	Data   interface{}
	Result chan interface{} // optional: for synchronous API responses
}

// TradeCommandData is the shared trade command payload used by both API and UI.
type TradeCommandData struct {
	Resource string
	Quantity int
	Buy      bool
	PlanetID int // optional: specific planet for the trade (0 = auto-select)
}

// CargoCommandData is the payload for cargo load/unload commands.
type CargoCommandData struct {
	ShipID   int
	PlanetID int
	Resource string
	Quantity int
	Load     bool // true = load onto ship, false = unload from ship
}

// BuildCommandData is the payload for starting construction.
type BuildCommandData struct {
	PlanetID     int    // planet to build on
	BuildingType string // "Mine", "Trading Post", "Refinery", "Habitat", "Shipyard"
	ResourceID   int    // for mines: which resource node to attach to (0 = auto)
}

// ShipBuildCommandData is the payload for building a ship.
type ShipBuildCommandData struct {
	PlanetID int    // planet with shipyard
	ShipType string // "Scout", "Cargo", "Colony", "Frigate", "Destroyer", "Cruiser"
}

// ShipMoveCommandData is the payload for moving a ship.
type ShipMoveCommandData struct {
	ShipID         int // ship to move
	TargetSystemID int // system to jump to
}

// UpgradeCommandData is the payload for upgrading a building.
type UpgradeCommandData struct {
	PlanetID      int // planet the building is on
	BuildingIndex int // index in planet.Buildings array
}

// ColonizeCommandData is the payload for colonizing a planet.
type ColonizeCommandData struct {
	ShipID   int // colony ship to use
	PlanetID int // unclaimed planet to colonize
}

// ShipRefuelCommandData is the payload for refueling a ship.
type ShipRefuelCommandData struct {
	ShipID   int // ship to refuel
	PlanetID int // planet to take Fuel from
	Amount   int // amount of Fuel to transfer (0 = fill up)
}

// FleetCreateCommandData is the payload for creating a fleet from a ship.
type FleetCreateCommandData struct {
	ShipID int // ship to promote to fleet
}

// FleetDisbandCommandData is the payload for disbanding a fleet.
type FleetDisbandCommandData struct {
	FleetID int // fleet to disband
}

// FleetAddShipCommandData is the payload for adding a ship to a fleet.
type FleetAddShipCommandData struct {
	ShipID  int // ship to add
	FleetID int // fleet to add to
}

// FleetRemoveShipCommandData is the payload for removing a ship from a fleet.
type FleetRemoveShipCommandData struct {
	ShipID  int // ship to remove
	FleetID int // fleet to remove from
}

// NewState creates a new empty game state
func NewState() *State {
	return &State{
		Systems:    make([]*entities.System, 0),
		Hyperlanes: make([]entities.Hyperlane, 0),
		Players:    make([]*entities.Player, 0),
		Commands:   make(chan GameCommand, 64),
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
	gs.Market = nil
	gs.TradeExec = nil
	gs.Commands = make(chan GameCommand, 64)
}
