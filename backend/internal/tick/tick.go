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
	tickMutex.RLock()
	gameTick := currentTick
	tickMutex.RUnlock()

	log.Printf("Applying building completions for tick #%d...", gameTick)

	// Query for completed building orders
	completedOrders, err := app.Dao().FindRecordsByFilter(
		"building_queue",
		"completion_tick <= {:current_tick}",
		"-created", // Process oldest first if multiple complete on same tick
		0,          // No limit, process all completed
		0,
		map[string]interface{}{"current_tick": gameTick},
	)
	if err != nil {
		return fmt.Errorf("failed to query building_queue: %w", err)
	}

	if len(completedOrders) == 0 {
		log.Println("No building completions to apply.")
		return nil
	}

	processedCount := 0
	for _, order := range completedOrders {
		systemID := order.GetString("system_id")
		buildingType := order.GetString("building_type")
		targetLevel := order.GetInt("target_level")
		ownerID := order.GetString("owner_id") // For bank creation

		system, err := app.Dao().FindRecordById("systems", systemID)
		if err != nil {
			log.Printf("Error finding system %s for building order %s: %v. Skipping.", systemID, order.Id, err)
			// Decide if we should delete the order or retry later. For now, skipping.
			continue
		}

		log.Printf("Processing completion: System %s, Building %s, Target Level %d, Order %s", systemID, buildingType, targetLevel, order.Id)

		if buildingType == "bank" {
			// Handle bank creation
			// Ensure no existing bank (should be guaranteed by BuildOrderHandler, but double check)
			existingBank, _ := app.Dao().FindFirstRecordByFilter("banks", "system_id = {:systemId}", map[string]interface{}{"systemId": systemID})
			if existingBank != nil {
				log.Printf("Bank already exists for system %s, building order %s. Skipping.", systemID, order.Id)
			} else {
				bankCollection, err := app.Dao().FindCollectionByNameOrId("banks")
				if err != nil {
					log.Printf("Error finding banks collection for order %s: %v. Skipping.", order.Id, err)
					continue
				}
				bank := models.NewRecord(bankCollection)
				bank.Set("name", fmt.Sprintf("CryptoServer-%s", systemID[:8])) // Consistent naming
				bank.Set("owner_id", ownerID)
				bank.Set("system_id", systemID)
				bank.Set("security_level", 1)
				bank.Set("processing_power", 10)
				bank.Set("credits_per_tick", 1) // Base income
				bank.Set("active", true)
				bank.Set("last_income_tick", gameTick) // Set last income tick to current tick

				if err := app.Dao().SaveRecord(bank); err != nil {
					log.Printf("Error creating bank for system %s (order %s): %v. Skipping.", systemID, order.Id, err)
					continue
				}
				log.Printf("Created bank for system %s, linked to user %s.", systemID, ownerID)
			}
		} else {
			// Handle regular building level update
			fieldName := ""
			switch buildingType {
			case "habitat":
				fieldName = "hab_lvl"
			case "farm":
				fieldName = "farm_lvl"
			case "mine":
				fieldName = "mine_lvl"
			case "factory":
				fieldName = "fac_lvl"
			case "shipyard":
				fieldName = "yard_lvl"
			default:
				log.Printf("Unknown building type %s in order %s. Skipping.", buildingType, order.Id)
				continue
			}

			// Ensure we are not downgrading or setting an invalid level
			currentLevel := system.GetInt(fieldName)
			if targetLevel > currentLevel {
				system.Set(fieldName, targetLevel)
				if err := app.Dao().SaveRecord(system); err != nil {
					log.Printf("Error updating system %s for building %s (order %s): %v. Skipping.", systemID, buildingType, order.Id, err)
					continue
				}
				log.Printf("System %s building %s upgraded to level %d.", systemID, buildingType, targetLevel)
			} else {
				log.Printf("Target level %d not greater than current level %d for %s on system %s (order %s). Skipping.", targetLevel, currentLevel, buildingType, systemID, order.Id)
			}
		}

		// Delete the processed record from the queue
		if err := app.Dao().DeleteRecord(order); err != nil {
			log.Printf("Error deleting building order %s: %v.", order.Id, err)
			// This is not ideal, as it might re-process. Consider marking as processed instead.
		} else {
			processedCount++
		}
	}

	log.Printf("Applied %d building completions out of %d found.", processedCount, len(completedOrders))
	return nil
}

// MoveCargo handles trade route cargo movement
func MoveCargo(app *pocketbase.PocketBase) error {
	// Get all trade routes that should arrive this tick
	tickMutex.RLock()
	tick := currentTick
	tickMutex.RUnlock()
	
	routes, err := app.Dao().FindRecordsByFilter("trade_routes", "eta_tick <= {:tick}", "", 0, 0, map[string]interface{}{
		"tick": tick,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch trade routes: %w", err)
	}

	for _, route := range routes {

		// Move cargo from source to destination
		fromId := route.GetString("from_id")
		toId := route.GetString("to_id")
		cargo := route.GetString("cargo")
		capacity := route.GetInt("cap")

		if err := transferCargo(app, fromId, toId, cargo, capacity); err != nil {
			log.Printf("Failed to transfer cargo for route %s: %v", route.Id, err)
			continue
		}

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
func transferCargo(app *pocketbase.PocketBase, fromId, toId, cargo string, capacity int) error {
	// Get source system
	fromSystem, err := app.Dao().FindRecordById("systems", fromId)
	if err != nil {
		return fmt.Errorf("source system not found: %w", err)
	}

	// Get destination system
	toSystem, err := app.Dao().FindRecordById("systems", toId)
	if err != nil {
		return fmt.Errorf("destination system not found: %w", err)
	}

	// Check available cargo at source
	available := fromSystem.GetInt(cargo)
	if available <= 0 {
		return nil // No cargo to move
	}

	// Move up to capacity amount
	amount := capacity
	if available < amount {
		amount = available
	}

	// Update systems
	fromSystem.Set(cargo, available-amount)
	toSystem.Set(cargo, toSystem.GetInt(cargo)+amount)

	// Save changes
	if err := app.Dao().SaveRecord(fromSystem); err != nil {
		return fmt.Errorf("failed to update source system: %w", err)
	}

	if err := app.Dao().SaveRecord(toSystem); err != nil {
		return fmt.Errorf("failed to update destination system: %w", err)
	}

	log.Printf("Transferred %d %s from %s to %s", amount, cargo, fromId, toId)
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