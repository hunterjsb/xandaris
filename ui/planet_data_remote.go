//go:build !js

package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/hunterjsb/xandaris/api"
	"github.com/hunterjsb/xandaris/entities"
)

// remoteCache holds cached API responses with timestamps.
type remoteCache struct {
	mu              sync.RWMutex
	planet          *PlanetDataResult
	rates           *RatesData
	queue           []ConstructionItemData
	lastPlanetFetch time.Time
	lastRatesFetch  time.Time
	lastQueueFetch  time.Time
	fetching        bool
}

// getRemoteCache lazily initializes the remote cache on the provider.
func (p *PlanetDataProvider) getRemoteCache() *remoteCache {
	if p.rc == nil {
		p.rc = &remoteCache{}
	}
	return p.rc.(*remoteCache)
}

// updateRemote handles remote-mode data refresh with background fetching.
func (p *PlanetDataProvider) updateRemote() {
	rc := p.getRemoteCache()

	rc.mu.RLock()
	fetching := rc.fetching
	stalePlanet := time.Since(rc.lastPlanetFetch) > 2*time.Second
	staleRates := time.Since(rc.lastRatesFetch) > 2*time.Second
	staleQueue := time.Since(rc.lastQueueFetch) > 2*time.Second
	rc.mu.RUnlock()

	if fetching || p.planetID == 0 {
		return
	}

	if stalePlanet || staleRates || staleQueue || p.refreshRequired {
		rc.mu.Lock()
		rc.fetching = true
		rc.mu.Unlock()
		p.refreshRequired = false

		go p.fetchRemoteData(stalePlanet || p.refreshRequired, staleRates || p.refreshRequired, staleQueue || p.refreshRequired)
	}

	// Return cached data in the meantime
	rc.mu.RLock()
	p.cachedPD = rc.planet
	p.cachedRates = rc.rates
	p.cachedItems = rc.queue
	rc.mu.RUnlock()
}

func (p *PlanetDataProvider) fetchRemoteData(fetchPlanet, fetchRates, fetchQueue bool) {
	rc := p.getRemoteCache()
	defer func() {
		rc.mu.Lock()
		rc.fetching = false
		rc.mu.Unlock()
	}()

	planetID := p.planetID
	if fetchPlanet {
		p.fetchPlanetDetail(planetID)
	}
	if fetchRates {
		p.fetchPlanetRates(planetID)
	}
	if fetchQueue {
		p.fetchConstructionQueue(planetID)
	}
}

func (p *PlanetDataProvider) fetchPlanetDetail(planetID int) {
	data, err := p.remoteGet(fmt.Sprintf("/api/planet/%d", planetID))
	if err != nil {
		return
	}

	var resp struct {
		OK   bool             `json:"ok"`
		Data api.PlanetDetail `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return
	}

	pd := &PlanetDataResult{
		Owner:         resp.Data.Owner,
		Population:    resp.Data.Population,
		Happiness:     resp.Data.Happiness,
		PowerConsumed: resp.Data.PowerConsumed,
		PowerRatio:    resp.Data.PowerRatio,
	}

	storage := make([]StoredResourceEntry, 0, len(resp.Data.StoredResources))
	for resType, amount := range resp.Data.StoredResources {
		storage = append(storage, StoredResourceEntry{
			ResourceType: resType,
			Amount:       amount,
			Capacity:     1000,
		})
	}
	sort.Slice(storage, func(i, j int) bool {
		return storage[i].ResourceType < storage[j].ResourceType
	})
	pd.StoredResources = storage

	rc := p.getRemoteCache()
	rc.mu.Lock()
	rc.planet = pd
	rc.lastPlanetFetch = time.Now()
	rc.mu.Unlock()
}

func (p *PlanetDataProvider) fetchPlanetRates(planetID int) {
	data, err := p.remoteGet(fmt.Sprintf("/api/planets/rates/%d", planetID))
	if err != nil {
		return
	}

	var resp struct {
		OK   bool            `json:"ok"`
		Data api.PlanetRates `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return
	}

	rc := p.getRemoteCache()
	rc.mu.Lock()
	rc.rates = &RatesData{NetFlow: resp.Data.NetFlow}
	rc.lastRatesFetch = time.Now()
	rc.mu.Unlock()
}

func (p *PlanetDataProvider) fetchConstructionQueue(planetID int) {
	data, err := p.remoteGet("/api/construction")
	if err != nil {
		return
	}

	var resp struct {
		OK   bool                        `json:"ok"`
		Data []api.ConstructionQueueItem `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return
	}

	// Filter by planet — match location to planet ID or resource IDs on this planet
	planetIDStr := fmt.Sprintf("%d", planetID)
	localPlanet := p.findPlanet()

	var items []ConstructionItemData
	for _, item := range resp.Data {
		match := item.Location == planetIDStr
		if !match && localPlanet != nil {
			for _, resEntity := range localPlanet.Resources {
				if fmt.Sprintf("%d", resEntity.GetID()) == item.Location {
					match = true
					break
				}
			}
		}
		if match {
			items = append(items, ConstructionItemData{
				ID:             item.ID,
				Name:           item.Name,
				Location:       item.Location,
				Progress:       item.Progress,
				RemainingTicks: item.RemainingTicks,
			})
		}
	}

	rc := p.getRemoteCache()
	rc.mu.Lock()
	rc.queue = items
	rc.lastQueueFetch = time.Now()
	rc.mu.Unlock()
}

func (p *PlanetDataProvider) remoteGet(endpoint string) ([]byte, error) {
	req, err := http.NewRequest("GET", p.serverURL+endpoint, nil)
	if err != nil {
		return nil, err
	}
	if p.apiKey != "" {
		req.Header.Set("X-API-Key", p.apiKey)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// populateRemoteEntities creates synthetic resource and building entities from
// API data so the orbital rendering code works unchanged in remote mode.
func (p *PlanetDataProvider) populateRemoteEntities(planet *entities.Planet) bool {
	data, err := p.remoteGet(fmt.Sprintf("/api/planet/%d", p.planetID))
	if err != nil {
		return false
	}

	var resp struct {
		OK   bool             `json:"ok"`
		Data api.PlanetDetail `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return false
	}

	// Update planet fields
	planet.Population = resp.Data.Population
	planet.Happiness = resp.Data.Happiness
	planet.ProductivityBonus = resp.Data.ProductivityBonus
	planet.PowerGenerated = resp.Data.PowerGenerated
	planet.PowerConsumed = resp.Data.PowerConsumed

	// Update stored resources
	for resType, amount := range resp.Data.StoredResources {
		if s, ok := planet.StoredResources[resType]; ok && s != nil {
			s.Amount = amount
		} else {
			planet.StoredResources[resType] = &entities.ResourceStorage{
				ResourceType: resType,
				Amount:       amount,
				Capacity:     1000,
			}
		}
	}

	// Rebuild resource entities if count changed
	if len(resp.Data.ResourceDeposits) != len(planet.Resources) {
		planet.Resources = make([]entities.Entity, 0, len(resp.Data.ResourceDeposits))
		for i, dep := range resp.Data.ResourceDeposits {
			angle := float64(i) / float64(len(resp.Data.ResourceDeposits)) * 2 * math.Pi
			res := entities.NewResource(
				dep.ID,
				fmt.Sprintf("%s Deposit", dep.ResourceType),
				dep.ResourceType,
				float64(planet.Size)*1.5,
				angle,
				entities.ResourceColor(dep.ResourceType),
			)
			res.Abundance = dep.Abundance
			res.ExtractionRate = dep.ExtractionRate
			res.Size = 8
			res.NodePosition = angle
			planet.Resources = append(planet.Resources, res)
		}
		return true
	}

	// Update in place
	for i, dep := range resp.Data.ResourceDeposits {
		if i < len(planet.Resources) {
			if res, ok := planet.Resources[i].(*entities.Resource); ok {
				res.Abundance = dep.Abundance
				res.ExtractionRate = dep.ExtractionRate
			}
		}
	}

	// Rebuild building entities if count changed
	if len(resp.Data.Buildings) != len(planet.Buildings) {
		planet.Buildings = make([]entities.Entity, 0, len(resp.Data.Buildings))
		for i, bld := range resp.Data.Buildings {
			angle := float64(i) / float64(len(resp.Data.Buildings)) * 2 * math.Pi
			b := entities.NewBuilding(
				-(i + 1),
				bld.Type,
				bld.Type,
				float64(planet.Size)*2.0,
				angle,
				entities.BuildingColor(bld.Type),
			)
			b.Level = bld.Level
			b.IsOperational = bld.IsOperational
			b.Owner = resp.Data.Owner
			planet.Buildings = append(planet.Buildings, b)
		}
		return true
	}

	for i, bld := range resp.Data.Buildings {
		if i < len(planet.Buildings) {
			if b, ok := planet.Buildings[i].(*entities.Building); ok {
				b.Level = bld.Level
				b.IsOperational = bld.IsOperational
			}
		}
	}

	return false
}
