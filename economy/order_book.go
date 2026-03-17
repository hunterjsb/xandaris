package economy

import (
	"fmt"
	"sync"
)

// MarketOrder is a limit buy or sell order on a system's order book.
// Buy orders: "I'll pay up to MaxPrice for Quantity units"
// Sell orders: "I'll sell Quantity units at MinPrice or higher"
type MarketOrder struct {
	ID        int
	SystemID  int
	PlanetID  int    // planet where goods are delivered/collected
	Player    string
	Resource  string
	Action    string // "buy" or "sell"
	Quantity  int    // remaining unfilled quantity
	Price     int    // limit price (max for buy, min for sell)
	Filled    int    // how much has been filled so far
	Active    bool
}

// OrderBook manages per-system market orders.
type OrderBook struct {
	mu     sync.RWMutex
	orders []*MarketOrder
	nextID int
}

// NewOrderBook creates an empty order book.
func NewOrderBook() *OrderBook {
	return &OrderBook{
		orders: make([]*MarketOrder, 0),
		nextID: 1,
	}
}

// PlaceOrder adds a new limit order to the book.
func (ob *OrderBook) PlaceOrder(systemID, planetID int, player, resource, action string, quantity, price int) *MarketOrder {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order := &MarketOrder{
		ID:       ob.nextID,
		SystemID: systemID,
		PlanetID: planetID,
		Player:   player,
		Resource: resource,
		Action:   action,
		Quantity: quantity,
		Price:    price,
		Active:   true,
	}
	ob.nextID++
	ob.orders = append(ob.orders, order)

	fmt.Printf("[OrderBook] #%d: %s %s %d %s @ %dcr in SYS-%d\n",
		order.ID, player, action, quantity, resource, price, systemID+1)
	return order
}

// GetOrders returns active orders for a system, optionally filtered.
func (ob *OrderBook) GetOrders(systemID int, resource string) []*MarketOrder {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	var result []*MarketOrder
	for _, o := range ob.orders {
		if !o.Active || o.Quantity <= 0 {
			continue
		}
		if o.SystemID != systemID {
			continue
		}
		if resource != "" && o.Resource != resource {
			continue
		}
		result = append(result, o)
	}
	return result
}

// GetPlayerOrders returns all active orders for a player.
func (ob *OrderBook) GetPlayerOrders(player string) []*MarketOrder {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	var result []*MarketOrder
	for _, o := range ob.orders {
		if o.Active && o.Player == player {
			result = append(result, o)
		}
	}
	return result
}

// CancelOrder cancels an order by ID if owned by the player.
func (ob *OrderBook) CancelOrder(id int, player string) bool {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	for _, o := range ob.orders {
		if o.ID == id && o.Player == player {
			o.Active = false
			return true
		}
	}
	return false
}

// MatchOrders finds buy+sell pairs in a system that can be filled.
// Returns matched pairs. Caller handles resource/credit transfer.
type OrderMatch struct {
	BuyOrder  *MarketOrder
	SellOrder *MarketOrder
	Quantity  int
	Price     int // transaction price (average of buy/sell limits)
}

// FindMatches returns matchable buy+sell order pairs for a system.
func (ob *OrderBook) FindMatches(systemID int) []OrderMatch {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var matches []OrderMatch

	for _, buy := range ob.orders {
		if !buy.Active || buy.Action != "buy" || buy.SystemID != systemID || buy.Quantity <= 0 {
			continue
		}
		for _, sell := range ob.orders {
			if !sell.Active || sell.Action != "sell" || sell.SystemID != systemID || sell.Quantity <= 0 {
				continue
			}
			if buy.Resource != sell.Resource || buy.Player == sell.Player {
				continue
			}
			// Match: buy price >= sell price
			if buy.Price >= sell.Price {
				qty := buy.Quantity
				if qty > sell.Quantity {
					qty = sell.Quantity
				}
				price := (buy.Price + sell.Price) / 2 // split the difference

				matches = append(matches, OrderMatch{
					BuyOrder:  buy,
					SellOrder: sell,
					Quantity:  qty,
					Price:     price,
				})

				buy.Quantity -= qty
				buy.Filled += qty
				sell.Quantity -= qty
				sell.Filled += qty

				if buy.Quantity <= 0 {
					buy.Active = false
					break
				}
			}
		}
	}

	return matches
}

// GetAllOrders returns all orders (for save/load).
func (ob *OrderBook) GetAllOrders() []*MarketOrder {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	result := make([]*MarketOrder, len(ob.orders))
	copy(result, ob.orders)
	return result
}

// RestoreOrders loads orders from a save.
func (ob *OrderBook) RestoreOrders(orders []*MarketOrder) {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	ob.orders = orders
	for _, o := range orders {
		if o.ID >= ob.nextID {
			ob.nextID = o.ID + 1
		}
	}
}
