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

// TradeCallback is called after every successful trade (for event logging).
type TradeCallback func(record TradeRecord)

// ShipDispatcher provides cargo ship operations for cross-system trade.
type ShipDispatcher interface {
	FindAvailableCargoShip(owner string, systemID int) *entities.Ship
	DispatchShipToSystem(ship *entities.Ship, targetSystemID int) bool
	AreSystemsConnected(fromID, toID int) bool
	FindPath(fromID, toID int) []int
}

// TradeExecutor provides the single code path for all trades.
// Both the UI and API call into this to execute trades.
type TradeExecutor struct {
	market     *Market
	history    []TradeRecord
	mu         sync.Mutex
	tick       int64
	OnTrade    TradeCallback // optional callback for event logging
	Deliveries *DeliveryManager
	Dispatcher ShipDispatcher // for cross-system cargo ship dispatch

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

	// Cross-system trade requires cargo ship delivery
	if !useSystemScope {
		// Check if dispatcher is available
		if te.Dispatcher == nil || te.Deliveries == nil {
			// Fallback: allow instant trade (no logistics system wired)
			feeRate := 0.10
			if rm := te.market.getResourceMarket(resource); rm != nil {
				feeRate = ComputeImportFee(rm.TotalSupply, rm.TotalDemand)
			}
			importFee := int(math.Round(float64(total) * feeRate))
			total += importFee
			if player.Credits < total {
				return TradeRecord{}, fmt.Errorf("insufficient credits with import fee (need %d, have %d)", total, player.Credits)
			}
			removeFromOthers(players, player, resource, quantity)
			destPlanet.AddStoredResource(resource, quantity)
			player.Credits -= total
		} else {
			// Find a source system with stock
			sourceSystemID := te.findSourceSystem(players, player, resource, quantity)
			if sourceSystemID < 0 {
				return TradeRecord{}, fmt.Errorf("no system has %d %s available", quantity, resource)
			}

			// Check hyperlane connectivity
			if !te.Dispatcher.AreSystemsConnected(systemID, sourceSystemID) {
				return TradeRecord{}, fmt.Errorf("no trade route to %s supply (systems not connected)", resource)
			}

			// Apply distance-based import fee
			path := te.Dispatcher.FindPath(sourceSystemID, systemID)
			hops := len(path)
			feeRate := 0.05 * float64(hops) // 5% per hop
			if feeRate > 0.30 {
				feeRate = 0.30
			}
			importFee := int(math.Round(float64(total) * feeRate))
			total += importFee
			if player.Credits < total {
				return TradeRecord{}, fmt.Errorf("insufficient credits with import fee (need %d, have %d)", total, player.Credits)
			}

			// Find available cargo ship
			ship := te.Dispatcher.FindAvailableCargoShip(player.Name, systemID)
			if ship == nil {
				// Also check source system
				ship = te.Dispatcher.FindAvailableCargoShip(player.Name, sourceSystemID)
			}
			if ship == nil {
				return TradeRecord{}, fmt.Errorf("no cargo ship available for cross-system trade (build one at a Shipyard)")
			}

			// Check cargo capacity
			totalCargo := ship.GetTotalCargo()
			if totalCargo+quantity > ship.MaxCargo {
				avail := ship.MaxCargo - totalCargo
				return TradeRecord{}, fmt.Errorf("cargo ship only has %d/%d space (need %d)", avail, ship.MaxCargo, quantity)
			}

			// Execute: deduct credits, remove from seller, load cargo, dispatch
			player.Credits -= total
			removeFromOthersInSystem(players, player, resource, quantity, sourceSystemID, te.systems)
			ship.CargoHold[resource] += quantity

			// Set up delivery route
			if ship.CurrentSystem == systemID {
				// Ship is at destination — route: go to source, pick up (already loaded), come back
				routePath := te.Dispatcher.FindPath(systemID, sourceSystemID)
				returnPath := te.Dispatcher.FindPath(sourceSystemID, systemID)
				ship.RoutePath = append(routePath, returnPath...)
			} else {
				// Ship is elsewhere — route to destination
				ship.RoutePath = te.Dispatcher.FindPath(ship.CurrentSystem, systemID)
			}

			delivery := te.Deliveries.CreateDelivery(te.tick, player.Name, "", resource, quantity, price, total, destPlanet.GetID(), systemID, sourceSystemID, ship.GetID())
			ship.DeliveryID = delivery.ID

			// Dispatch ship to first hop
			if len(ship.RoutePath) > 0 {
				te.Dispatcher.DispatchShipToSystem(ship, ship.RoutePath[0])
			}
		}
	} else {
		// Same-system trade — instant, no fee
		removeFromOthersInSystem(players, player, resource, quantity, systemID, te.systems)
		destPlanet.AddStoredResource(resource, quantity)
		player.Credits -= total
	}

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
	if te.OnTrade != nil {
		te.OnTrade(record)
	}

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
		// Dynamic export fee
		if !localSell {
			feeRate := 0.10
			if rm := te.market.getResourceMarket(resource); rm != nil {
				feeRate = ComputeImportFee(rm.TotalSupply, rm.TotalDemand)
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
	if te.OnTrade != nil {
		te.OnTrade(record)
	}

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

// findSourceSystem finds a system with sufficient NPC stock of a resource.
func (te *TradeExecutor) findSourceSystem(players []*entities.Player, exclude *entities.Player, resource string, qty int) int {
	if te.systems == nil {
		return -1
	}
	for _, system := range te.systems {
		stock := aggregateOtherStockInSystem(players, exclude, resource, system.ID, te.systems)
		if stock >= qty {
			return system.ID
		}
	}
	return -1
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
