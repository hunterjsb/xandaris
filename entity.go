package main

import "image/color"

// EntityType represents the type of entity as a string
type EntityType string

// Entity is the interface that all system entities must implement
type Entity interface {
	GetID() int
	GetName() string
	GetType() EntityType
	GetOrbitDistance() float64 // Distance from system center (for orbital positioning)
	GetOrbitAngle() float64    // Angle in radians for orbital positioning
	GetColor() color.RGBA
	GetDescription() string
}

// AddEntity adds an entity to a system
func (s *System) AddEntity(entity Entity) {
	if s.Entities == nil {
		s.Entities = make([]Entity, 0)
	}
	s.Entities = append(s.Entities, entity)
}

// GetEntitiesByType returns all entities of a specific type
func (s *System) GetEntitiesByType(entityType EntityType) []Entity {
	result := make([]Entity, 0)
	for _, entity := range s.Entities {
		if entity.GetType() == entityType {
			result = append(result, entity)
		}
	}
	return result
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

// HasEntityType checks if the system contains any entities of the specified type
func (s *System) HasEntityType(entityType EntityType) bool {
	return s.CountEntitiesByType(entityType) > 0
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
