package economy

import (
	"fmt"
	"sync"
)

// TradeContract is a binding supply agreement between two factions.
// The supplier commits to delivering a resource, the buyer commits to paying.
// Contracts auto-execute via the local exchange or cargo ship delivery.
type TradeContract struct {
	ID           int
	Supplier     string // faction that delivers resources
	Buyer        string // faction that pays credits
	Resource     string
	Quantity     int     // units per delivery
	PricePerUnit int     // agreed price (locked in)
	Interval     int     // ticks between deliveries
	SystemID     int     // system where delivery happens
	PlanetID     int     // buyer's planet (delivery destination)
	TicksLeft    int     // ticks until next delivery
	Deliveries   int     // completed deliveries count
	Active       bool
}

// ContractManager tracks active trade contracts between factions.
type ContractManager struct {
	mu        sync.RWMutex
	contracts []*TradeContract
	nextID    int
}

// NewContractManager creates a new contract manager.
func NewContractManager() *ContractManager {
	return &ContractManager{
		contracts: make([]*TradeContract, 0),
		nextID:    1,
	}
}

// CreateContract creates a new trade contract. Both parties must agree.
func (cm *ContractManager) CreateContract(supplier, buyer, resource string, qty, pricePerUnit, interval, systemID, planetID int) *TradeContract {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c := &TradeContract{
		ID:           cm.nextID,
		Supplier:     supplier,
		Buyer:        buyer,
		Resource:     resource,
		Quantity:     qty,
		PricePerUnit: pricePerUnit,
		Interval:     interval,
		SystemID:     systemID,
		PlanetID:     planetID,
		TicksLeft:    interval,
		Active:       true,
	}
	cm.nextID++
	cm.contracts = append(cm.contracts, c)

	fmt.Printf("[Contract] #%d: %s supplies %d %s to %s @ %dcr/unit every %d ticks\n",
		c.ID, supplier, qty, resource, buyer, pricePerUnit, interval)
	return c
}

// GetActiveContracts returns active contracts, optionally filtered by player.
func (cm *ContractManager) GetActiveContracts(player string) []*TradeContract {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var result []*TradeContract
	for _, c := range cm.contracts {
		if !c.Active {
			continue
		}
		if player == "" || c.Supplier == player || c.Buyer == player {
			result = append(result, c)
		}
	}
	return result
}

// GetAllContracts returns all contracts (for save/load persistence).
func (cm *ContractManager) GetAllContracts() []*TradeContract {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	result := make([]*TradeContract, len(cm.contracts))
	copy(result, cm.contracts)
	return result
}

// CancelContract cancels a contract (either party can cancel).
func (cm *ContractManager) CancelContract(id int, player string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, c := range cm.contracts {
		if c.ID == id && (c.Supplier == player || c.Buyer == player) {
			c.Active = false
			return true
		}
	}
	return false
}

// TickContracts decrements timers and returns contracts ready for execution.
func (cm *ContractManager) TickContracts() []*TradeContract {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var ready []*TradeContract
	for _, c := range cm.contracts {
		if !c.Active {
			continue
		}
		c.TicksLeft--
		if c.TicksLeft <= 0 {
			ready = append(ready, c)
			c.TicksLeft = c.Interval // reset timer
		}
	}
	return ready
}

// CompleteDelivery increments the delivery counter.
func (cm *ContractManager) CompleteDelivery(id int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for _, c := range cm.contracts {
		if c.ID == id {
			c.Deliveries++
			return
		}
	}
}
