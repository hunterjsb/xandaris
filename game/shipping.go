package game

import (
	"fmt"
	"sync"
)

// ShippingRoute defines a recurring cargo route between two planets.
// Ships assigned to routes automatically load/travel/unload/return.
type ShippingRoute struct {
	ID            int    `json:"id"`
	Owner         string `json:"owner"`
	SourcePlanet  int    `json:"source_planet"`  // planet to load from
	DestPlanet    int    `json:"dest_planet"`    // planet to deliver to
	Resource      string `json:"resource"`       // what to carry ("Iron", "Fuel", "Colonists", etc)
	Quantity      int    `json:"quantity"`        // how much per trip (0 = fill cargo hold)
	ShipID        int    `json:"ship_id"`         // assigned ship (0 = auto-assign)
	Active        bool   `json:"active"`
	TripsComplete int    `json:"trips_complete"` // lifetime counter
}

// ShippingManager manages all active shipping routes.
type ShippingManager struct {
	mu     sync.RWMutex
	routes []*ShippingRoute
	nextID int
}

// NewShippingManager creates a new shipping manager.
func NewShippingManager() *ShippingManager {
	return &ShippingManager{
		routes: make([]*ShippingRoute, 0),
		nextID: 1,
	}
}

// CreateRoute adds a new shipping route and returns its ID.
func (sm *ShippingManager) CreateRoute(owner string, sourcePlanet, destPlanet int, resource string, quantity, shipID int) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	route := &ShippingRoute{
		ID:           sm.nextID,
		Owner:        owner,
		SourcePlanet: sourcePlanet,
		DestPlanet:   destPlanet,
		Resource:     resource,
		Quantity:     quantity,
		ShipID:       shipID,
		Active:       true,
	}
	sm.nextID++
	sm.routes = append(sm.routes, route)

	fmt.Printf("[Shipping] Route #%d: %s %s from planet %d → %d (ship %d)\n",
		route.ID, resource, owner, sourcePlanet, destPlanet, shipID)
	return route.ID
}

// CancelRoute deactivates a route by ID.
func (sm *ShippingManager) CancelRoute(id int) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for _, r := range sm.routes {
		if r.ID == id {
			r.Active = false
			return true
		}
	}
	return false
}

// GetRoutes returns routes for a player (empty = all).
func (sm *ShippingManager) GetRoutes(owner string) []*ShippingRoute {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	var result []*ShippingRoute
	for _, r := range sm.routes {
		if !r.Active {
			continue
		}
		if owner == "" || r.Owner == owner {
			result = append(result, r)
		}
	}
	return result
}

// GetRouteForShip returns the active route assigned to a ship, if any.
func (sm *ShippingManager) GetRouteForShip(shipID int) *ShippingRoute {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	for _, r := range sm.routes {
		if r.Active && r.ShipID == shipID {
			return r
		}
	}
	return nil
}

// GetRoutesForPlanet returns routes that source from or deliver to a planet.
func (sm *ShippingManager) GetRoutesForPlanet(planetID int) []*ShippingRoute {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	var result []*ShippingRoute
	for _, r := range sm.routes {
		if r.Active && (r.SourcePlanet == planetID || r.DestPlanet == planetID) {
			result = append(result, r)
		}
	}
	return result
}

// CompleteTrip increments the trip counter for a route.
func (sm *ShippingManager) CompleteTrip(routeID int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for _, r := range sm.routes {
		if r.ID == routeID {
			r.TripsComplete++
			return
		}
	}
}

// AssignShip sets the ship ID for a route.
func (sm *ShippingManager) AssignShip(routeID, shipID int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for _, r := range sm.routes {
		if r.ID == routeID {
			r.ShipID = shipID
			return
		}
	}
}

// GetAllRoutes returns all routes (for save/load).
func (sm *ShippingManager) GetAllRoutes() []*ShippingRoute {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return append([]*ShippingRoute{}, sm.routes...)
}
