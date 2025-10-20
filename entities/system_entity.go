package entities

import (
	"fmt"
	"image/color"
)

// System represents a star system
type System struct {
	ID          int
	X           float64
	Y           float64
	Name        string
	Color       color.RGBA
	Connections []int // IDs of connected systems
	Entities    []Entity
}

// Hyperlane represents a connection between two systems
type Hyperlane struct {
	From int
	To   int
}

// AddEntity adds an entity to the system
func (s *System) AddEntity(entity Entity) {
	if s.Entities == nil {
		s.Entities = make([]Entity, 0)
	}
	s.Entities = append(s.Entities, entity)
}

// GetEntities returns all entities in the system
func (s *System) GetEntities() []Entity {
	return s.Entities
}

// GetEntitiesByType returns all entities of a specific type
func (s *System) GetEntitiesByType(entityType EntityType) []Entity {
	var result []Entity
	for _, entity := range s.Entities {
		if entity.GetType() == entityType {
			result = append(result, entity)
		}
	}
	return result
}

// HasEntityType checks if the system has any entities of a specific type
func (s *System) HasEntityType(entityType EntityType) bool {
	for _, entity := range s.Entities {
		if entity.GetType() == entityType {
			return true
		}
	}
	return false
}

// GetContextMenuTitle implements ContextMenuProvider
func (s *System) GetContextMenuTitle() string {
	return s.Name
}

// GetContextMenuItems implements ContextMenuProvider
func (s *System) GetContextMenuItems() []string {
	items := []string{}

	// Add star information first
	starEntities := s.GetEntitiesByType(EntityTypeStar)
	if len(starEntities) > 0 {
		star := starEntities[0]
		items = append(items, fmt.Sprintf("Star: %s", star.GetDescription()))
		items = append(items, "") // Empty line for spacing
	}

	// Add entity counts summary
	planetCount := len(s.GetEntitiesByType(EntityTypePlanet))
	stationCount := len(s.GetEntitiesByType(EntityTypeStation))

	items = append(items, fmt.Sprintf("Planets: %d", planetCount))
	if stationCount > 0 {
		items = append(items, fmt.Sprintf("Stations: %d", stationCount))
	}
	items = append(items, "") // Empty line for spacing

	// List planets
	for _, entity := range s.GetEntitiesByType(EntityTypePlanet) {
		items = append(items, fmt.Sprintf("  - %s", entity.GetDescription()))
	}

	// List stations
	for _, entity := range s.GetEntitiesByType(EntityTypeStation) {
		items = append(items, fmt.Sprintf("  - %s", entity.GetDescription()))
	}

	return items
}

// GetPosition implements Clickable interface
func (s *System) GetPosition() (float64, float64) {
	return s.X, s.Y
}

// GetClickRadius implements Clickable interface
func (s *System) GetClickRadius(view string) float64 {
	return float64(8) // circleRadius
}

// HasOwnershipByPlayer checks if the system contains any planets owned by the specified player
func (s *System) HasOwnershipByPlayer(playerName string) bool {
	for _, entity := range s.Entities {
		if planet, ok := entity.(*Planet); ok {
			if planet.Owner == playerName {
				return true
			}
		}
		if station, ok := entity.(*Station); ok {
			if station.IsPlayerOwned() && station.Owner == playerName {
				return true
			}
		}
	}
	return false
}

// GetEntityByID finds an entity by its ID
func (s *System) GetEntityByID(id int) Entity {
	for _, entity := range s.Entities {
		if entity.GetID() == id {
			return entity
		}
	}
	return nil
}

// CountEntities returns the total number of entities in the system
func (s *System) CountEntities() int {
	return len(s.Entities)
}

// CountEntitiesByType returns the count of entities of a specific type
func (s *System) CountEntitiesByType(entityType EntityType) int {
	count := 0
	for _, entity := range s.Entities {
		if entity.GetType() == entityType {
			count++
		}
	}
	return count
}

// RemoveEntity removes an entity from the system by ID
func (s *System) RemoveEntity(entityID int) bool {
	for i, entity := range s.Entities {
		if entity.GetID() == entityID {
			// Remove entity by slicing
			s.Entities = append(s.Entities[:i], s.Entities[i+1:]...)
			return true
		}
	}
	return false
}

// GetEntitiesInOrbitRange returns entities within a certain orbital distance range
func (s *System) GetEntitiesInOrbitRange(minDistance, maxDistance float64) []Entity {
	result := make([]Entity, 0)
	for _, entity := range s.Entities {
		distance := entity.GetOrbitDistance()
		if distance >= minDistance && distance <= maxDistance {
			result = append(result, entity)
		}
	}
	return result
}
