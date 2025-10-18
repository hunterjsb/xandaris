package entities

import (
	"fmt"
	"image/color"
)

// ResourceStorage tracks stored resources on a planet
type ResourceStorage struct {
	ResourceType string
	Amount       int
	Capacity     int
}

// Planet represents a planet entity in a star system
type Planet struct {
	BaseEntity
	Size                   int                         // Radius in pixels
	PlanetType             string                      // Subtype like "Terrestrial", "Gas Giant", etc.
	Population             int64                       // Number of inhabitants
	BasePopulationCapacity int64                       // Housing provided by the planet itself
	Resources              []Entity                    // Resource entities on this planet
	Buildings              []Entity                    // Building entities on this planet
	Temperature            int                         // Temperature in Celsius
	Atmosphere             string                      // Type of atmosphere
	HasRings               bool                        // Whether the planet has rings
	Habitability           int                         // Habitability score 0-100
	Owner                  string                      // Name of the player/faction who owns this planet
	StoredResources        map[string]*ResourceStorage // Resources stored on this planet (credits, materials, etc.)
	StorageCapacity        int                         // Total storage capacity
}

// NewPlanet creates a new planet entity
func NewPlanet(id int, name, planetType string, orbitDistance, orbitAngle float64, c color.RGBA) *Planet {
	planet := &Planet{
		BaseEntity: BaseEntity{
			ID:            id,
			Name:          name,
			Type:          EntityTypePlanet,
			SubType:       planetType,
			Color:         c,
			OrbitDistance: orbitDistance,
			OrbitAngle:    orbitAngle,
		},
		PlanetType:             planetType,
		Size:                   5,
		Temperature:            20,
		Atmosphere:             AtmosphereThin,
		Population:             0,
		BasePopulationCapacity: 0,
		Resources:              []Entity{},
		Buildings:              []Entity{},
		HasRings:               false,
		Habitability:           50,
		Owner:                  "", // Unowned by default
		StoredResources:        make(map[string]*ResourceStorage),
		StorageCapacity:        10000, // Base storage capacity
	}

	planet.RecalculateBasePopulationCapacity()
	return planet
}

// GetDescription returns a brief description of the planet
func (p *Planet) GetDescription() string {
	return fmt.Sprintf("%s (%s)", p.Name, p.PlanetType)
}

// GetClickRadius returns the click detection radius
func (p *Planet) GetClickRadius() float64 {
	return float64(p.Size) + 3 // Small margin for accurate clicking
}

// GetOwner returns the owner of this planet
func (p *Planet) GetOwner() string {
	return p.Owner
}

// GetContextMenuTitle implements ContextMenuProvider
func (p *Planet) GetContextMenuTitle() string {
	return p.Name
}

// GetContextMenuItems implements ContextMenuProvider
func (p *Planet) GetContextMenuItems() []string {
	items := []string{}

	items = append(items, fmt.Sprintf("Type: %s", p.PlanetType))
	items = append(items, fmt.Sprintf("Temperature: %d°C", p.Temperature))
	items = append(items, fmt.Sprintf("Atmosphere: %s", p.Atmosphere))
	capacity := p.GetTotalPopulationCapacity()
	if capacity > 0 {
		items = append(items, fmt.Sprintf("Population: %d/%d", p.Population, capacity))
	} else {
		items = append(items, fmt.Sprintf("Population: %d (no habitat)", p.Population))
	}
	items = append(items, fmt.Sprintf("Habitability: %d%%", p.Habitability))

	if p.HasRings {
		items = append(items, "Has planetary rings")
	}

	if len(p.Resources) > 0 {
		items = append(items, "") // Empty line
		items = append(items, fmt.Sprintf("Resources: %d deposits", len(p.Resources)))
		items = append(items, "View planet for details")
	}

	if len(p.Buildings) > 0 {
		items = append(items, fmt.Sprintf("Buildings: %d", len(p.Buildings)))
	}

	if p.Owner != "" {
		items = append(items, "") // Empty line
		items = append(items, fmt.Sprintf("Owner: %s", p.Owner))
	}

	// Show stored resources if any
	if len(p.StoredResources) > 0 {
		items = append(items, "") // Empty line
		items = append(items, "Stored Resources:")
		for resourceType, storage := range p.StoredResources {
			items = append(items, fmt.Sprintf("  %s: %d/%d", resourceType, storage.Amount, storage.Capacity))
		}
	}

	return items
}

// IsHabitable returns whether the planet can support life
func (p *Planet) IsHabitable() bool {
	return p.Temperature > -50 && p.Temperature < 60 &&
		p.Atmosphere != AtmosphereNone && p.Atmosphere != AtmosphereCorrosive &&
		p.PlanetType != "Lava"
}

// GetDetailedInfo returns detailed information about the planet
func (p *Planet) GetDetailedInfo() map[string]string {
	return map[string]string{
		"Type":         p.PlanetType,
		"Population":   fmt.Sprintf("%d", p.Population),
		"Capacity":     fmt.Sprintf("%d", p.GetTotalPopulationCapacity()),
		"Temperature":  fmt.Sprintf("%d°C", p.Temperature),
		"Atmosphere":   p.Atmosphere,
		"Size":         fmt.Sprintf("%d km radius", p.Size*1000),
		"Habitability": fmt.Sprintf("%d%%", p.Habitability),
	}
}

// RecalculateBasePopulationCapacity updates the base capacity based on planet traits
func (p *Planet) RecalculateBasePopulationCapacity() {
	if !p.IsHabitable() || p.Habitability <= 0 {
		p.BasePopulationCapacity = 0
		return
	}

	sizeFactor := float64(p.Size)
	habitabilityFactor := float64(p.Habitability) / 100.0

	base := int64(sizeFactor * sizeFactor * 2500.0 * habitabilityFactor)

	if base < 1000 {
		base = 1000
	}

	p.BasePopulationCapacity = base
}

// GetBasePopulationCapacity returns the capacity provided by the planet itself
func (p *Planet) GetBasePopulationCapacity() int64 {
	return p.BasePopulationCapacity
}

// GetBuildingPopulationCapacity returns housing provided by constructed buildings
func (p *Planet) GetBuildingPopulationCapacity() int64 {
	total := int64(0)
	planetID := fmt.Sprintf("%d", p.GetID())

	for _, entity := range p.Buildings {
		if building, ok := entity.(*Building); ok {
			capacity := building.GetEffectivePopulationCapacity()
			if capacity <= 0 {
				continue
			}
			if building.AttachmentType != "Planet" {
				continue
			}
			if building.AttachedTo != "" && building.AttachedTo != planetID {
				continue
			}
			total += capacity
		}
	}

	return total
}

// GetTotalPopulationCapacity returns total housing capacity (planet + buildings)
func (p *Planet) GetTotalPopulationCapacity() int64 {
	return p.GetBasePopulationCapacity() + p.GetBuildingPopulationCapacity()
}

// GetAvailablePopulationCapacity returns the remaining space before reaching capacity
func (p *Planet) GetAvailablePopulationCapacity() int64 {
	capacity := p.GetTotalPopulationCapacity()
	if capacity <= p.Population {
		return 0
	}
	return capacity - p.Population
}

// AddStoredResource adds an amount of a resource to the planet's storage
func (p *Planet) AddStoredResource(resourceType string, amount int) int {
	if p.StoredResources == nil {
		p.StoredResources = make(map[string]*ResourceStorage)
	}

	storage, exists := p.StoredResources[resourceType]
	if !exists {
		storage = &ResourceStorage{
			ResourceType: resourceType,
			Amount:       0,
			Capacity:     p.StorageCapacity,
		}
		p.StoredResources[resourceType] = storage
	}

	// Calculate how much can be added (limited by capacity)
	availableSpace := storage.Capacity - storage.Amount
	actualAmount := amount
	if actualAmount > availableSpace {
		actualAmount = availableSpace
	}

	storage.Amount += actualAmount
	return actualAmount // Return how much was actually added
}

// RemoveStoredResource removes an amount of a resource from the planet's storage
func (p *Planet) RemoveStoredResource(resourceType string, amount int) int {
	storage, exists := p.StoredResources[resourceType]
	if !exists {
		return 0
	}

	// Can't remove more than what's available
	actualAmount := amount
	if actualAmount > storage.Amount {
		actualAmount = storage.Amount
	}

	storage.Amount -= actualAmount
	return actualAmount // Return how much was actually removed
}

// GetStoredAmount returns the amount of a specific resource stored
func (p *Planet) GetStoredAmount(resourceType string) int {
	storage, exists := p.StoredResources[resourceType]
	if !exists {
		return 0
	}
	return storage.Amount
}

// HasStoredResource checks if the planet has at least a certain amount of a resource
func (p *Planet) HasStoredResource(resourceType string, amount int) bool {
	return p.GetStoredAmount(resourceType) >= amount
}

// GetTotalStorageUsed returns the total amount of storage space used
func (p *Planet) GetTotalStorageUsed() int {
	total := 0
	for _, storage := range p.StoredResources {
		total += storage.Amount
	}
	return total
}

// GetStorageUtilization returns storage usage as a percentage
func (p *Planet) GetStorageUtilization() float64 {
	if p.StorageCapacity == 0 {
		return 0
	}
	return float64(p.GetTotalStorageUsed()) / float64(p.StorageCapacity) * 100.0
}
