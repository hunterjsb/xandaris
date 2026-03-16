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
	Credits    *CreditLedger  // credit limit tracking between empires

	// Systems reference for system-scoped trading.
	// When set, human trades are scoped to the trading planet's system.
	systems        []*entities.System
	planetToSystem map[int]int        // planetID → systemID, cached
	systemPlanets  map[int]map[int]bool // systemID → set of planetIDs, cached
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
	te.rebuildPlanetIndex()
}

// rebuildPlanetIndex builds the planetID→systemID and systemID→planetIDs caches.
// Must be called with te.mu held.
func (te *TradeExecutor) rebuildPlanetIndex() {
	te.planetToSystem = make(map[int]int, len(te.systems)*4)
	te.systemPlanets = make(map[int]map[int]bool, len(te.systems))
	for _, system := range te.systems {
		ids := make(map[int]bool)
		for _, entity := range system.Entities {
			if planet, ok := entity.(*entities.Planet); ok {
				ids[planet.GetID()] = true
				te.planetToSystem[planet.GetID()] = system.ID
			}
		}
		te.systemPlanets[system.ID] = ids
	}
}

// getCachedSystemPlanets returns the planet ID set for a system, using the cache.
func (te *TradeExecutor) getCachedSystemPlanets(systemID int) map[int]bool {
	if te.systemPlanets != nil {
		if ids, ok := te.systemPlanets[systemID]; ok {
			return ids
		}
	}
	// Fallback: compute on the fly (shouldn't happen after SetSystems)
	return getPlanetIDsInSystem(systemID, te.systems)
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

// Buy purchases resources from other players' planets in the SAME system.
// Resources must physically exist on seller planets in your system — no teleportation.
// Requires a Trading Post on your planet. Resources arrive via local delivery (5-tick delay).
// For cross-system purchases, use cargo ships (load at source, fly, unload at dest).
func (te *TradeExecutor) Buy(player *entities.Player, players []*entities.Player, resource string, quantity int, tradingPlanet ...*entities.Planet) (TradeRecord, error) {
	if player == nil {
		return TradeRecord{}, fmt.Errorf("no player")
	}
	if quantity <= 0 {
		return TradeRecord{}, fmt.Errorf("invalid quantity")
	}

	te.mu.Lock()
	defer te.mu.Unlock()

	// Determine the trading planet first (needed for local price calc)
	var destPlanet *entities.Planet
	if len(tradingPlanet) > 0 && tradingPlanet[0] != nil {
		destPlanet = tradingPlanet[0]
	} else {
		destPlanet = firstPlanetWithTradingPost(player)
	}
	if destPlanet == nil {
		return TradeRecord{}, fmt.Errorf("build a Trading Post to access the market")
	}

	// Validate Trading Post
	tp := getTradingPost(destPlanet)
	if tp == nil {
		return TradeRecord{}, fmt.Errorf("build a Trading Post on %s to trade", destPlanet.Name)
	}

	// LOCAL ONLY: resources must come from other players' planets in THIS system
	systemID := te.getSystemForPlanet(destPlanet)
	if systemID < 0 {
		return TradeRecord{}, fmt.Errorf("planet not found in any system")
	}

	localStock := te.aggregateOtherStockInSystem(players, player, resource, systemID)
	if localStock < quantity {
		return TradeRecord{}, fmt.Errorf("insufficient local stock in system (need %d, available %d in this system — use cargo ships for cross-system trade)", quantity, localStock)
	}

	// Local price: base market price adjusted by local supply scarcity
	// More local stock = cheaper (buyer's market). Less = more expensive.
	basePrice := te.market.GetBuyPrice(resource)
	localPriceMult := LocalPriceMultiplier(localStock, quantity)
	price := basePrice * localPriceMult
	total := int(math.Round(price * float64(quantity)))
	if total <= 0 {
		total = quantity
	}

	if player.Credits < total {
		return TradeRecord{}, fmt.Errorf("insufficient credits (need %d, have %d)", total, player.Credits)
	}

	// Throughput check
	throughput := TradingPostThroughput(tp.Level)
	if throughput > 0 && quantity > throughput {
		return TradeRecord{}, fmt.Errorf("Trading Post throughput exceeded (max %d units, level %d)", throughput, tp.Level)
	}

	// TP processing fee
	tpFee := TradingPostFee(tp.Level)
	if tpFee > 0 {
		fee := int(math.Round(float64(total) * tpFee))
		total += fee
		if player.Credits < total {
			return TradeRecord{}, fmt.Errorf("insufficient credits with %.1f%% TP fee (need %d, have %d)", tpFee*100, total, player.Credits)
		}
	}

	// Execute: remove from local sellers, deduct credits, create local delivery
	te.removeFromOthersInSystem(players, player, resource, quantity, systemID)
	player.Credits -= total

	if te.Deliveries != nil {
		te.Deliveries.CreateLocalDelivery(
			te.tick, player.Name, "", resource, quantity,
			price, total, destPlanet.GetID(), systemID,
			DeliveryDirectionBuy, 5,
		)
	} else {
		destPlanet.AddStoredResource(resource, quantity)
	}

	// Credit tracking
	if te.Credits != nil {
		te.Credits.AddOutstanding(player.Name, "market", total)
	}

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

	fmt.Printf("[Trade] %s bought %d %s @ %.0f = %d credits (local, system %d)\n",
		player.Name, quantity, resource, price, total, systemID)

	return record, nil
}

// Sell sells resources to other players' planets in the SAME system.
// Resources are taken from your planet and delivered locally (5-tick delay).
// Credits are paid on delivery completion, not instantly.
// For cross-system sales, use cargo ships (load, fly, sell-at-dock).
func (te *TradeExecutor) Sell(player *entities.Player, players []*entities.Player, resource string, quantity int, tradingPlanet ...*entities.Planet) (TradeRecord, error) {
	if player == nil {
		return TradeRecord{}, fmt.Errorf("no player")
	}
	if quantity <= 0 {
		return TradeRecord{}, fmt.Errorf("invalid quantity")
	}

	te.mu.Lock()
	defer te.mu.Unlock()

	// Determine source planet
	var srcPlanet *entities.Planet
	if len(tradingPlanet) > 0 && tradingPlanet[0] != nil {
		srcPlanet = tradingPlanet[0]
	} else {
		srcPlanet = firstPlanetWithTradingPost(player)
	}
	if srcPlanet == nil {
		return TradeRecord{}, fmt.Errorf("build a Trading Post to sell on the market")
	}

	// Validate Trading Post
	tp := getTradingPost(srcPlanet)
	if tp == nil {
		return TradeRecord{}, fmt.Errorf("build a Trading Post on %s to trade", srcPlanet.Name)
	}

	// LOCAL ONLY: must have a buyer in this system
	sellSystemID := te.getSystemForPlanet(srcPlanet)

	// Local sell price: base market price adjusted by local demand
	// Selling into a system with LOW stock of this resource = higher price (scarcity premium)
	// Selling into a system already FLOODED = lower price
	basePrice := te.market.GetSellPrice(resource)
	localStock := te.aggregateOtherStockInSystem(players, player, resource, sellSystemID)
	localPriceMult := LocalSellPriceMultiplier(localStock)
	price := basePrice * localPriceMult
	total := int(math.Round(price * float64(quantity)))
	if total <= 0 {
		total = quantity
	}

	// Throughput check
	throughput := TradingPostThroughput(tp.Level)
	if throughput > 0 && quantity > throughput {
		return TradeRecord{}, fmt.Errorf("Trading Post throughput exceeded (max %d units, level %d)", throughput, tp.Level)
	}

	// TP fee (deducted from proceeds)
	tpFee := TradingPostFee(tp.Level)
	if tpFee > 0 {
		fee := int(math.Round(float64(total) * tpFee))
		total -= fee
		if total < 1 {
			total = 1
		}
	}

	// Check stock on source planet
	stored := srcPlanet.StoredResources[resource]
	planetStock := 0
	if stored != nil {
		planetStock = stored.Amount
	}
	if planetStock < quantity {
		return TradeRecord{}, fmt.Errorf("insufficient stock on %s (need %d, have %d)", srcPlanet.Name, quantity, planetStock)
	}

	if sellSystemID < 0 {
		return TradeRecord{}, fmt.Errorf("planet not found in any system")
	}

	// Find a buyer planet in this system
	destPlanetID := te.findBuyerPlanetID(players, player, sellSystemID)
	if destPlanetID == 0 {
		return TradeRecord{}, fmt.Errorf("no buyer in this system — use cargo ships to sell in other systems")
	}

	// Remove resources from source planet (into escrow)
	srcPlanet.RemoveStoredResource(resource, quantity)

	// Local delivery — credits paid on completion, not instantly
	if te.Deliveries != nil {
		te.Deliveries.CreateLocalDelivery(
			te.tick, "", player.Name, resource, quantity,
			price, total, destPlanetID, sellSystemID,
			DeliveryDirectionSell, 5,
		)
	} else {
		// No delivery manager fallback — instant (shouldn't happen in production)
		te.addToOtherPlanet(players, player, resource, quantity, sellSystemID)
		player.Credits += total
	}

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

	fmt.Printf("[Trade] %s sold %d %s @ %.0f = %d credits (local, system %d)\n",
		player.Name, quantity, resource, price, total, sellSystemID)

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
func (te *TradeExecutor) aggregateOtherStockInSystem(players []*entities.Player, exclude *entities.Player, resource string, systemID int) int {
	planetIDs := te.getCachedSystemPlanets(systemID)
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
func (te *TradeExecutor) removeFromOthersInSystem(players []*entities.Player, exclude *entities.Player, resource string, qty int, systemID int) {
	planetIDs := te.getCachedSystemPlanets(systemID)
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
				if b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
					return planet
				}
			}
		}
	}
	return nil // No Trading Post = no trading
}

// getTradingPost returns the Trading Post building on a planet, or nil.
func getTradingPost(planet *entities.Planet) *entities.Building {
	if planet == nil {
		return nil
	}
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok {
			if b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
				return b
			}
		}
	}
	return nil
}

// TradingPostThroughput returns the max units per trade interval for a TP level.
func TradingPostThroughput(level int) int {
	switch level {
	case 1:
		return 100
	case 2:
		return 250
	case 3:
		return 500
	case 4:
		return 1000
	default:
		return 0 // level 5 = unlimited
	}
}

// TradingPostFee returns the transaction fee rate for a TP level.
func TradingPostFee(level int) float64 {
	switch level {
	case 1:
		return 0.03
	case 2:
		return 0.02
	case 3:
		return 0.01
	case 4:
		return 0.005
	default:
		return 0.0 // level 5 = free
	}
}

// TradingPostCreditLimit returns the credit limit per empire for a TP level.
func TradingPostCreditLimit(level int) int {
	switch level {
	case 1:
		return 5000
	case 2:
		return 15000
	case 3:
		return 50000
	case 4:
		return 200000
	default:
		return 0 // level 5 = unlimited (0 means no limit)
	}
}

// LocalPriceMultiplier adjusts buy price based on local supply scarcity.
// High local stock = cheaper (0.5x at 1000+ units). Low stock = expensive (2.0x at <50 units).
func LocalPriceMultiplier(localStock, buyQuantity int) float64 {
	available := localStock - buyQuantity
	if available < 0 {
		available = 0
	}
	switch {
	case available > 1000:
		return 0.5 // flooded market, cheap
	case available > 500:
		return 0.7
	case available > 200:
		return 0.9
	case available > 100:
		return 1.0 // fair price
	case available > 50:
		return 1.3 // getting scarce
	case available > 10:
		return 1.6 // scarce
	default:
		return 2.0 // very scarce, premium price
	}
}

// LocalSellPriceMultiplier adjusts sell price based on how much local stock already exists.
// Selling into a market with LOW stock = premium (1.5x). HIGH stock = depressed (0.5x).
func LocalSellPriceMultiplier(localBuyerStock int) float64 {
	switch {
	case localBuyerStock > 1000:
		return 0.5 // buyers already have plenty, low demand
	case localBuyerStock > 500:
		return 0.7
	case localBuyerStock > 200:
		return 0.9
	case localBuyerStock > 100:
		return 1.0 // fair
	case localBuyerStock > 50:
		return 1.2 // buyers want this
	case localBuyerStock > 10:
		return 1.4
	default:
		return 1.5 // buyers desperately need this, premium
	}
}

// findBuyerPlanetID finds a planet in the given system owned by another player (for sell deliveries).
func (te *TradeExecutor) findBuyerPlanetID(players []*entities.Player, exclude *entities.Player, systemID int) int {
	planetIDs := te.getCachedSystemPlanets(systemID)
	for _, p := range players {
		if p == nil || p == exclude {
			continue
		}
		for _, planet := range p.OwnedPlanets {
			if planet != nil && planetIDs[planet.GetID()] {
				return planet.GetID()
			}
		}
	}
	return 0
}

// findSourceSystem finds a system with sufficient NPC stock of a resource.
func (te *TradeExecutor) findSourceSystem(players []*entities.Player, exclude *entities.Player, resource string, qty int) int {
	if te.systems == nil {
		return -1
	}
	for _, system := range te.systems {
		stock := te.aggregateOtherStockInSystem(players, exclude, resource, system.ID)
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
