package tick

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase"

	"github.com/hunterjsb/xandaris/internal/economy"
	"github.com/hunterjsb/xandaris/internal/websocket"
)

var (
	currentTick   int64 = 1
	tickMutex     sync.RWMutex
	tickRate      int   = 6 // ticks per minute (10 seconds per tick)
	processingTick bool = false
)

// StartContinuousProcessor starts the continuous game tick processing
func StartContinuousProcessor(app *pocketbase.PocketBase) {
	log.Println("Starting continuous game tick processor...")
	tickInterval := time.Duration(60/tickRate) * time.Second
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := ProcessTick(app); err != nil {
				log.Printf("Error processing tick: %v", err)
			}
		}
	}
}

// GetCurrentTick returns the current game tick
func GetCurrentTick(app *pocketbase.PocketBase) int64 {
	tickMutex.RLock()
	defer tickMutex.RUnlock()
	return currentTick
}

// GetTickRate returns the current tick rate (ticks per minute)
func GetTickRate() int {
	return tickRate
}

// ProcessTick handles a single game tick processing
func ProcessTick(app *pocketbase.PocketBase) error {
	tickMutex.Lock()
	if processingTick {
		tickMutex.Unlock()
		return nil // Skip if already processing
	}
	processingTick = true
	currentTick++
	tick := currentTick
	tickMutex.Unlock()

	defer func() {
		tickMutex.Lock()
		processingTick = false
		tickMutex.Unlock()
	}()

	log.Printf("Processing game tick #%d...", tick)
	startTime := time.Now()

	// 1. Process Pending Fleet Orders
	if err := ProcessPendingFleetOrders(app, tick); err != nil {
		log.Printf("Error processing pending fleet orders: %v", err)
		// Decide if this error is critical enough to stop the entire tick.
		// For now, we'll log and continue, but this could be returned.
	}

	// 2. Update markets and economy
	if err := economy.UpdateMarkets(app); err != nil {
		log.Printf("Error updating markets: %v", err)
		return fmt.Errorf("market update failed: %w", err)
	}

	// 3. Move cargo on trade routes
	if err := MoveCargo(app); err != nil {
		log.Printf("Error moving cargo: %v", err)
		return fmt.Errorf("cargo movement failed: %w", err)
	}

	// 4. Evaluate treaties
	if err := EvaluateTreaties(app); err != nil {
		log.Printf("Error evaluating treaties: %v", err)
		return fmt.Errorf("treaty evaluation failed: %w", err)
	}

	// 5. Broadcast tick completion via WebSocket
	websocket.BroadcastTickUpdate(int(tick), "")

	duration := time.Since(startTime)
	log.Printf("Game tick #%d completed in %v", tick, duration)
	return nil
}

// MoveCargo handles trade route cargo movement
func MoveCargo(app *pocketbase.PocketBase) error {
	// Get all trade routes that should arrive this tick
	tickMutex.RLock()
	tick := currentTick
	tickMutex.RUnlock()
	
	routes, err := app.Dao().FindRecordsByExpr("trade_routes", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch trade routes: %w", err)
	}

	for _, route := range routes {

		// Move cargo from source to destination
		fromId := route.GetString("from_system")
		toId := route.GetString("to_system")
		resourceType := route.GetString("resource_type")

		// For now, just log the trade route activity
		log.Printf("Trade route %s: moving %s from %s to %s", route.Id, resourceType, fromId, toId)

		// Calculate next ETA (trade routes are recurring)
		// Travel time is 12 ticks (2 minutes at 6 ticks/minute)
		travelTime := int64(12)
		nextETATick := tick + travelTime
		route.Set("eta_tick", nextETATick)

		if err := app.Dao().SaveRecord(route); err != nil {
			log.Printf("Failed to update trade route %s: %v", route.Id, err)
		}
	}

	log.Printf("Processed %d trade routes", len(routes))
	return nil
}

// transferCargo moves resources between systems
func transferCargo(app *pocketbase.PocketBase, fromId, toId, resourceType string, capacity int) error {
	// TODO: Implement proper cargo transfer with new schema
	// This would involve:
	// 1. Get owner of trade route
	// 2. Check their resource inventory
	// 3. Move resources between their global inventory
	// 4. Handle fleet/cargo ship logistics
	log.Printf("Trade route cargo transfer not fully implemented yet")
	return nil
}


// EvaluateTreaties checks for expired treaties and updates statuses
func EvaluateTreaties(app *pocketbase.PocketBase) error {
	// Check if treaties collection exists
	_, err := app.Dao().FindCollectionByNameOrId("treaties")
	if err != nil {
		// Treaties collection doesn't exist, skip
		return nil
	}

	// Get all active treaties
	treaties, err := app.Dao().FindRecordsByExpr("treaties", nil, nil)
	if err != nil {
		// No treaties found, this is normal
		log.Printf("No treaties found, skipping evaluation")
		return nil
	}

	now := time.Now()
	expired := 0

	for _, treaty := range treaties {
		expiresAt := treaty.GetDateTime("expires_at")
		if !expiresAt.IsZero() && expiresAt.Time().Before(now) {
			treaty.Set("status", "expired")
			if err := app.Dao().SaveRecord(treaty); err != nil {
				log.Printf("Failed to expire treaty %s: %v", treaty.Id, err)
				continue
			}
			expired++
		}
	}

	if expired > 0 {
		log.Printf("Expired %d treaties", expired)
	}
	return nil
}

// ProcessPendingFleetOrders fetches and processes fleet_orders that are due.
func ProcessPendingFleetOrders(app *pocketbase.PocketBase, currentTick int64) error {
	log.Printf("Starting ProcessPendingFleetOrders for tick #%d", currentTick)

	_, err := app.Dao().FindCollectionByNameOrId("fleet_orders")
	if err != nil {
		return fmt.Errorf("failed to find 'fleet_orders' collection: %w", err)
	}

	sort := "execute_at_tick ASC, created ASC"
	
	records, err := app.Dao().FindRecordsByFilter(
		"fleet_orders",
		fmt.Sprintf("status = 'pending' && execute_at_tick <= %d", currentTick),
		sort, 
		0,    
		0,    
	)
	if err != nil {
		return fmt.Errorf("failed to fetch pending fleet orders: %w", err)
	}

	if len(records) == 0 {
		log.Printf("No pending fleet orders to process for tick #%d", currentTick)
		return nil
	}

	log.Printf("Found %d pending fleet orders to process for tick #%d", len(records), currentTick)

	_, err = app.Dao().FindCollectionByNameOrId("fleets")
	if err != nil {
		// If fleets collection isn't found, we can't process any fleet orders.
		return fmt.Errorf("failed to find 'fleets' collection: %w", err)
	}

	for _, order := range records {
		orderType := order.GetString("type") // Should always be "move" for fleet_orders
		log.Printf("Processing fleet order %s of type %s", order.Id, orderType)

		// Atomically mark as "processing"
		originalStatus := order.GetString("status")
		order.Set("status", "processing")
		if err := app.Dao().SaveRecord(order); err != nil {
			log.Printf("Error marking fleet order %s as processing: %v. Original status: %s. Skipping.", order.Id, err, originalStatus)
			order.Set("status", originalStatus) 
			continue 
		}

		var processingError error
		finalStatus := "completed" // Assume success

		if orderType == "move" {
			fleetID := order.GetString("fleet_id")
			if fleetID == "" {
				processingError = fmt.Errorf("missing fleet_id in fleet order %s", order.Id)
				finalStatus = "failed"
			} else {
				destinationSystemID := order.GetString("destination_system_id")
				if destinationSystemID == "" {
					processingError = fmt.Errorf("missing destination_system_id in fleet order %s", order.Id)
					finalStatus = "failed"
				} else {
					fleet, err := app.Dao().FindRecordById("fleets", fleetID)
					if err != nil {
						processingError = fmt.Errorf("fleet %s not found for order %s: %w", fleetID, order.Id, err)
						finalStatus = "failed"
					} else {
						// Successfully fetched fleet and destination
						fleet.Set("current_system", destinationSystemID)

						if err := app.Dao().SaveRecord(fleet); err != nil {
							processingError = fmt.Errorf("failed to save fleet %s for order %s: %w", fleetID, order.Id, err)
							finalStatus = "failed"
						} else {
							log.Printf("Fleet %s moved to system %s successfully for order %s.", fleetID, destinationSystemID, order.Id)
						}
					}
				}
			}
		} else {
			log.Printf("Unknown order type '%s' in fleet_orders collection for order ID %s. Marking as failed.", orderType, order.Id)
			processingError = fmt.Errorf("unknown order type in fleet_orders: %s", orderType)
			finalStatus = "failed"
		}

		// Update order status and save
		order.Set("status", finalStatus)
		if processingError != nil {
			log.Printf("Fleet order %s failed: %v", order.Id, processingError)
			// Error details are logged, status is set to "failed" - that's sufficient
		}

		if err := app.Dao().SaveRecord(order); err != nil {
			log.Printf("CRITICAL: Failed to save final status for fleet order %s (%s): %v. Data may be inconsistent.", order.Id, finalStatus, err)
		} else {
			log.Printf("Fleet order %s successfully updated to status '%s'.", order.Id, finalStatus)
		}
	}

	log.Printf("Finished ProcessPendingFleetOrders for tick #%d. Processed %d fleet orders.", currentTick, len(records))
	return nil
}