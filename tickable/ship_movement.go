package tickable

import (
	"math"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ShipMovementSystem{
		BaseSystem: NewBaseSystem("ShipMovement", 20),
	})
}

// ShipMovementSystem handles ship travel between systems
type ShipMovementSystem struct {
	*BaseSystem
}

// OnTick processes ship movement each game tick
func (sms *ShipMovementSystem) OnTick(tick int64) {
	context := sms.GetContext()
	if context == nil {
		return
	}

	// Get game from context
	gameInterface := context.GetGame()
	if gameInterface == nil {
		return
	}

	// Cast to Game type (we'll need to handle this carefully)
	type GameWithSystems interface {
		GetSystems() map[int]*entities.System
		GetHyperlanes() []entities.Hyperlane
	}

	game, ok := gameInterface.(GameWithSystems)
	if !ok {
		return
	}

	// Process all ships across all systems
	for _, system := range game.GetSystems() {
		for _, entity := range system.Entities {
			if ship, ok := entity.(*entities.Ship); ok {
				sms.processShipMovement(ship, game)
			}
		}
	}
}

// processShipMovement handles movement for a single ship
func (sms *ShipMovementSystem) processShipMovement(ship *entities.Ship, game interface {
	GetSystems() map[int]*entities.System
	GetHyperlanes() []entities.Hyperlane
}) {
	// Only process ships that are moving
	if ship.Status != entities.ShipStatusMoving {
		return
	}

	// Check if ship has a valid target
	if ship.TargetSystem == -1 {
		ship.Status = entities.ShipStatusOrbiting
		return
	}

	// Get target system
	targetSystem := game.GetSystems()[ship.TargetSystem]
	if targetSystem == nil {
		// Invalid target, stop moving
		ship.Status = entities.ShipStatusOrbiting
		ship.TargetSystem = -1
		return
	}

	// Calculate travel speed (affected by ship's speed multiplier)
	baseSpeed := 0.01 // 1% per tick = 100 ticks to complete jump
	travelSpeed := baseSpeed * ship.Speed

	// Consume fuel while traveling
	if ship.CurrentFuel > 0 {
		ship.ConsumeFuel(int(math.Ceil(ship.FuelPerTick)))
	} else {
		// Out of fuel - ship is stranded in hyperspace
		ship.Status = entities.ShipStatusIdle
		// Ship stays at current progress but can't move
		return
	}

	// Update travel progress
	ship.TravelProgress += travelSpeed

	// Check if ship has arrived
	if ship.TravelProgress >= 1.0 {
		sms.arriveAtSystem(ship, targetSystem, game.GetSystems())
	}
}

// arriveAtSystem handles ship arrival at target system
func (sms *ShipMovementSystem) arriveAtSystem(ship *entities.Ship, targetSystem *entities.System, systems map[int]*entities.System) {
	// Remove ship from current system
	currentSystem := systems[ship.CurrentSystem]
	if currentSystem != nil {
		currentSystem.RemoveEntity(ship.ID)
	}

	// Add ship to target system
	targetSystem.AddEntity(ship)

	// Update ship state
	ship.CurrentSystem = ship.TargetSystem
	ship.TargetSystem = -1
	ship.TravelProgress = 0.0
	ship.Status = entities.ShipStatusOrbiting

	// Place ship in orbit around the star
	// Find the star to get a reference orbit distance
	starEntities := targetSystem.GetEntitiesByType(entities.EntityTypeStar)
	if len(starEntities) > 0 {
		// Place ship in a mid-range orbit
		ship.OrbitDistance = 150.0
		ship.OrbitAngle = 0.0
	}
}

// ShipMovementHelper provides helper functions for ship movement
// This can be accessed from other parts of the game
type ShipMovementHelper struct {
	systems    map[int]*entities.System
	hyperlanes []entities.Hyperlane
}

// NewShipMovementHelper creates a new helper
func NewShipMovementHelper(systems map[int]*entities.System, hyperlanes []entities.Hyperlane) *ShipMovementHelper {
	return &ShipMovementHelper{
		systems:    systems,
		hyperlanes: hyperlanes,
	}
}

// StartJourney initiates a ship's journey to a target system
func (smh *ShipMovementHelper) StartJourney(ship *entities.Ship, targetSystemID int) bool {
	// Verify ship can jump
	if !ship.CanJump() {
		return false
	}

	// Verify target system exists
	targetSystem := smh.systems[targetSystemID]
	if targetSystem == nil {
		return false
	}

	// Verify hyperlane connection exists
	currentSystem := smh.systems[ship.CurrentSystem]
	if currentSystem == nil || !smh.hasHyperlaneConnection(currentSystem.ID, targetSystemID) {
		return false
	}

	// Consume fuel for jump initiation
	ship.ConsumeFuel(ship.FuelPerJump)

	// Set ship to moving status
	ship.Status = entities.ShipStatusMoving
	ship.TargetSystem = targetSystemID
	ship.TravelProgress = 0.0

	return true
}

// hasHyperlaneConnection checks if two systems are connected
func (smh *ShipMovementHelper) hasHyperlaneConnection(fromID, toID int) bool {
	for _, hyperlane := range smh.hyperlanes {
		if (hyperlane.From == fromID && hyperlane.To == toID) ||
			(hyperlane.From == toID && hyperlane.To == fromID) {
			return true
		}
	}
	return false
}

// GetConnectedSystems returns all systems connected to the given system
func (smh *ShipMovementHelper) GetConnectedSystems(systemID int) []int {
	connected := make([]int, 0)
	for _, hyperlane := range smh.hyperlanes {
		if hyperlane.From == systemID {
			connected = append(connected, hyperlane.To)
		} else if hyperlane.To == systemID {
			connected = append(connected, hyperlane.From)
		}
	}
	return connected
}

// CanReachSystem checks if a ship can reach a target system (has fuel and connection)
func (smh *ShipMovementHelper) CanReachSystem(ship *entities.Ship, targetSystemID int) bool {
	// Check fuel
	if !ship.CanJump() {
		return false
	}

	// Check connection
	return smh.hasHyperlaneConnection(ship.CurrentSystem, targetSystemID)
}
