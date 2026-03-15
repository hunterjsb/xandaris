package ui

import (
	"fmt"
	"sort"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/tickable"
)

// StoredResourceEntry is a sorted, snapshot-friendly view of a single stored resource.
type StoredResourceEntry struct {
	ResourceType string
	Amount       int
	Capacity     int
}

// PlanetDataResult holds a snapshot of planet data for the UI.
type PlanetDataResult struct {
	Owner           string
	Population      int64
	Happiness       float64
	PowerConsumed   float64
	PowerRatio      float64
	StoredResources []StoredResourceEntry
}

// RatesData holds net production/consumption rates per resource.
type RatesData struct {
	NetFlow map[string]float64
}

// ConstructionItemData is a UI-friendly snapshot of a construction queue entry.
type ConstructionItemData struct {
	ID             string
	Name           string
	Location       string
	Progress       int
	RemainingTicks int
}

// PlanetDataProvider abstracts local vs remote data access for planet UI components.
type PlanetDataProvider struct {
	ctx             UIContext
	remote          bool
	serverURL       string
	apiKey          string
	planetID        int
	cachedPlanet    *entities.Planet
	cachedPD        *PlanetDataResult
	cachedRates     *RatesData
	cachedItems     []ConstructionItemData
	refreshRequired bool
	rc              interface{} // *remoteCache on desktop, nil on WASM
}

// NewPlanetDataProvider creates a new planet data provider.
func NewPlanetDataProvider(ctx UIContext, isRemote bool, serverURL, apiKey string) *PlanetDataProvider {
	return &PlanetDataProvider{
		ctx:       ctx,
		remote:    isRemote,
		serverURL: serverURL,
		apiKey:    apiKey,
	}
}

// SetPlanetID sets the planet to provide data for.
func (p *PlanetDataProvider) SetPlanetID(id int) {
	if p.planetID != id {
		p.planetID = id
		p.refreshRequired = true
	}
}

// Update refreshes cached data. Called once per frame from the view.
func (p *PlanetDataProvider) Update() {
	if p.remote {
		p.updateRemote()
		return
	}
	planet := p.findPlanet()
	if planet == nil {
		// Keep stale cache rather than flickering to nil
		return
	}
	p.cachedPlanet = planet
	p.cachedPD = p.buildPlanetData(planet)
	p.cachedRates = p.buildRatesData(planet)
	p.cachedItems = p.buildConstructionItems()
	p.refreshRequired = false
}

// GetPlanetData returns the cached planet data snapshot.
func (p *PlanetDataProvider) GetPlanetData() *PlanetDataResult {
	return p.cachedPD
}

// GetRatesData returns the cached production/consumption rates.
func (p *PlanetDataProvider) GetRatesData() *RatesData {
	return p.cachedRates
}

// GetConstructionItems returns a snapshot of the current construction queue.
func (p *PlanetDataProvider) GetConstructionItems() []ConstructionItemData {
	return p.cachedItems
}

// ForceRefresh marks data as stale so it refreshes on the next Update.
func (p *PlanetDataProvider) ForceRefresh() {
	p.refreshRequired = true
}

// IsRemote returns true if this provider fetches data from a remote server.
func (p *PlanetDataProvider) IsRemote() bool {
	return p.remote
}

// HasMineQueued checks if a mine is already queued for a resource node.
func (p *PlanetDataProvider) HasMineQueued(resourceLocation string) bool {
	cs := tickable.GetConstructionSystem()
	if cs != nil {
		return cs.HasMineInQueue(resourceLocation)
	}
	return false
}

// PopulatePlanetEntities refreshes a planet's buildings/resources from the
// server state. For local play this is a no-op (pointers are shared).
// Returns true if entities were updated.
func (p *PlanetDataProvider) PopulatePlanetEntities(planet *entities.Planet) bool {
	if !p.remote {
		return false
	}
	return p.populateRemoteEntities(planet)
}

// findPlanet locates the planet entity by ID from the game state.
func (p *PlanetDataProvider) findPlanet() *entities.Planet {
	if p.planetID == 0 {
		return nil
	}
	for _, sys := range p.ctx.GetState().Systems {
		for _, entity := range sys.Entities {
			if planet, ok := entity.(*entities.Planet); ok {
				if planet.GetID() == p.planetID {
					return planet
				}
			}
		}
	}
	return nil
}

// buildPlanetData creates a PlanetDataResult snapshot from a live planet.
func (p *PlanetDataProvider) buildPlanetData(planet *entities.Planet) *PlanetDataResult {
	pd := &PlanetDataResult{
		Owner:         planet.Owner,
		Population:    planet.Population,
		Happiness:     planet.Happiness,
		PowerConsumed: planet.PowerConsumed,
		PowerRatio:    planet.GetPowerRatio(),
	}

	// Build sorted resource list
	for resourceType, storage := range planet.StoredResources {
		pd.StoredResources = append(pd.StoredResources, StoredResourceEntry{
			ResourceType: resourceType,
			Amount:       storage.Amount,
			Capacity:     storage.Capacity,
		})
	}
	sort.Slice(pd.StoredResources, func(i, j int) bool {
		return pd.StoredResources[i].ResourceType < pd.StoredResources[j].ResourceType
	})

	return pd
}

// buildRatesData computes net production/consumption rates for the planet.
func (p *PlanetDataProvider) buildRatesData(planet *entities.Planet) *RatesData {
	flow := make(map[string]float64)

	// Mine production
	for _, resEntity := range planet.Resources {
		res, ok := resEntity.(*entities.Resource)
		if !ok || res.Abundance <= 0 {
			continue
		}
		resIDStr := fmt.Sprintf("%d", res.GetID())
		multiplier := 0.0
		for _, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok {
				if b.BuildingType == "Mine" && b.AttachedTo == resIDStr && b.IsOperational {
					multiplier += b.GetStaffingRatio() * b.ProductionBonus
				}
			}
		}
		if multiplier > 0 {
			af := float64(res.Abundance) / 70.0
			if af > 1.0 {
				af = 1.0
			}
			if af < 0.1 {
				af = 0.1
			}
			flow[res.ResourceType] += 8.0 * res.ExtractionRate * multiplier * af
		}
	}

	// Refinery: +Fuel, -Oil
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok && b.BuildingType == "Refinery" && b.IsOperational {
			lm := 1.0 + float64(b.Level-1)*0.3
			flow["Fuel"] += 3.0 * lm
			flow["Oil"] -= 2.0 * lm
		}
	}

	// Factory: +Electronics, -Rare Metals, -Iron
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok && b.BuildingType == "Factory" && b.IsOperational {
			lm := 1.0 + float64(b.Level-1)*0.3
			flow["Electronics"] += 2.0 * lm
			flow["Rare Metals"] -= 2.0 * lm
			flow["Iron"] -= 1.0 * lm
		}
	}

	// Population consumption
	for _, rate := range economy.PopulationConsumption {
		flow[rate.ResourceType] -= float64(planet.Population) / rate.PopDivisor * rate.PerPopulation
	}

	// Building upkeep
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok && b.IsOperational {
			if upkeeps, found := economy.BuildingResourceUpkeep[b.BuildingType]; found {
				for _, u := range upkeeps {
					flow[u.ResourceType] -= float64(u.Amount)
				}
			}
		}
	}

	return &RatesData{NetFlow: flow}
}

// buildConstructionItems creates a snapshot of the construction queue for the human player.
func (p *PlanetDataProvider) buildConstructionItems() []ConstructionItemData {
	cs := tickable.GetConstructionSystem()
	if cs == nil || p.ctx.GetState().HumanPlayer == nil {
		return nil
	}

	items := cs.GetConstructionsByOwner(p.ctx.GetState().HumanPlayer.Name)

	// Sort by start time for stable ordering
	sort.Slice(items, func(i, j int) bool {
		return items[i].Started < items[j].Started
	})

	result := make([]ConstructionItemData, 0, len(items))
	for _, item := range items {
		item.Mutex.RLock()
		cid := ConstructionItemData{
			ID:             item.ID,
			Name:           item.Name,
			Location:       item.Location,
			Progress:       item.Progress,
			RemainingTicks: item.RemainingTicks,
		}
		item.Mutex.RUnlock()
		result = append(result, cid)
	}
	return result
}
