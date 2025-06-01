package tick

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"

	"github.com/hunterjsb/xandaris/internal/diplomacy"
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
	// Get all fleets that have an ETA in the past (should have arrived)
	currentTime := time.Now().Format("2006-01-02 15:04:05.000Z")
	
	fleets, err := app.Dao().FindRecordsByFilter("fleets", "eta != '' && eta <= '"+currentTime+"'", "", 0, 0)
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

// EvaluateTreaties checks for expired diplomatic proposals and relations.
func EvaluateTreaties(app *pocketbase.PocketBase) error {
	now := time.Now().UTC()
	nowStr := now.Format("2006-01-02 15:04:05.999Z")

	// Handle Proposal Expiration
	proposalsCollection, err := app.Dao().FindCollectionByNameOrId("diplomatic_proposals")
	if err != nil {
		log.Printf("Diplomatic proposals collection not found, skipping proposal expiration: %v", err)
	} else {
		expiredProposals := 0
		// Find pending proposals where expiration_date is in the past.
		// Note: PocketBase filter date comparisons require the date to be in its string representation.
		records, err := app.Dao().FindRecordsByFilter(
			proposalsCollection.Name,
			"status = 'pending' && expiration_date < {:now}",
			"-created", // sort by oldest first, though not strictly necessary here
			0,          // limit (0 for no limit)
			0,          // offset
			map[string]any{"now": nowStr},
		)
		if err != nil {
			log.Printf("Error fetching pending diplomatic proposals: %v", err)
		} else {
			for _, proposalRecord := range records {
				proposalRecord.Set("status", "expired")
				if err := app.Dao().SaveRecord(proposalRecord); err != nil {
					log.Printf("Failed to update proposal %s to expired: %v", proposalRecord.Id, err)
					continue
				}
				expiredProposals++
				log.Printf("Diplomatic proposal %s (type: %s) between %s and %s has expired.",
					proposalRecord.Id,
					proposalRecord.GetString("type"),
					proposalRecord.GetString("proposer_id"),
					proposalRecord.GetString("receiver_id"))
			}
			if expiredProposals > 0 {
				log.Printf("Total expired diplomatic proposals: %d", expiredProposals)
			}
		}
	}

	// Handle Relation Expiration (End of Treaties)
	relationsCollection, err := app.Dao().FindCollectionByNameOrId("diplomatic_relations")
	if err != nil {
		log.Printf("Diplomatic relations collection not found, skipping relation expiration: %v", err)
	} else {
		updatedRelations := 0
		// Find relations with an end_date set and where end_date is in the past.
		// Exclude wars, as they don't expire by date.
		records, err := app.Dao().FindRecordsByFilter(
			relationsCollection.Name,
			"end_date != null && end_date < {:now} && status != 'war'",
			"-created",
			0,
			0,
			map[string]any{"now": nowStr},
		)
		if err != nil {
			log.Printf("Error fetching expiring diplomatic relations: %v", err)
		} else {
			for _, relationRecord := range records {
				originalStatus := relationRecord.GetString("status")
				newStatus := "peace" // Default to peace after expiration

				// Specific handling for different expiring statuses
				if originalStatus == "alliance" {
					newStatus = "peace" // Alliance expires into peace
					log.Printf("Diplomatic relation (Alliance) %s between %s and %s has expired, changing status to Peace.",
						relationRecord.Id, relationRecord.GetString("player1_id"), relationRecord.GetString("player2_id"))
				} else if originalStatus == "peace" {
					// Fixed-term peace expires, becomes indefinite peace (effectively, the treaty obligations end)
					log.Printf("Diplomatic relation (Fixed-term Peace) %s between %s and %s has ended.",
						relationRecord.Id, relationRecord.GetString("player1_id"), relationRecord.GetString("player2_id"))
				} else if originalStatus == "truce" {
					newStatus = "peace" // Truce expires into peace
					log.Printf("Diplomatic relation (Truce) %s between %s and %s has expired, changing status to Peace.",
						relationRecord.Id, relationRecord.GetString("player1_id"), relationRecord.GetString("player2_id"))
				} else {
					log.Printf("Diplomatic relation (Status: %s) %s between %s and %s with end_date %s has passed. Setting to 'peace'.",
						originalStatus, relationRecord.Id, relationRecord.GetString("player1_id"), relationRecord.GetString("player2_id"), relationRecord.GetDateTime("end_date"))
				}

				relationRecord.Set("status", newStatus)
				relationRecord.Set("duration_ticks", 0) // Becomes indefinite
				relationRecord.Set("end_date", nil)     // Clear the end date

				if err := app.Dao().SaveRecord(relationRecord); err != nil {
					log.Printf("Failed to update relation %s to %s: %v", relationRecord.Id, newStatus, err)
					continue
				}
				updatedRelations++
			}
			if updatedRelations > 0 {
				log.Printf("Total updated diplomatic relations due to expiration: %d", updatedRelations)
			}
		}
	}
	return nil
}