package entities

import (
	"image/color"
	"sync"
)

// EntityType represents the category of entity
type EntityType string

const (
	EntityTypePlanet   EntityType = "Planet"
	EntityTypeStation  EntityType = "Station"
	EntityTypeShip     EntityType = "Ship"
	EntityTypeFleet    EntityType = "Fleet"
	EntityTypeStar     EntityType = "Star"
	EntityTypeResource EntityType = "Resource"
	EntityTypeBuilding EntityType = "Building"
	// EntityTypeAsteroid EntityType = "Asteroid"
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
	GetClickRadius(view string) float64

	// Universal attributes (with sensible defaults)
	GetOwner() string          // Empty string = unowned/neutral
	GetHP() (current, max int) // Return 0,0 if entity has no HP system
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

	// Attachment system for parent-child relationships
	attachments   []Entity
	attachmentsMu sync.RWMutex
	parentID      int
	attachmentPos AttachmentPosition // Position relative to parent
}

// AttachmentPosition defines where an entity is attached relative to its parent
type AttachmentPosition struct {
	OffsetX       float64 // Offset from parent center
	OffsetY       float64
	RelativeAngle float64 // Angle relative to parent
	RelativeScale float64 // Scale relative to parent (1.0 = same size)
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

// GetClickRadius returns the click detection radius (default implementation)
func (b *BaseEntity) GetClickRadius(view string) float64 {
	return 5.0 // Default click radius
}

// GetDescription returns empty description by default
func (b *BaseEntity) GetDescription() string {
	return ""
}

// GetOwner returns empty string (unowned) by default
func (b *BaseEntity) GetOwner() string {
	return ""
}

// GetHP returns 0,0 (no HP system) by default
func (b *BaseEntity) GetHP() (int, int) {
	return 0, 0
}

// ColorFromRGBA creates a color.RGBA from individual components
func ColorFromRGBA(r, g, b, a uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: a}
}

// Attachment system methods

// AttachEntity attaches a child entity to this entity
func (b *BaseEntity) AttachEntity(child Entity) {
	b.attachmentsMu.Lock()
	defer b.attachmentsMu.Unlock()

	// Check if already attached
	for _, existing := range b.attachments {
		if existing.GetID() == child.GetID() {
			return
		}
	}

	b.attachments = append(b.attachments, child)

	// Set parent ID if child is a BaseEntity
	if baseChild, ok := child.(*BaseEntity); ok {
		baseChild.parentID = b.ID
	}
}

// DetachEntity removes a child entity from this entity
func (b *BaseEntity) DetachEntity(childID int) bool {
	b.attachmentsMu.Lock()
	defer b.attachmentsMu.Unlock()

	for i, child := range b.attachments {
		if child.GetID() == childID {
			// Remove from slice
			b.attachments = append(b.attachments[:i], b.attachments[i+1:]...)

			// Clear parent ID if child is a BaseEntity
			if baseChild, ok := child.(*BaseEntity); ok {
				baseChild.parentID = 0
			}

			return true
		}
	}

	return false
}

// GetAttachments returns all attached entities
func (b *BaseEntity) GetAttachments() []Entity {
	b.attachmentsMu.RLock()
	defer b.attachmentsMu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]Entity, len(b.attachments))
	copy(result, b.attachments)
	return result
}

// GetAttachmentsByType returns all attached entities of a specific type
func (b *BaseEntity) GetAttachmentsByType(entityType EntityType) []Entity {
	b.attachmentsMu.RLock()
	defer b.attachmentsMu.RUnlock()

	result := make([]Entity, 0)
	for _, attachment := range b.attachments {
		if attachment.GetType() == entityType {
			result = append(result, attachment)
		}
	}
	return result
}

// HasAttachments returns whether this entity has any attachments
func (b *BaseEntity) HasAttachments() bool {
	b.attachmentsMu.RLock()
	defer b.attachmentsMu.RUnlock()
	return len(b.attachments) > 0
}

// GetParentID returns the ID of the parent entity (0 if no parent)
func (b *BaseEntity) GetParentID() int {
	return b.parentID
}

// SetAttachmentPosition sets the attachment position relative to parent
func (b *BaseEntity) SetAttachmentPosition(pos AttachmentPosition) {
	b.attachmentPos = pos
}

// GetAttachmentPosition returns the attachment position relative to parent
func (b *BaseEntity) GetAttachmentPosition() AttachmentPosition {
	return b.attachmentPos
}

// ClearAttachments removes all attached entities
func (b *BaseEntity) ClearAttachments() {
	b.attachmentsMu.Lock()
	defer b.attachmentsMu.Unlock()

	// Clear parent IDs
	for _, child := range b.attachments {
		if baseChild, ok := child.(*BaseEntity); ok {
			baseChild.parentID = 0
		}
	}

	b.attachments = nil
}
