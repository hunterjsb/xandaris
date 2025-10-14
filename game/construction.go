package game

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/tickable"
)

// ConstructionHandler handles construction completion events
type ConstructionHandler struct {
	systems      []*entities.System
	players      []*entities.Player
	tickManager  TickManagerInterface
}

// TickManagerInterface provides access to tick information
type TickManagerInterface interface {
	GetCurrentTick() int64
}

// NewConstructionHandler creates a new construction handler
func NewConstructionHandler(systems []*entities.System, players []*entities.Player, tickManager TickManagerInterface) *ConstructionHandler {
	return &ConstructionHandler{
		systems:      systems,
		players:      players,
		tickManager:  tickManager,
	}
}

// HandleConstructionComplete adds completed buildings/ships to the game
func (ch *ConstructionHandler) HandleConstructionComplete(completion tickable.ConstructionCompletion) {
	// Handle ship construction
	if completion.Item.Type == "Ship" {
		ch.handleShipConstruction(completion)
		return
	}

	// Handle building construction
	// Find the planet or resource by ID
	locationID := completion.Location

	// Search all systems for the entity
	for _, system := range ch.systems {
		for _, entity := range system.Entities {
			// Check planets
			if planet, ok := entity.(*entities.Planet); ok {
				if fmt.Sprintf("%d", planet.GetID()) == locationID {
					// Found the planet, add building
					building := ch.createBuildingFromCompletion(completion, planet)
					if building != nil {
						planet.Buildings = append(planet.Buildings, building)

						// Initialize Fuel storage when a refinery is built
						if b, ok := building.(*entities.Building); ok && b.BuildingType == "Refinery" {
							// Ensure the planet has Fuel storage initialized
							if _, exists := planet.StoredResources["Fuel"]; !exists {
								planet.AddStoredResource("Fuel", 0) // Initialize with 0 fuel
							}
						}
					}
					return
				}

				// Check resources on this planet
				for _, resource := range planet.Resources {
					if fmt.Sprintf("%d", resource.GetID()) == locationID {
						// Found the resource, add building
						building := ch.createBuildingFromCompletion(completion, resource)
						if building != nil {
							// Buildings on resources need to be tracked somewhere
							// For now, we'll add to the parent planet
							planet.Buildings = append(planet.Buildings, building)
						}
					}
				}
			}
		}
	}
}

// handleShipConstruction spawns a completed ship
func (ch *ConstructionHandler) handleShipConstruction(completion tickable.ConstructionCompletion) {
	// Parse location to find planet (format: "planet_<ID>")
	var planetID int
	fmt.Sscanf(completion.Location, "planet_%d", &planetID)

	// Find the planet and system
	var targetPlanet *entities.Planet
	var targetSystem *entities.System

	for _, system := range ch.systems {
		for _, entity := range system.Entities {
			if planet, ok := entity.(*entities.Planet); ok {
				if planet.GetID() == planetID {
					targetPlanet = planet
					targetSystem = system
					break
				}
			}
		}
		if targetPlanet != nil {
			break
		}
	}

	if targetPlanet == nil || targetSystem == nil {
		fmt.Printf("[Game] ERROR: Could not find planet %d for ship construction\n", planetID)
		return
	}

	// Parse ship type from completion name
	shipType := entities.ShipType(completion.Item.Name)

	// Generate unique ship ID
	shipID := int(ch.tickManager.GetCurrentTick())*1000 + rand.Intn(1000)

	// Find owner player
	var owner *entities.Player
	for _, player := range ch.players {
		if player.Name == completion.Owner {
			owner = player
			break
		}
	}

	if owner == nil {
		fmt.Printf("[Game] ERROR: Could not find owner %s for ship\n", completion.Owner)
		return
	}

	// Create the ship
	shipName := fmt.Sprintf("%s %s-%d", owner.Name, shipType, len(owner.OwnedShips)+1)
	ship := entities.NewShip(shipID, shipName, shipType, targetSystem.ID, owner.Name, owner.Color)

	// Set ship position to orbit the PLANET, not the star
	// OrbitDistance = 0 means it orbits the planet at the planet's location
	// This makes it only visible in PlanetView, not SystemView
	ship.OrbitDistance = targetPlanet.OrbitDistance                                      // Store which planet's orbit
	ship.OrbitAngle = targetPlanet.OrbitAngle + 1.0 + float64(len(owner.OwnedShips))*0.3 // Spread ships around planet

	// Add ship to system BEFORE adding to player (important for save/load)
	targetSystem.AddEntity(ship)

	// Add ship to player's owned ships
	owner.AddOwnedShip(ship)

	fmt.Printf("[Game] Ship constructed: %s (%s) for %s at %s\n",
		shipName, shipType, owner.Name, targetPlanet.Name)
	fmt.Printf("[Game] Ship orbit: distance=%.2f, angle=%.2f (planet: dist=%.2f, angle=%.2f)\n",
		ship.OrbitDistance, ship.OrbitAngle, targetPlanet.OrbitDistance, targetPlanet.OrbitAngle)
}

// createBuildingFromCompletion creates a building entity from a completion
func (ch *ConstructionHandler) createBuildingFromCompletion(completion tickable.ConstructionCompletion, attachedTo entities.Entity) entities.Entity {
	// Generate parameters for building
	params := entities.GenerationParams{
		SystemID:      0,
		OrbitDistance: 20.0 + float64(len(ch.systems))*5.0, // Position around planet
		OrbitAngle:    float64(completion.Tick%628) / 100.0,
		SystemSeed:    completion.Tick,
	}

	// Get the appropriate building generator based on item name
	generators := entities.GetGeneratorsByType(entities.EntityTypeBuilding)
	for _, gen := range generators {
		if gen.GetSubType() == completion.Item.Type ||
			gen.GetSubType()+" Complex" == completion.Item.Name ||
			gen.GetSubType()+" Module" == completion.Item.Name ||
			"Orbital "+gen.GetSubType() == completion.Item.Name ||
			"Oil "+gen.GetSubType() == completion.Item.Name ||
			"Mining Complex" == completion.Item.Name && gen.GetSubType() == "Mine" {
			building := gen.Generate(params)
			if b, ok := building.(*entities.Building); ok {
				b.Owner = completion.Owner
				b.AttachedTo = completion.Location

				// If building a mine on a resource, link it to the resource node
				if b.BuildingType == "Mine" {
					if resource, ok := attachedTo.(*entities.Resource); ok {
						b.ResourceNodeID = resource.GetID()
						b.AttachmentType = "Resource"
					}
				}

				return b
			}
		}
	}

	return nil
}
