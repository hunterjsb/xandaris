package tickable

import (
	"fmt"
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
	TickCount     int64 // diagnostic: how many ticks processed
	ShipsFound    int   // diagnostic: ships found in last tick
	MovingFound   int   // diagnostic: moving ships processed in last tick
}

// GetShipMovementDiag returns diagnostic counters
func GetShipMovementDiag() (ticks int64, shipsFound, movingFound int) {
	sys := GetSystemByName("ShipMovement")
	if sms, ok := sys.(*ShipMovementSystem); ok {
		return sms.TickCount, sms.ShipsFound, sms.MovingFound
	}
	return -1, -1, -1
}

// GetShipMovementPlayers returns player ship moving count from the system's perspective
func GetShipMovementPlayers() int {
	sys := GetSystemByName("ShipMovement")
	if sms, ok := sys.(*ShipMovementSystem); ok {
		ctx := sms.GetContext()
		if ctx == nil {
			return -1
		}
		game := ctx.GetGame()
		if game == nil {
			return -2
		}
		count := 0
		for _, p := range ctx.GetPlayers() {
			if p == nil { continue }
			for _, s := range p.OwnedShips {
				if s != nil && s.Status == entities.ShipStatusMoving {
					count++
				}
			}
		}
		return count
	}
	return -3
}

// OnTick processes ship movement each game tick
func (sms *ShipMovementSystem) OnTick(tick int64) {
	context := sms.GetContext()
	if context == nil {
		if tick%1000 == 0 {
			fmt.Println("[ShipMovement] ERROR: context is nil")
		}
		return
	}

	game := context.GetGame()
	if game == nil {
		if tick%1000 == 0 {
			fmt.Println("[ShipMovement] ERROR: game is nil")
		}
		return
	}

	systemsMap := game.GetSystemsMap()
	sms.TickCount++
	sms.ShipsFound = 0
	sms.MovingFound = 0

	// Process all ships — check both system entities and player-owned ships
	// to catch ships that aren't in a system's entity list (e.g. after API creation)
	seen := make(map[int]bool)

	for _, system := range systemsMap {
		for _, entity := range system.Entities {
			if ship, ok := entity.(*entities.Ship); ok {
				seen[ship.GetID()] = true
				sms.ShipsFound++
				if ship.Status == entities.ShipStatusMoving {
					sms.MovingFound++
				}
				sms.processShipMovement(ship, systemsMap)
			}
		}
	}

	// Count player ships with Moving status for diagnostics
	playerMoving := 0
	for _, player := range game.GetPlayers() {
		if player == nil { continue }
		for _, s := range player.OwnedShips {
			if s != nil && s.Status == entities.ShipStatusMoving {
				playerMoving++
			}
		}
	}
	if tick%500 == 0 && playerMoving > 0 {
		fmt.Printf("[ShipMovement] tick=%d sysShips=%d playerMoving=%d\n", tick, sms.ShipsFound, playerMoving)
	}

	// Sync player-owned ship status to system entity copies
	// (player ships and system entity ships can be different objects after save/load)
	for _, player := range game.GetPlayers() {
		if player == nil {
			continue
		}
		for _, pShip := range player.OwnedShips {
			if pShip == nil {
				continue
			}
			if !seen[pShip.GetID()] {
				// Ship not in any system — add it and process
				seen[pShip.GetID()] = true
				if sys := systemsMap[pShip.CurrentSystem]; sys != nil {
					sys.AddEntity(pShip)
				}
				sms.processShipMovement(pShip, systemsMap)
			} else if pShip.Status == entities.ShipStatusMoving {
				// Ship IS in system but system copy might have stale status
				if sys := systemsMap[pShip.CurrentSystem]; sys != nil {
					for _, e := range sys.Entities {
						if sysShip, ok := e.(*entities.Ship); ok && sysShip.GetID() == pShip.GetID() {
							if sysShip.Status != entities.ShipStatusMoving {
								// Sync status from player ship to system entity
								sysShip.Status = pShip.Status
								sysShip.TargetSystem = pShip.TargetSystem
								sysShip.TravelProgress = pShip.TravelProgress
								sysShip.CurrentFuel = pShip.CurrentFuel
								sms.MovingFound++
								sms.processShipMovement(sysShip, systemsMap)
							}
							break
						}
					}
				}
			}
		}
	}
}

// processShipMovement handles movement for a single ship
func (sms *ShipMovementSystem) processShipMovement(ship *entities.Ship, systems map[int]*entities.System) {
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
	targetSystem := systems[ship.TargetSystem]
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

	if tick := sms.GetContext().GetTick(); tick%500 == 0 {
		fmt.Printf("[ShipMovement] %s progress=%.2f%% speed=%.4f fuel=%d target=%d\n",
			ship.Name, ship.TravelProgress*100, travelSpeed, ship.CurrentFuel, ship.TargetSystem)
	}

	// Check if ship has arrived
	if ship.TravelProgress >= 1.0 {
		sms.arriveAtSystem(ship, targetSystem, systems)
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

	// Also update the ship in the system entity list (may be a different object)
	for _, e := range currentSystem.Entities {
		if sysShip, ok := e.(*entities.Ship); ok && sysShip.GetID() == ship.GetID() {
			sysShip.Status = entities.ShipStatusMoving
			sysShip.TargetSystem = targetSystemID
			sysShip.TravelProgress = 0.0
			sysShip.CurrentFuel = ship.CurrentFuel
			break
		}
	}

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

// FindPath returns a multi-hop path from one system to another via hyperlanes.
// Returns the path as a slice of system IDs (excluding source, including destination).
// Returns nil if no path exists.
func (smh *ShipMovementHelper) FindPath(fromID, toID int) []int {
	if fromID == toID {
		return []int{}
	}

	visited := make(map[int]bool)
	parent := make(map[int]int)
	queue := []int{fromID}
	visited[fromID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == toID {
			// Reconstruct path
			path := []int{}
			for n := toID; n != fromID; n = parent[n] {
				path = append([]int{n}, path...)
			}
			return path
		}

		for _, neighbor := range smh.GetConnectedSystems(current) {
			if !visited[neighbor] {
				visited[neighbor] = true
				parent[neighbor] = current
				queue = append(queue, neighbor)
			}
		}
	}

	return nil // no path exists
}

// AreSystemsConnected checks if two systems are reachable via any hyperlane path.
func (smh *ShipMovementHelper) AreSystemsConnected(fromID, toID int) bool {
	return smh.FindPath(fromID, toID) != nil
}
