package entities

import (
	"fmt"
	"image/color"
)

const DEFAULT_RESOURCE_CAPACITY = 1_000

// ResourceStorage tracks stored resources on a planet
type ResourceStorage struct {
	ResourceType string
	Amount       int
	Capacity     int
}

// Planet represents a planet entity in a star system
type Planet struct {
	BaseEntity
	Size            int                         // Radius in pixels
	PlanetType      string                      // Subtype like "Terrestrial", "Gas Giant", etc.
	Population      int64                       // Number of inhabitants
	Resources       []Entity                    // Resource entities on this planet
	Buildings       []Entity                    // Building entities on this planet
	Temperature     int                         // Temperature in Celsius
	Atmosphere      string                      // Type of atmosphere
	HasRings        bool                        // Whether the planet has rings
	Habitability    int                         // Habitability score 0-100
	Owner             string                      // Name of the player/faction who owns this planet
	StoredResources   map[string]*ResourceStorage // Resources stored on this planet (credits, materials, etc.)
	WorkforceTotal    int64                       // Total workforce available (subset of population)
	WorkforceUsed     int64                       // Workforce assigned to buildings/ships
	Happiness         float64                     // 0.0-1.0 based on resource fulfillment
	ProductivityBonus float64                     // Multiplier from happiness (0.5-1.5)
	TechLevel         float64                     // 0.0+ accumulated from Electronics consumption
	PowerGenerated    float64                     // MW generated this tick
	PowerConsumed     float64                     // MW demanded this tick
	PowerRatio        float64                     // 0.0-1.0 generated/consumed
	PowerHistory      []float64                   // last 50 power ratios for sparkline
	Specialties       map[string]float64          // workforce specialization bonuses (mining, refining, etc.)
}

// NewPlanet creates a new planet entity
func NewPlanet(id int, name, planetType string, orbitDistance, orbitAngle float64, c color.RGBA) *Planet {
	return &Planet{
		BaseEntity: BaseEntity{
			ID:            id,
			Name:          name,
			Type:          EntityTypePlanet,
			SubType:       planetType,
			Color:         c,
			OrbitDistance: orbitDistance,
			OrbitAngle:    orbitAngle,
		},
		PlanetType:      planetType,
		Size:            5,
		Temperature:     20,
		Atmosphere:      AtmosphereThin,
		Population:      0,
		Resources:       []Entity{},
		Buildings:       []Entity{},
		HasRings:        false,
		Habitability:    50,
		Owner:           "", // Unowned by default
		StoredResources: make(map[string]*ResourceStorage),
		WorkforceTotal:    0,
		WorkforceUsed:     0,
		Happiness:         0.5,
		ProductivityBonus: 1.0,
	}
}

// GetDescription returns a brief description of the planet
func (p *Planet) GetDescription() string {
	return fmt.Sprintf("%s (%s)", p.Name, p.PlanetType)
}

// GetClickRadius returns the click detection radius
func (p *Planet) GetClickRadius(view string) float64 {
	switch view {
	case "system":
		// In system view, planets are small, so a larger relative radius is needed.
		return float64(p.Size) + 8
	case "planet":
		// In planet view, the planet is large, so the radius can be closer to its visual size.
		return float64(p.Size) * 8 * 1.1 // 10% larger than the visual radius in planet view
	default:
		// Default for galaxy view or other contexts
		return float64(p.Size) + 5
	}
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

	if capacity > 0 {
		baseCap := p.GetBaseHousingCapacity()
		otherCap := capacity - baseCap
		otherCap = max(otherCap, 0)
		items = append(items, fmt.Sprintf("Housing: %d base / %d buildings", baseCap, otherCap))
	}

	if p.WorkforceTotal > 0 {
		items = append(items, fmt.Sprintf("Workforce: %d used / %d total", p.WorkforceUsed, p.WorkforceTotal))
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
	info := map[string]string{
		"Type":         p.PlanetType,
		"Population":   fmt.Sprintf("%d", p.Population),
		"Capacity":     fmt.Sprintf("%d", p.GetTotalPopulationCapacity()),
		"Temperature":  fmt.Sprintf("%d°C", p.Temperature),
		"Atmosphere":   p.Atmosphere,
		"Size":         fmt.Sprintf("%d km radius", p.Size*1000),
		"Habitability": fmt.Sprintf("%d%%", p.Habitability),
	}

	if p.WorkforceTotal > 0 {
		info["Workforce"] = fmt.Sprintf("%d / %d", p.WorkforceUsed, p.WorkforceTotal)
	}

	return info
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

// GetBaseBuilding returns the primary base structure for the planet, if present
func (p *Planet) GetBaseBuilding() *Building {
	for _, entity := range p.Buildings {
		if building, ok := entity.(*Building); ok {
			if building.BuildingType == BuildingBase {
				return building
			}
		}
	}
	return nil
}

// GetBaseHousingCapacity returns the housing provided by the base structure (0 if none)
func (p *Planet) GetBaseHousingCapacity() int64 {
	if base := p.GetBaseBuilding(); base != nil {
		return base.GetEffectivePopulationCapacity()
	}
	return 0
}

// SetBaseOwner updates the ownership metadata on the base structure, when present
func (p *Planet) SetBaseOwner(owner string) {
	if base := p.GetBaseBuilding(); base != nil {
		base.Owner = owner
	}
}

// GetTotalPopulationCapacity returns total housing capacity (planet + buildings).
// Tech level provides +10% capacity per level (e.g. Tech 2.0 = +20%).
func (p *Planet) GetTotalPopulationCapacity() int64 {
	base := p.GetBuildingPopulationCapacity()
	if p.TechLevel > 0 {
		techMult := 1.0 + p.TechLevel*0.1
		base = int64(float64(base) * techMult)
	}
	return base
}

// GetPowerRatio returns the power supply ratio (0.0-1.0). Returns 1.0 if no power needed.
func (p *Planet) GetPowerRatio() float64 {
	if p.PowerConsumed <= 0 {
		return 1.0
	}
	ratio := p.PowerGenerated / p.PowerConsumed
	if ratio > 1.0 {
		return 1.0
	}
	if ratio < 0 {
		return 0
	}
	return ratio
}

// CountBuildings returns the number of buildings of a given type on this planet.
func (p *Planet) CountBuildings(buildingType string) int {
	count := 0
	for _, be := range p.Buildings {
		if b, ok := be.(*Building); ok && b.BuildingType == buildingType {
			count++
		}
	}
	return count
}

// HasBuilding returns true if the planet has at least one building of the given type.
func (p *Planet) HasBuilding(buildingType string) bool {
	return p.CountBuildings(buildingType) > 0
}

// HasOperationalBuilding returns true if the planet has at least one operational building of the given type.
func (p *Planet) HasOperationalBuilding(buildingType string) bool {
	for _, be := range p.Buildings {
		if b, ok := be.(*Building); ok && b.BuildingType == buildingType && b.IsOperational {
			return true
		}
	}
	return false
}

// GetDockingFeeRate returns the trade fee rate for this planet's Trading Post.
// L1=5%, L2=4%, L3=3%, L4=2%, L5=1%. Returns 0 if no Trading Post.
func (p *Planet) GetDockingFeeRate() float64 {
	for _, be := range p.Buildings {
		if b, ok := be.(*Building); ok && b.BuildingType == BuildingTradingPost && b.IsOperational {
			rate := 0.06 - float64(b.Level)*0.01
			if rate < 0.01 {
				rate = 0.01
			}
			return rate
		}
	}
	return 0
}

// GetAvailablePopulationCapacity returns the remaining space before reaching capacity
func (p *Planet) GetAvailablePopulationCapacity() int64 {
	capacity := p.GetTotalPopulationCapacity()
	if capacity <= p.Population {
		return 0
	}
	return capacity - p.Population
}

// UpdateWorkforceTotals recalculates the total workforce pool based on current population
func (p *Planet) UpdateWorkforceTotals() {
	if p.Population <= 0 {
		p.WorkforceTotal = 0
		return
	}

	p.WorkforceTotal = p.Population / 2 // Simple baseline: 50% of population is workforce
	p.WorkforceTotal = max(p.WorkforceTotal, 0)
}

// GetAvailableWorkforce returns unassigned worker count
func (p *Planet) GetAvailableWorkforce() int64 {
	return max(p.WorkforceTotal-p.WorkforceUsed, 0)

}

// RebalanceWorkforce distributes workers across buildings based on availability
func (p *Planet) RebalanceWorkforce() {
	p.UpdateWorkforceTotals()

	available := max(p.WorkforceTotal, 0)

	p.WorkforceUsed = 0

	base := p.GetBaseBuilding()
	if base != nil {
		target := int64(base.WorkersRequired)
		if base.DesiredWorkers >= 0 {
			target = min(int64(base.DesiredWorkers), int64(base.WorkersRequired))
		}
		if target < 0 {
			target = 0
		}
		assign := min(target, available)
		base.SetWorkersAssigned(int(assign))
		available -= assign
		p.WorkforceUsed += assign
	}

	// Priority-based assignment: power generators first, then mines, then others.
	// This ensures colonies bootstrap power before anything else.
	priorityOrder := []string{
		BuildingGenerator,     // Power first — everything depends on it
		BuildingFusionReactor, // Advanced power
		BuildingRefinery,      // Fuel production (feeds generators — must come before mines)
		BuildingMine,          // Resource extraction
		BuildingFactory,       // Electronics production
		BuildingResearchLab,   // Passive electronics
	}
	assigned := make(map[*Building]bool)

	// Pass 1: assign by priority
	for _, priorityType := range priorityOrder {
		for _, entity := range p.Buildings {
			building, ok := entity.(*Building)
			if !ok || assigned[building] || (base != nil && building == base) {
				continue
			}
			if building.BuildingType != priorityType {
				continue
			}
			target := int64(building.WorkersRequired)
			if building.DesiredWorkers >= 0 {
				target = min(int64(building.DesiredWorkers), int64(building.WorkersRequired))
			}
			if target <= 0 {
				building.SetWorkersAssigned(0)
				assigned[building] = true
				continue
			}
			assign := min(target, available)
			building.SetWorkersAssigned(int(assign))
			available -= assign
			p.WorkforceUsed += assign
			assigned[building] = true
		}
	}

	// Pass 2: assign remaining buildings (Trading Post, Habitat, Shipyard, etc.)
	for _, entity := range p.Buildings {
		building, ok := entity.(*Building)
		if !ok || assigned[building] || (base != nil && building == base) {
			continue
		}
		target := int64(building.WorkersRequired)
		if building.DesiredWorkers >= 0 {
			target = min(int64(building.DesiredWorkers), int64(building.WorkersRequired))
		}
		if target <= 0 {
			building.SetWorkersAssigned(0)
			continue
		}
		assign := min(target, available)
		building.SetWorkersAssigned(int(assign))
		available -= assign
		p.WorkforceUsed += assign
	}

	if p.WorkforceUsed > p.WorkforceTotal {
		p.WorkforceUsed = p.WorkforceTotal
	}
	if p.WorkforceUsed < 0 {
		p.WorkforceUsed = 0
	}
}

// getStorageCapacity returns the effective per-resource storage cap for this planet.
// Tech level grants +20% capacity per level (e.g. Tech 2.0 = 1400, Tech 5.0 = 2000).
func (p *Planet) getStorageCapacity() int {
	cap := float64(DEFAULT_RESOURCE_CAPACITY) * (1.0 + p.TechLevel*0.2)
	return int(cap)
}

// AddStoredResource adds an amount of a resource to the planet's storage
func (p *Planet) AddStoredResource(resourceType string, amount int) int {
	if p.StoredResources == nil {
		p.StoredResources = make(map[string]*ResourceStorage)
	}

	effectiveCap := p.getStorageCapacity()

	storage, exists := p.StoredResources[resourceType]
	if !exists {
		storage = &ResourceStorage{
			ResourceType: resourceType,
			Amount:       0,
			Capacity:     effectiveCap,
		}
		p.StoredResources[resourceType] = storage
	} else {
		// Update capacity to reflect current tech level
		storage.Capacity = effectiveCap
	}

	// Calculate how much can be added (limited by capacity)
	availableSpace := storage.Capacity - storage.Amount
	actualAmount := amount
	actualAmount = min(actualAmount, availableSpace)

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
