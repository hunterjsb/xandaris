package tick

import (
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

	// 1. Update markets and economy
	if err := economy.UpdateMarkets(app); err != nil {
		log.Printf("Error updating markets: %v", err)
		return fmt.Errorf("market update failed: %w", err)
	}

	// 2. Apply building completions
	if err := ApplyBuildingCompletions(app); err != nil {
		log.Printf("Error applying building completions: %v", err)
		return fmt.Errorf("building completion failed: %w", err)
	}

	// 3. Move cargo on trade routes
	if err := MoveCargo(app); err != nil {
		log.Printf("Error moving cargo: %v", err)
		return fmt.Errorf("cargo movement failed: %w", err)
	}

	// 4. Resolve fleet arrivals
	if err := ResolveFleetArrivals(app); err != nil {
		log.Printf("Error resolving fleet arrivals: %v", err)
		return fmt.Errorf("fleet resolution failed: %w", err)
	}

	// 5. Evaluate treaties
	if err := EvaluateTreaties(app); err != nil {
		log.Printf("Error evaluating treaties: %v", err)
		return fmt.Errorf("treaty evaluation failed: %w", err)
	}

	// 6. Broadcast tick completion via WebSocket
	websocket.BroadcastTickUpdate(int(tick), "")

	duration := time.Since(startTime)
	log.Printf("Game tick #%d completed in %v", tick, duration)
	return nil
}

// ApplyBuildingCompletions checks for completed buildings and applies them
func ApplyBuildingCompletions(app *pocketbase.PocketBase) error {
	// TODO: Implement building queue system
	// For now, buildings complete instantly when queued
	log.Println("Applied building completions")
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


// ResolveFleetArrivals handles fleet arrivals and combat
func ResolveFleetArrivals(app *pocketbase.PocketBase) error {
	// Get all fleets that have an ETA (including past due)
	currentTime := time.Now()
	log.Printf("DEBUG: Current time for fleet arrivals: %s", currentTime.Format("2006-01-02 15:04:05.000Z"))
	
	// Find all fleets with destination_system set (in transit)
	fleets, err := app.Dao().FindRecordsByFilter("fleets", "destination_system != '' && eta != ''", "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch fleets in transit: %w", err)
	}

	arrivedCount := 0
	for _, fleet := range fleets {
		etaTime := fleet.GetDateTime("eta")
		if etaTime.IsZero() {
			continue
		}

		// Convert PocketBase DateTime to Go time.Time for comparison
		etaGoTime := etaTime.Time()
		
		// Check if fleet should have arrived (ETA is in the past or now)
		if etaGoTime.Before(currentTime) || etaGoTime.Equal(currentTime) {
			log.Printf("DEBUG: Processing arrival for fleet %s, eta: %s, current: %s, dest: %s", 
				fleet.Id, etaGoTime.Format("2006-01-02 15:04:05.000Z"), fleet.GetString("current_system"), fleet.GetString("destination_system"))
			
			// Check if this is a multi-hop journey with next_stop set
			nextStop := fleet.GetString("next_stop")
			
			if nextStop != "" {
				// This is a multi-hop journey - move to next_stop
				fleet.Set("current_system", nextStop)
				fleet.Set("next_stop", "")
				
				// Clear destination and eta - frontend will send next hop if needed
				fleet.Set("destination_system", "")
				fleet.Set("eta", "")
				log.Printf("DEBUG: Fleet %s reached waypoint %s in multi-hop journey", fleet.Id, nextStop)
			} else {
				// Single-hop journey - move to destination and clear
				fleet.Set("current_system", fleet.GetString("destination_system"))
				fleet.Set("destination_system", "")
				fleet.Set("eta", "")
				log.Printf("DEBUG: Fleet %s completed single-hop journey", fleet.Id)
			}

			if err := app.Dao().SaveRecord(fleet); err != nil {
				log.Printf("Failed to save arrived fleet %s: %v", fleet.Id, err)
			} else {
				log.Printf("DEBUG: Successfully moved fleet %s to destination", fleet.Id)
				arrivedCount++
			}
		}
	}

	log.Printf("Resolved %d fleet arrivals", arrivedCount)
	return nil
}

// resolveFleetArrival handles a single fleet arrival
func resolveFleetArrival(app *pocketbase.PocketBase, fleet *models.Record) error {
	destinationSystemId := fleet.GetString("destination_system")
	ownerId := fleet.GetString("owner_id")
	
	// Get ships in this fleet to calculate total strength
	ships, err := app.Dao().FindRecordsByFilter("ships", "fleet_id = '"+fleet.Id+"'", "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to get ships for fleet: %w", err)
	}
	
	totalStrength := 0
	for _, ship := range ships {
		shipTypeId := ship.GetString("ship_type")
		shipType, err := app.Dao().FindRecordById("ship_types", shipTypeId)
		if err != nil {
			continue
		}
		strength := shipType.GetInt("strength")
		count := ship.GetInt("count")
		totalStrength += strength * count
	}

	// Update fleet location
	fleet.Set("current_system", destinationSystemId)
	fleet.Set("destination_system", "")
	fleet.Set("eta", "")
	
	// Get planets in target system
	planets, err := app.Dao().FindRecordsByFilter("planets", "system_id = '"+destinationSystemId+"'", "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to get planets in system: %w", err)
	}

	if len(planets) == 0 {
		log.Printf("Fleet %s (owner %s) arrived at empty system %s", fleet.Id, ownerId, destinationSystemId)
		return app.Dao().SaveRecord(fleet)
	}

	// For now, just move fleet to the system and log arrival
	// TODO: Implement proper colonization, combat, and planet management
	log.Printf("Fleet %s (owner %s, strength %d) arrived at system %s with %d planets", 
		fleet.Id, ownerId, totalStrength, destinationSystemId, len(planets))

	return app.Dao().SaveRecord(fleet)
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