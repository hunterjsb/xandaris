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
	// Get all fleets that should arrive this tick (eta_tick <= current_tick)
	tickMutex.RLock()
	tick := currentTick
	tickMutex.RUnlock()
	
	fleets, err := app.Dao().FindRecordsByFilter("fleets", "eta_tick <= {:tick}", "", 0, 0, map[string]interface{}{
		"tick": tick,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch arriving fleets: %w", err)
	}

	for _, fleet := range fleets {
		if err := resolveFleetArrival(app, fleet); err != nil {
			log.Printf("Failed to resolve fleet %s arrival: %v", fleet.Id, err)
			continue
		}

		// Delete the fleet record (it has arrived)
		if err := app.Dao().DeleteRecord(fleet); err != nil {
			log.Printf("Failed to delete fleet %s: %v", fleet.Id, err)
		}
	}

	log.Printf("Resolved %d fleet arrivals", len(fleets))
	return nil
}

// resolveFleetArrival handles a single fleet arrival
func resolveFleetArrival(app *pocketbase.PocketBase, fleet *models.Record) error {
	toId := fleet.GetString("to_id")
	ownerId := fleet.GetString("owner_id")
	strength := fleet.GetInt("strength")

	// Get target system
	system, err := app.Dao().FindRecordById("systems", toId)
	if err != nil {
		return fmt.Errorf("target system not found: %w", err)
	}

	currentOwner := system.GetString("owner_id")

	// If system is unowned or owned by the same player, colonize/reinforce
	if currentOwner == "" || currentOwner == ownerId {
		system.Set("owner_id", ownerId)
		if currentOwner == "" {
			// New colonization
			system.Set("pop", strength*10) // Each fleet strength = 10 population
			system.Set("morale", 100)
			log.Printf("System %s colonized by %s", toId, ownerId)
		} else {
			// Reinforcement
			system.Set("pop", system.GetInt("pop")+strength*5)
			log.Printf("System %s reinforced by %s", toId, ownerId)
		}
	} else {
		// Combat with current owner
		defenseStrength := system.GetInt("pop") / 10 // Population provides defense
		if strength > defenseStrength {
			// Attacker wins
			system.Set("owner_id", ownerId)
			system.Set("pop", (strength-defenseStrength)*5)
			system.Set("morale", 50) // Conquered systems have low morale
			log.Printf("System %s conquered by %s", toId, ownerId)
		} else {
			// Defender wins
			system.Set("pop", system.GetInt("pop")-(strength*5))
			system.Set("morale", system.GetInt("morale")+10) // Successful defense boosts morale
			log.Printf("Attack on system %s repelled", toId)
		}
	}

	return app.Dao().SaveRecord(system)
}

// EvaluateTreaties checks for expired treaties and updates statuses
func EvaluateTreaties(app *pocketbase.PocketBase) error {
	// Get all active treaties
	treaties, err := app.Dao().FindRecordsByFilter("treaties", "status = 'active'", "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch treaties: %w", err)
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