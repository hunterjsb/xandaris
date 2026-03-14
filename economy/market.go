package economy

import (
	"math"
	"sync"

	"github.com/hunterjsb/xandaris/entities"
)

const (
	// demandBuffer: the market wants this many intervals of consumption in stock.
	// Lower = prices respond faster to scarcity. 10 intervals ≈ 10 seconds of buffer.
	demandBuffer = 10.0

	// EMA alpha: higher = faster price response.
	emaAlpha = 0.20

	// Spread: 5% each side of mid price.
	spreadPct = 0.05

	// Trade volume decay per price update cycle.
	tradeVolumeDecay = 0.90
)

// ResourceMarket tracks supply/demand/price state for a single resource.
type ResourceMarket struct {
	BasePrice       float64
	CurrentPrice    float64
	BuyPrice        float64
	SellPrice       float64
	TotalSupply     float64 // total stock across all planets
	TotalDemand     float64 // consumption rate per interval
	PriceVelocity   float64
	PriceHistory    []float64
	TradeVolumeBuy  float64 // decaying recent buy volume
	TradeVolumeSell float64 // decaying recent sell volume
}

// MarketSnapshot is a read-only copy of the full market state.
type MarketSnapshot struct {
	Resources map[string]ResourceMarket
}

// Market is the central price engine. Thread-safe via RWMutex.
type Market struct {
	resources    map[string]*ResourceMarket
	systemSupply map[int]map[string]float64 // per-system supply levels
	mu           sync.RWMutex
}

// NewMarket creates a market with entries for every known base-priced resource.
func NewMarket() *Market {
	m := &Market{
		resources: make(map[string]*ResourceMarket),
	}
	for name, base := range BasePrices {
		m.resources[name] = &ResourceMarket{
			BasePrice:    base,
			CurrentPrice: base,
			BuyPrice:     base * (1 + spreadPct),
			SellPrice:    base * (1 - spreadPct),
			PriceHistory: []float64{base},
		}
	}
	return m
}

// UpdatePricesWithSystems recalculates prices and also computes per-system supply data.
func (m *Market) UpdatePricesWithSystems(players []*entities.Player, systems []*entities.System) {
	// Build planet→system lookup
	planetSystem := make(map[int]int) // planet ID → system ID
	for _, sys := range systems {
		for _, entity := range sys.Entities {
			if planet, ok := entity.(*entities.Planet); ok {
				planetSystem[planet.GetID()] = sys.ID
			}
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Aggregate supply globally and per-system
	supply := make(map[string]float64)
	sysSupply := make(map[int]map[string]float64)

	for _, player := range players {
		if player == nil {
			continue
		}
		for _, planet := range player.OwnedPlanets {
			if planet == nil {
				continue
			}
			sysID := planetSystem[planet.GetID()]
			if _, ok := sysSupply[sysID]; !ok {
				sysSupply[sysID] = make(map[string]float64)
			}
			for resType, storage := range planet.StoredResources {
				if storage != nil {
					supply[resType] += float64(storage.Amount)
					sysSupply[sysID][resType] += float64(storage.Amount)
				}
			}
		}
	}
	m.systemSupply = sysSupply

	m.updatePricesLocked(supply)
}

// UpdatePrices recalculates all prices from current supply/demand across players.
func (m *Market) UpdatePrices(players []*entities.Player) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Aggregate supply (total stock) per resource across all players.
	supply := make(map[string]float64)
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, planet := range player.OwnedPlanets {
			if planet == nil {
				continue
			}
			for resType, storage := range planet.StoredResources {
				if storage != nil {
					supply[resType] += float64(storage.Amount)
				}
			}
		}
	}

	m.updatePricesLocked(supply)
}

// updatePricesLocked does the actual price computation (must be called with lock held).
func (m *Market) updatePricesLocked(supply map[string]float64) {
	for name, rm := range m.resources {
		rm.TotalSupply = supply[name]

		// Effective demand = consumption rate * buffer + trade buy pressure
		// This converts "flow" (units/interval) to "desired stock level"
		effectiveDemand := rm.TotalDemand*demandBuffer + rm.TradeVolumeBuy
		if effectiveDemand < 1 {
			effectiveDemand = 1
		}

		// Effective supply = stock + trade sell pressure (extra supply from sells)
		effectiveSupply := rm.TotalSupply + rm.TradeVolumeSell

		ratio := effectiveSupply / effectiveDemand

		// Clamp ratio to [0.2, 5.0]
		ratio = clamp(ratio, 0.2, 5.0)

		// Target price from supply/demand ratio
		targetPrice := rm.BasePrice / ratio

		// Smooth via EMA
		rm.CurrentPrice = rm.CurrentPrice*(1-emaAlpha) + targetPrice*emaAlpha

		// Bid/ask spread
		rm.BuyPrice = rm.CurrentPrice * (1 + spreadPct)
		rm.SellPrice = rm.CurrentPrice * (1 - spreadPct)

		// Clamp to [10%, 1000%] of base price — wide range for meaningful scarcity signals
		minPrice := rm.BasePrice * 0.10
		maxPrice := rm.BasePrice * 10.0
		rm.BuyPrice = clamp(rm.BuyPrice, minPrice, maxPrice)
		rm.SellPrice = clamp(rm.SellPrice, minPrice, maxPrice)
		rm.CurrentPrice = clamp(rm.CurrentPrice, minPrice, maxPrice)

		// Track velocity (change since last update)
		if len(rm.PriceHistory) > 0 {
			rm.PriceVelocity = rm.CurrentPrice - rm.PriceHistory[len(rm.PriceHistory)-1]
		}

		// Append to history (keep last 100)
		rm.PriceHistory = append(rm.PriceHistory, rm.CurrentPrice)
		if len(rm.PriceHistory) > 100 {
			rm.PriceHistory = rm.PriceHistory[len(rm.PriceHistory)-100:]
		}

		// Decay trade volumes
		rm.TradeVolumeBuy *= tradeVolumeDecay
		rm.TradeVolumeSell *= tradeVolumeDecay
	}
}

// SetDemand sets the consumption rate for a resource (units per interval).
func (m *Market) SetDemand(resourceType string, demand float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rm := m.getOrCreate(resourceType)
	rm.TotalDemand = demand
}

// AddTradeVolume bumps the trade pressure signal. Called by executor after trades.
func (m *Market) AddTradeVolume(resourceType string, quantity int, isBuy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rm := m.getOrCreate(resourceType)
	if isBuy {
		rm.TradeVolumeBuy += float64(quantity)
	} else {
		rm.TradeVolumeSell += float64(quantity)
	}
}

// GetLocalBuyPrice returns the buy price adjusted for local supply in a system.
func (m *Market) GetLocalBuyPrice(resource string, systemID int) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rm, ok := m.resources[resource]
	if !ok {
		return GetBasePrice(resource)
	}

	// Adjust by local supply ratio vs global average
	localSupply := 0.0
	if m.systemSupply != nil {
		if sysSup, ok := m.systemSupply[systemID]; ok {
			localSupply = sysSup[resource]
		}
	}

	// If no system supply data, return global price
	if m.systemSupply == nil || len(m.systemSupply) == 0 {
		return rm.BuyPrice
	}

	// Average supply per system
	avgSupply := rm.TotalSupply / float64(len(m.systemSupply))
	if avgSupply < 1 {
		avgSupply = 1
	}

	// Local adjustment: scarce locally = higher price, surplus = lower price
	ratio := localSupply / avgSupply
	if ratio < 0.1 {
		ratio = 0.1
	}
	if ratio > 5.0 {
		ratio = 5.0
	}

	localPrice := rm.BuyPrice / ratio
	minPrice := rm.BasePrice * 0.25
	maxPrice := rm.BasePrice * 4.0
	return clamp(localPrice, minPrice, maxPrice)
}

// GetLocalSellPrice returns the sell price adjusted for local supply in a system.
func (m *Market) GetLocalSellPrice(resource string, systemID int) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rm, ok := m.resources[resource]
	if !ok {
		return GetBasePrice(resource) * 0.9
	}

	if m.systemSupply == nil || len(m.systemSupply) == 0 {
		return rm.SellPrice
	}

	localSupply := 0.0
	if sysSup, ok := m.systemSupply[systemID]; ok {
		localSupply = sysSup[resource]
	}

	avgSupply := rm.TotalSupply / float64(len(m.systemSupply))
	if avgSupply < 1 {
		avgSupply = 1
	}

	ratio := localSupply / avgSupply
	if ratio < 0.1 {
		ratio = 0.1
	}
	if ratio > 5.0 {
		ratio = 5.0
	}

	localPrice := rm.SellPrice / ratio
	minPrice := rm.BasePrice * 0.25
	maxPrice := rm.BasePrice * 4.0
	return clamp(localPrice, minPrice, maxPrice)
}

// GetSnapshot returns a read-only copy of the market state.
func (m *Market) GetSnapshot() MarketSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snap := MarketSnapshot{
		Resources: make(map[string]ResourceMarket, len(m.resources)),
	}
	for name, rm := range m.resources {
		snap.Resources[name] = *rm
	}
	return snap
}

// GetBuyPrice returns the current buy price for a resource.
func (m *Market) GetBuyPrice(resourceType string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if rm, ok := m.resources[resourceType]; ok {
		return rm.BuyPrice
	}
	return GetBasePrice(resourceType)
}

// GetSellPrice returns the current sell price for a resource.
func (m *Market) GetSellPrice(resourceType string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if rm, ok := m.resources[resourceType]; ok {
		return rm.SellPrice
	}
	return GetBasePrice(resourceType) * 0.9
}

// ExecuteTrade is the legacy price-only calculator. Use TradeExecutor for full trades.
func (m *Market) ExecuteTrade(resourceType string, quantity int, buy bool) (int, bool) {
	m.mu.RLock()
	rm, ok := m.resources[resourceType]
	if !ok {
		m.mu.RUnlock()
		return 0, false
	}
	var price float64
	if buy {
		price = rm.BuyPrice
	} else {
		price = rm.SellPrice
	}
	m.mu.RUnlock()

	total := int(math.Round(price * float64(quantity)))
	if total <= 0 {
		total = quantity
	}

	return total, true
}

// RestoreMarket recreates a Market from a saved snapshot. Returns NewMarket() if snapshot is nil.
func RestoreMarket(snap *MarketSnapshot) *Market {
	if snap == nil {
		return NewMarket()
	}
	m := &Market{
		resources: make(map[string]*ResourceMarket),
	}
	for name, rm := range snap.Resources {
		cp := rm // copy
		m.resources[name] = &cp
	}
	// Ensure all base-priced resources exist
	for name, base := range BasePrices {
		if _, ok := m.resources[name]; !ok {
			m.resources[name] = &ResourceMarket{
				BasePrice:    base,
				CurrentPrice: base,
				BuyPrice:     base * (1 + spreadPct),
				SellPrice:    base * (1 - spreadPct),
				PriceHistory: []float64{base},
			}
		}
	}
	return m
}

// ComputeImportFee calculates the dynamic import/export fee rate for a resource.
// Returns 0.05-0.20 based on supply/demand ratio. Single source of truth.
func ComputeImportFee(totalSupply float64, totalDemand float64) float64 {
	if totalDemand <= 0 {
		return 0.10
	}
	ratio := totalSupply / (totalDemand * 10)
	if ratio > 2.0 {
		return 0.05
	}
	if ratio < 0.5 {
		return 0.20
	}
	fee := 0.15 - ratio*0.05
	if fee < 0.05 {
		return 0.05
	}
	return fee
}

// ComputeScarcity returns a human-readable scarcity label for a resource.
// Single source of truth — used by API and UI.
func ComputeScarcity(totalSupply, totalDemand float64) string {
	if totalSupply <= 0 {
		return "Depleted"
	}
	if totalDemand > 0 {
		ratio := totalSupply / (totalDemand * 10)
		if ratio > 3.0 {
			return "Abundant"
		}
		if ratio > 1.0 {
			return "Moderate"
		}
		if ratio > 0.3 {
			return "Scarce"
		}
		return "Critical"
	}
	if totalSupply > 500 {
		return "Abundant"
	}
	return "Moderate"
}

// GetTradeVolume returns total recent trade volume across all resources.
func (m *Market) GetTradeVolume() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	total := 0.0
	for _, rm := range m.resources {
		total += rm.TradeVolumeBuy + rm.TradeVolumeSell
	}
	return total
}

// getResourceMarket returns the market data for a resource (read-only, thread-safe).
func (m *Market) getResourceMarket(resourceType string) *ResourceMarket {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if rm, ok := m.resources[resourceType]; ok {
		return rm
	}
	return nil
}

func (m *Market) getOrCreate(resourceType string) *ResourceMarket {
	rm, ok := m.resources[resourceType]
	if !ok {
		base := GetBasePrice(resourceType)
		rm = &ResourceMarket{
			BasePrice:    base,
			CurrentPrice: base,
			BuyPrice:     base * (1 + spreadPct),
			SellPrice:    base * (1 - spreadPct),
			PriceHistory: []float64{base},
		}
		m.resources[resourceType] = rm
	}
	return rm
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
