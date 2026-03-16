package economy

import (
	"fmt"
	"sync"
)

// Delivery type constants.
const (
	DeliveryTypeLocal     = "local"      // same-system, no ship needed
	DeliveryTypeCargoShip = "cargo_ship" // cross-system, requires physical ship
)

// Delivery direction constants.
const (
	DeliveryDirectionBuy  = "buy"
	DeliveryDirectionSell = "sell"
)

// PendingDelivery represents an in-flight trade that requires physical cargo transport.
type PendingDelivery struct {
	ID               int
	Tick             int64
	BuyerName        string
	SellerName       string
	Resource         string
	Quantity         int
	UnitPrice        float64
	Total            int    // credits already deducted from buyer (buy) or value owed to seller (sell)
	DestPlanetID     int    // buyer's planet (dropoff)
	DestSystemID     int    // buyer's system
	SourceSystemID   int    // seller's system (pickup)
	SourcePlanetID   int    // seller's planet (for sell deliveries)
	ShipID           int    // cargo ship assigned (0 for local deliveries)
	Status           string // "in_transit", "delivered", "failed"
	DeliveryType     string // "local" or "cargo_ship"
	Direction        string // "buy" or "sell"
	EstimatedArrival int64  // tick when delivery should complete (for local deliveries)
}

// DeliveryManager tracks pending trade deliveries.
type DeliveryManager struct {
	mu         sync.RWMutex
	deliveries []*PendingDelivery
	nextID     int
}

// NewDeliveryManager creates a new delivery manager.
func NewDeliveryManager() *DeliveryManager {
	return &DeliveryManager{
		deliveries: make([]*PendingDelivery, 0),
		nextID:     1,
	}
}

// CreateDelivery registers a new pending delivery (cargo ship type).
func (dm *DeliveryManager) CreateDelivery(tick int64, buyer, seller, resource string, qty int, unitPrice float64, total int, destPlanetID, destSystemID, sourceSystemID, shipID int) *PendingDelivery {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	d := &PendingDelivery{
		ID:             dm.nextID,
		Tick:           tick,
		BuyerName:      buyer,
		SellerName:     seller,
		Resource:       resource,
		Quantity:       qty,
		UnitPrice:      unitPrice,
		Total:          total,
		DestPlanetID:   destPlanetID,
		DestSystemID:   destSystemID,
		SourceSystemID: sourceSystemID,
		ShipID:         shipID,
		Status:         "in_transit",
		DeliveryType:   DeliveryTypeCargoShip,
		Direction:      DeliveryDirectionBuy,
	}
	dm.nextID++
	dm.deliveries = append(dm.deliveries, d)

	fmt.Printf("[Delivery] #%d: %s -> %s, %d %s via ship %d\n", d.ID, seller, buyer, qty, resource, shipID)
	return d
}

// CreateLocalDelivery registers a same-system delivery that completes after a delay (no ship needed).
func (dm *DeliveryManager) CreateLocalDelivery(tick int64, buyer, seller, resource string, qty int, unitPrice float64, total int, destPlanetID, systemID int, direction string, delayTicks int64) *PendingDelivery {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	d := &PendingDelivery{
		ID:               dm.nextID,
		Tick:             tick,
		BuyerName:        buyer,
		SellerName:       seller,
		Resource:         resource,
		Quantity:         qty,
		UnitPrice:        unitPrice,
		Total:            total,
		DestPlanetID:     destPlanetID,
		DestSystemID:     systemID,
		SourceSystemID:   systemID,
		Status:           "in_transit",
		DeliveryType:     DeliveryTypeLocal,
		Direction:        direction,
		EstimatedArrival: tick + delayTicks,
	}
	dm.nextID++
	dm.deliveries = append(dm.deliveries, d)

	fmt.Printf("[Delivery] #%d (local): %s -> %s, %d %s, arrives tick %d\n",
		d.ID, seller, buyer, qty, resource, d.EstimatedArrival)
	return d
}

// GetDeliveriesForShip returns all active deliveries assigned to a ship.
func (dm *DeliveryManager) GetDeliveriesForShip(shipID int) []*PendingDelivery {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var result []*PendingDelivery
	for _, d := range dm.deliveries {
		if d.ShipID == shipID && d.Status == "in_transit" {
			result = append(result, d)
		}
	}
	return result
}

// CompleteDelivery marks a delivery as completed.
func (dm *DeliveryManager) CompleteDelivery(deliveryID int) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	for _, d := range dm.deliveries {
		if d.ID == deliveryID {
			d.Status = "delivered"
			fmt.Printf("[Delivery] #%d completed: %d %s delivered to %s\n", d.ID, d.Quantity, d.Resource, d.BuyerName)
			return
		}
	}
}

// FailDelivery marks a delivery as failed and returns the credit amount to refund.
func (dm *DeliveryManager) FailDelivery(deliveryID int) (buyerName string, refundAmount int) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	for _, d := range dm.deliveries {
		if d.ID == deliveryID {
			d.Status = "failed"
			fmt.Printf("[Delivery] #%d FAILED: %d %s lost, refunding %d to %s\n", d.ID, d.Quantity, d.Resource, d.Total, d.BuyerName)
			return d.BuyerName, d.Total
		}
	}
	return "", 0
}

// GetActiveDeliveries returns all in-transit deliveries.
func (dm *DeliveryManager) GetActiveDeliveries() []*PendingDelivery {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var result []*PendingDelivery
	for _, d := range dm.deliveries {
		if d.Status == "in_transit" {
			result = append(result, d)
		}
	}
	return result
}

// GetAllDeliveries returns all deliveries (for save/load).
func (dm *DeliveryManager) GetAllDeliveries() []*PendingDelivery {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return append([]*PendingDelivery{}, dm.deliveries...)
}

// RestoreDeliveries loads deliveries from a save (for save/load).
func (dm *DeliveryManager) RestoreDeliveries(deliveries []*PendingDelivery) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.deliveries = deliveries
	for _, d := range deliveries {
		if d.ID >= dm.nextID {
			dm.nextID = d.ID + 1
		}
	}
}
