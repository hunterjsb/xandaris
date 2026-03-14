package economy

import (
	"fmt"
	"math"
	"sync"

	"github.com/hunterjsb/xandaris/entities"
)

// TradeRecord logs a single completed trade.
type TradeRecord struct {
	Tick      int64
	Player    string
	Resource  string
	Quantity  int
	Action    string  // "buy" or "sell"
	UnitPrice float64
	Total     int
}

// TradeExecutor provides the single code path for all trades.
// Both the UI and API call into this to execute trades.
type TradeExecutor struct {
	market  *Market
	history []TradeRecord
	mu      sync.Mutex
	tick    int64

	// Systems reference for system-scoped trading.
	// When set, human trades are scoped to the trading planet's system.
	systems []*entities.System
}

// NewTradeExecutor creates a new executor bound to the given market.
func NewTradeExecutor(market *Market) *TradeExecutor {
	return &TradeExecutor{
		market:  market,
		history: make([]TradeRecord, 0, 200),
	}
}

// SetSystems stores the systems reference for system-scoped NPC stock lookups.
func (te *TradeExecutor) SetSystems(systems []*entities.System) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.systems = systems
}

// SetMarket swaps the market reference (used after load game).
func (te *TradeExecutor) SetMarket(market *Market) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.market = market
}

// SetTick updates the tick for trade log timestamps.
func (te *TradeExecutor) SetTick(tick int64) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.tick = tick
}

// Buy executes a purchase at a specific planet.
// For humans, resources come from NPC planets in the same system as tradingPlanet.
// For AI, the market acts as an abstract exchange (resources are created).
// If tradingPlanet is nil, falls back to auto-selecting the first planet with a Trading Post.
func (te *TradeExecutor) Buy(player *entities.Player, players []*entities.Player, resource string, quantity int, tradingPlanet ...*entities.Planet) (TradeRecord, error) {
	if player == nil {
		return TradeRecord{}, fmt.Errorf("no player")
	}
	if quantity <= 0 {
		return TradeRecord{}, fmt.Errorf("invalid quantity")
	}

	te.mu.Lock()
	defer te.mu.Unlock()

	// Get price
	price := te.market.GetBuyPrice(resource)
	total := int(math.Round(price * float64(quantity)))
	if total <= 0 {
		total = quantity
	}

	// Check credits
	if player.Credits < total {
		return TradeRecord{}, fmt.Errorf("insufficient credits (need %d, have %d)", total, player.Credits)
	}

	// Determine the trading planet
	var destPlanet *entities.Planet
	if len(tradingPlanet) > 0 && tradingPlanet[0] != nil {
		destPlanet = tradingPlanet[0]
	} else {
		destPlanet = firstPlanetWithTradingPost(player)
	}
	if destPlanet == nil {
		return TradeRecord{}, fmt.Errorf("no planet with trading post to receive goods")
	}

	// All trades are real: resources must come from other players' planets.
	// Try local system first (cheaper), fall back to galaxy-wide (with markup).
	var npcAvail int
	useSystemScope := false
	systemID := -1

	if player.IsHuman() {
		systemID = te.getSystemForPlanet(destPlanet)
		if systemID >= 0 {
			npcAvail = aggregateOtherStockInSystem(players, player, resource, systemID, te.systems)
			if npcAvail >= quantity {
				useSystemScope = true
			}
		}
	}
	if !useSystemScope {
		// Galaxy-wide fallback
		npcAvail = aggregateOtherStock(players, player, resource)
	}

	if npcAvail < quantity {
		return TradeRecord{}, fmt.Errorf("insufficient market stock (need %d, available %d)", quantity, npcAvail)
	}

	// Dynamic import fee for galaxy-wide human trades.
	// Fee scales with scarcity: 5% when plentiful (supply > demand*20),
	// up to 20% when scarce (supply < demand*5). Simulates shipping difficulty.
	if player.IsHuman() && !useSystemScope {
		feeRate := 0.10 // default 10%
		rm := te.market.getResourceMarket(resource)
		if rm != nil && rm.TotalDemand > 0 {
			ratio := rm.TotalSupply / (rm.TotalDemand * 10)
			if ratio > 2.0 {
				feeRate = 0.05 // plentiful: cheap shipping
			} else if ratio < 0.5 {
				feeRate = 0.20 // scarce: expensive to source
			} else {
				feeRate = 0.15 - ratio*0.05 // linear interpolation
				if feeRate < 0.05 {
					feeRate = 0.05
				}
			}
		}
		importFee := int(math.Round(float64(total) * feeRate))
		total += importFee
		if player.Credits < total {
			return TradeRecord{}, fmt.Errorf("insufficient credits with import fee (need %d, have %d)", total, player.Credits)
		}
	}

	if useSystemScope {
		removeFromOthersInSystem(players, player, resource, quantity, systemID, te.systems)
	} else {
		removeFromOthers(players, player, resource, quantity)
	}

	// Add to buyer's trading planet
	destPlanet.AddStoredResource(resource, quantity)
	player.Credits -= total

	// Bump trade volume on market
	te.market.AddTradeVolume(resource, quantity, true)

	record := TradeRecord{
		Tick:      te.tick,
		Player:    player.Name,
		Resource:  resource,
		Quantity:  quantity,
		Action:    "buy",
		UnitPrice: price,
		Total:     total,
	}
	te.appendRecord(record)

	fmt.Printf("[Trade] %s bought %d %s @ %.0f = %d credits\n",
		player.Name, quantity, resource, price, total)

	return record, nil
}

// Sell executes a sale from a specific planet.
// For humans, stock is taken from tradingPlanet only (not aggregated).
// For AI, resources are destroyed (absorbed by the abstract market).
// If tradingPlanet is nil, falls back to aggregating across all player planets.
func (te *TradeExecutor) Sell(player *entities.Player, players []*entities.Player, resource string, quantity int, tradingPlanet ...*entities.Planet) (TradeRecord, error) {
	if player == nil {
		return TradeRecord{}, fmt.Errorf("no player")
	}
	if quantity <= 0 {
		return TradeRecord{}, fmt.Errorf("invalid quantity")
	}

	te.mu.Lock()
	defer te.mu.Unlock()

	// Get price
	price := te.market.GetSellPrice(resource)
	total := int(math.Round(price * float64(quantity)))
	if total <= 0 {
		total = quantity
	}

	// Determine the trading planet
	var srcPlanet *entities.Planet
	if len(tradingPlanet) > 0 && tradingPlanet[0] != nil {
		srcPlanet = tradingPlanet[0]
	}

	if srcPlanet != nil {
		// Planet-scoped sell: stock comes from this planet only
		stored := srcPlanet.StoredResources[resource]
		planetStock := 0
		if stored != nil {
			planetStock = stored.Amount
		}
		if planetStock < quantity {
			return TradeRecord{}, fmt.Errorf("insufficient stock on %s (need %d, have %d)", srcPlanet.Name, quantity, planetStock)
		}
		srcPlanet.RemoveStoredResource(resource, quantity)
	} else {
		// Legacy: aggregate across all player planets
		playerStock := aggregatePlayerStock(player, resource)
		if playerStock < quantity {
			return TradeRecord{}, fmt.Errorf("insufficient stock (need %d, have %d)", quantity, playerStock)
		}
		removeFromPlayer(player, resource, quantity)
	}

	// All sells transfer resources to another player's planet (real economy).
	// Human sells prefer same system; AI sells go galaxy-wide.
	sellSystemID := -1
	localSell := false
	if player.IsHuman() && srcPlanet != nil {
		sellSystemID = te.getSystemForPlanet(srcPlanet)
		// Check if there's actually an NPC in this system to sell to
		if sellSystemID >= 0 {
			for _, p := range players {
				if p == nil || p == player {
					continue
				}
				for _, planet := range p.OwnedPlanets {
					if planet != nil && te.getSystemForPlanet(planet) == sellSystemID {
						localSell = true
						break
					}
				}
				if localSell {
					break
				}
			}
		}
		// Dynamic export fee (mirrors import fee logic)
		if !localSell {
			feeRate := 0.10
			rm := te.market.getResourceMarket(resource)
			if rm != nil && rm.TotalDemand > 0 {
				ratio := rm.TotalSupply / (rm.TotalDemand * 10)
				if ratio > 2.0 {
					feeRate = 0.05
				} else if ratio < 0.5 {
					feeRate = 0.20
				} else {
					feeRate = 0.15 - ratio*0.05
					if feeRate < 0.05 {
						feeRate = 0.05
					}
				}
			}
			exportFee := int(math.Round(float64(total) * feeRate))
			total -= exportFee
			if total < 1 {
				total = 1
			}
		}
	}
	te.addToOtherPlanet(players, player, resource, quantity, sellSystemID)

	// Credit seller
	player.Credits += total

	// Bump trade volume on market
	te.market.AddTradeVolume(resource, quantity, false)

	record := TradeRecord{
		Tick:      te.tick,
		Player:    player.Name,
		Resource:  resource,
		Quantity:  quantity,
		Action:    "sell",
		UnitPrice: price,
		Total:     total,
	}
	te.appendRecord(record)

	fmt.Printf("[Trade] %s sold %d %s @ %.0f = %d credits\n",
		player.Name, quantity, resource, price, total)

	return record, nil
}

// GetHistory returns the most recent N trade records.
func (te *TradeExecutor) GetHistory(limit int) []TradeRecord {
	te.mu.Lock()
	defer te.mu.Unlock()
	if limit <= 0 || limit > len(te.history) {
		limit = len(te.history)
	}
	start := len(te.history) - limit
	result := make([]TradeRecord, limit)
	copy(result, te.history[start:])
	return result
}

func (te *TradeExecutor) appendRecord(r TradeRecord) {
	te.history = append(te.history, r)
	if len(te.history) > 200 {
		te.history = te.history[len(te.history)-200:]
	}
}

// getSystemForPlanet returns the system ID containing the planet, or -1.
func (te *TradeExecutor) getSystemForPlanet(planet *entities.Planet) int {
	if te.systems == nil || planet == nil {
		return -1
	}
	for _, system := range te.systems {
		for _, entity := range system.Entities {
			if p, ok := entity.(*entities.Planet); ok && p.GetID() == planet.GetID() {
				return system.ID
			}
		}
	}
	return -1
}

// addToOtherPlanet adds resources to another player's planet, preferring same system.
func (te *TradeExecutor) addToOtherPlanet(players []*entities.Player, exclude *entities.Player, resource string, qty int, preferSystemID int) {
	// Try same system first
	if preferSystemID >= 0 && te.systems != nil {
		for _, p := range players {
			if p == nil || p == exclude {
				continue
			}
			for _, planet := range p.OwnedPlanets {
				if planet == nil {
					continue
				}
				if te.getSystemForPlanet(planet) == preferSystemID {
					planet.AddStoredResource(resource, qty)
					return
				}
			}
		}
	}
	// Fallback: any other player's planet with a trading post
	for _, p := range players {
		if p == nil || p == exclude {
			continue
		}
		dest := firstPlanetWithTradingPost(p)
		if dest != nil {
			dest.AddStoredResource(resource, qty)
			return
		}
	}
	// Last resort: any other player's planet
	for _, p := range players {
		if p == nil || p == exclude {
			continue
		}
		if len(p.OwnedPlanets) > 0 && p.OwnedPlanets[0] != nil {
			p.OwnedPlanets[0].AddStoredResource(resource, qty)
			return
		}
	}
}

// aggregateOtherStock sums stock of a resource across all players EXCEPT the given one.
func aggregateOtherStock(players []*entities.Player, exclude *entities.Player, resource string) int {
	total := 0
	for _, p := range players {
		if p == nil || p == exclude {
			continue
		}
		for _, planet := range p.OwnedPlanets {
			if planet == nil {
				continue
			}
			if s := planet.StoredResources[resource]; s != nil {
				total += s.Amount
			}
		}
	}
	return total
}

// aggregateOtherStockInSystem sums NPC stock only on planets in the same system.
func aggregateOtherStockInSystem(players []*entities.Player, exclude *entities.Player, resource string, systemID int, systems []*entities.System) int {
	planetIDs := getPlanetIDsInSystem(systemID, systems)
	total := 0
	for _, p := range players {
		if p == nil || p == exclude {
			continue
		}
		for _, planet := range p.OwnedPlanets {
			if planet == nil {
				continue
			}
			if !planetIDs[planet.GetID()] {
				continue
			}
			if s := planet.StoredResources[resource]; s != nil {
				total += s.Amount
			}
		}
	}
	return total
}

// removeFromOthers removes qty of resource from other players' planets.
func removeFromOthers(players []*entities.Player, exclude *entities.Player, resource string, qty int) {
	remaining := qty
	for _, p := range players {
		if p == nil || p == exclude || remaining <= 0 {
			continue
		}
		for _, planet := range p.OwnedPlanets {
			if planet == nil || remaining <= 0 {
				continue
			}
			if s := planet.StoredResources[resource]; s != nil && s.Amount > 0 {
				take := remaining
				if take > s.Amount {
					take = s.Amount
				}
				planet.RemoveStoredResource(resource, take)
				remaining -= take
			}
		}
	}
}

// removeFromOthersInSystem removes from NPC planets only in the specified system.
func removeFromOthersInSystem(players []*entities.Player, exclude *entities.Player, resource string, qty int, systemID int, systems []*entities.System) {
	planetIDs := getPlanetIDsInSystem(systemID, systems)
	remaining := qty
	for _, p := range players {
		if p == nil || p == exclude || remaining <= 0 {
			continue
		}
		for _, planet := range p.OwnedPlanets {
			if planet == nil || remaining <= 0 {
				continue
			}
			if !planetIDs[planet.GetID()] {
				continue
			}
			if s := planet.StoredResources[resource]; s != nil && s.Amount > 0 {
				take := remaining
				if take > s.Amount {
					take = s.Amount
				}
				planet.RemoveStoredResource(resource, take)
				remaining -= take
			}
		}
	}
}

func aggregatePlayerStock(player *entities.Player, resource string) int {
	total := 0
	for _, planet := range player.OwnedPlanets {
		if planet == nil {
			continue
		}
		if s := planet.StoredResources[resource]; s != nil {
			total += s.Amount
		}
	}
	return total
}

func removeFromPlayer(player *entities.Player, resource string, qty int) {
	remaining := qty
	for _, planet := range player.OwnedPlanets {
		if planet == nil || remaining <= 0 {
			continue
		}
		if s := planet.StoredResources[resource]; s != nil && s.Amount > 0 {
			take := remaining
			if take > s.Amount {
				take = s.Amount
			}
			planet.RemoveStoredResource(resource, take)
			remaining -= take
		}
	}
}

func firstPlanetWithTradingPost(player *entities.Player) *entities.Planet {
	if player == nil {
		return nil
	}
	for _, planet := range player.OwnedPlanets {
		if planet == nil {
			continue
		}
		for _, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok {
				if b.BuildingType == "Trading Post" && b.IsOperational {
					return planet
				}
			}
		}
	}
	if len(player.OwnedPlanets) > 0 {
		return player.OwnedPlanets[0]
	}
	return nil
}

// getPlanetIDsInSystem returns a set of planet IDs in the given system.
func getPlanetIDsInSystem(systemID int, systems []*entities.System) map[int]bool {
	ids := make(map[int]bool)
	for _, system := range systems {
		if system.ID == systemID {
			for _, entity := range system.Entities {
				if planet, ok := entity.(*entities.Planet); ok {
					ids[planet.GetID()] = true
				}
			}
			break
		}
	}
	return ids
}
