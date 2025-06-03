package tick

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"

	"github.com/hunterjsb/xandaris/internal/economy"
	"github.com/hunterjsb/xandaris/internal/websocket"
)

var (
	currentTick   int64 = 1
	tickMutex     sync.RWMutex
	tickRate      int   = 60 // ticks per minute (1 second per tick)
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

	sort := "execute_at_tick,created"
	
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
						// Move fleet to destination
						fleet.Set("current_system", destinationSystemID)

						if err := app.Dao().SaveRecord(fleet); err != nil {
							processingError = fmt.Errorf("failed to save fleet %s for order %s: %w", fleetID, order.Id, err)
							finalStatus = "failed"
						} else {
							log.Printf("Fleet %s moved to system %s successfully for order %s.", fleetID, destinationSystemID, order.Id)
							
							// Check if this is part of a multi-hop route
							routePath := order.Get("route_path")
							currentHop := order.GetInt("current_hop")
							finalDestinationID := order.GetString("final_destination_id")
							
							log.Printf("DEBUG: Multi-hop check for order %s: routePath=%v, currentHop=%d, finalDest=%s", 
								order.Id, routePath, currentHop, finalDestinationID)
							
							if routePath != nil {
								// Multi-hop route - check if we need to continue
								var routePathSlice []string
								var ok bool
								
								// Handle different JSON formats from PocketBase
								switch v := routePath.(type) {
								case []interface{}:
									routePathSlice = make([]string, len(v))
									for i, item := range v {
										if str, ok := item.(string); ok {
											routePathSlice[i] = str
										} else {
											log.Printf("ERROR: Non-string item in route_path: %v", item)
											ok = false
											break
										}
									}
									ok = true
								case []string:
									routePathSlice = v
									ok = true
								case string:
									// If it's a JSON string, try to parse it
									log.Printf("DEBUG: Route path is string, attempting JSON parse: %s", v)
									if err := json.Unmarshal([]byte(v), &routePathSlice); err != nil {
										log.Printf("ERROR: Failed to parse route_path JSON string: %v", err)
										ok = false
									} else {
										ok = true
									}
								default:
									// Handle PocketBase JsonRaw type
									log.Printf("DEBUG: Attempting to unmarshal JsonRaw type: %T", v)
									if jsonBytes, err := v.(interface{ MarshalJSON() ([]byte, error) }).MarshalJSON(); err == nil {
										if json.Unmarshal(jsonBytes, &routePathSlice) == nil {
											ok = true
											log.Printf("DEBUG: Successfully parsed JsonRaw: %v", routePathSlice)
										} else {
											log.Printf("DEBUG: Failed to unmarshal JsonRaw to []string")
											ok = false
										}
									} else {
										log.Printf("DEBUG: Failed to marshal JsonRaw: %v", err)
										ok = false
									}
								}
								
								log.Printf("DEBUG: Route path slice conversion: ok=%t, len=%d, path=%v", ok, len(routePathSlice), routePathSlice)
								if ok && len(routePathSlice) > 0 && finalDestinationID != "" {
									nextHop := currentHop + 1
									
									log.Printf("DEBUG: Route progression check: nextHop=%d, totalSystems=%d", nextHop, len(routePathSlice))
									// Check if there are more hops remaining
									if nextHop < len(routePathSlice)-1 {
										// Create next hop order
										nextSystemId := routePathSlice[nextHop+1]
										
										log.Printf("DEBUG: Creating next hop order: nextSystemId=%s", nextSystemId)
										
										// Calculate execute time for next hop
										nextExecuteAtTick := GetCurrentTick(app) + int64(2) // 2 ticks per hop
										
										// Create new order for next hop
										fleetOrdersCollection, err := app.Dao().FindCollectionByNameOrId("fleet_orders")
										if err == nil {
											nextOrder := models.NewRecord(fleetOrdersCollection)
											nextOrder.Set("user_id", order.GetString("user_id"))
											nextOrder.Set("fleet_id", fleetID)
											nextOrder.Set("type", "move")
											nextOrder.Set("status", "pending")
											nextOrder.Set("execute_at_tick", nextExecuteAtTick)
											nextOrder.Set("destination_system_id", nextSystemId)
											nextOrder.Set("original_system_id", destinationSystemID)
											nextOrder.Set("travel_time_ticks", 2)
											// Preserve the original route path as []string to ensure consistency
											nextOrder.Set("route_path", routePathSlice)
											nextOrder.Set("current_hop", nextHop)
											nextOrder.Set("final_destination_id", finalDestinationID)
											
											if err := app.Dao().SaveRecord(nextOrder); err != nil {
												log.Printf("ERROR: Failed to create next hop order for fleet %s: %v", fleetID, err)
											} else {
												log.Printf("SUCCESS: Created next hop order for fleet %s: hop %d/%d to system %s", 
													fleetID, nextHop+1, len(routePathSlice)-1, nextSystemId)
											}
										} else {
											log.Printf("ERROR: Failed to find fleet_orders collection: %v", err)
										}
									} else {
										log.Printf("SUCCESS: Fleet %s completed multi-hop route to final destination %s", fleetID, finalDestinationID)
									}
								} else {
									log.Printf("DEBUG: Multi-hop route check failed: ok=%t, len=%d, finalDest='%s'", ok, len(routePathSlice), finalDestinationID)
								}
							} else {
								log.Printf("DEBUG: No route path found for order %s", order.Id)
							}
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