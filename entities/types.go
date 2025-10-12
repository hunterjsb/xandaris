package entities

import "image/color"

// EntityType represents the category of entity
type EntityType string

const (
	EntityTypePlanet  EntityType = "Planet"
	EntityTypeStation EntityType = "Station"
	// EntityTypeFleet    EntityType = "Fleet"
	// EntityTypeAsteroid EntityType = "Asteroid"
	EntityTypeStar     EntityType = "Star"
	EntityTypeResource EntityType = "Resource"
	EntityTypeBuilding EntityType = "Building"
)

// Entity is the core interface that all system entities must implement
type Entity interface {
	GetID() int
	GetName() string
	GetType() EntityType
	GetSubType() string // e.g., "Terrestrial", "Trading", "Military"
	GetOrbitDistance() float64
	GetOrbitAngle() float64
	GetColor() color.RGBA
	GetDescription() string

	// Positioning for rendering and interaction
	GetAbsolutePosition() (x, y float64)
	SetAbsolutePosition(x, y float64)

	// Click detection
	GetClickRadius() float64
}

// BaseEntity provides common entity functionality
type BaseEntity struct {
	ID            int
	Name          string
	Type          EntityType
	SubType       string
	Color         color.RGBA
	OrbitDistance float64
	OrbitAngle    float64
	AbsoluteX     float64
	AbsoluteY     float64
}

// GetID returns the entity ID
func (b *BaseEntity) GetID() int {
	return b.ID
}

// GetName returns the entity name
func (b *BaseEntity) GetName() string {
	return b.Name
}

// GetType returns the entity type
func (b *BaseEntity) GetType() EntityType {
	return b.Type
}

// GetSubType returns the entity subtype
func (b *BaseEntity) GetSubType() string {
	return b.SubType
}

// GetOrbitDistance returns the orbital distance
func (b *BaseEntity) GetOrbitDistance() float64 {
	return b.OrbitDistance
}

// GetOrbitAngle returns the orbital angle
func (b *BaseEntity) GetOrbitAngle() float64 {
	return b.OrbitAngle
}

// GetColor returns the entity color
func (b *BaseEntity) GetColor() color.RGBA {
	return b.Color
}

// GetAbsolutePosition returns the absolute x,y position
func (b *BaseEntity) GetAbsolutePosition() (float64, float64) {
	return b.AbsoluteX, b.AbsoluteY
}

// SetAbsolutePosition sets the absolute x,y position
func (b *BaseEntity) SetAbsolutePosition(x, y float64) {
	b.AbsoluteX = x
	b.AbsoluteY = y
}

// GetPosition returns position for Clickable interface
func (b *BaseEntity) GetPosition() (float64, float64) {
	return b.AbsoluteX, b.AbsoluteY
}

// ColorFromRGBA creates a color.RGBA from individual components
func ColorFromRGBA(r, g, b, a uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: a}
}
