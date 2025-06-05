package player

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	"github.com/hunterjsb/xandaris/internal/resources" // Added import
)

// CreateUserStarterFleet creates a new fleet with a settler ship and starter cargo for a user in a specific system.
// It returns the created fleet record, ship record, or an error if any step fails.
func CreateUserStarterFleet(app *pocketbase.PocketBase, userID string, systemID string) (fleetRecord *models.Record, shipRecord *models.Record, err error) {
	log.Printf("INFO: Attempting to create starter fleet for user %s in system %s", userID, systemID)

	// 1. Find "settler" ship type
	settlerShipType, err := app.Dao().FindFirstRecordByFilter("ship_types", "name = 'settler'")
	if err != nil {
		log.Printf("ERROR: CreateUserStarterFleet: Failed to query for 'settler' ship type: %v", err)
		return nil, nil, fmt.Errorf("failed to find settler ship type definition: %w", err)
	}
	if settlerShipType == nil {
		log.Printf("ERROR: CreateUserStarterFleet: 'settler' ship type definition not found in database.")
		return nil, nil, fmt.Errorf("settler ship type definition not found")
	}
	log.Printf("DEBUG: CreateUserStarterFleet: Found settler ship type ID: %s", settlerShipType.Id)

	// 2. Create new fleet record
	fleetCollection, err := app.Dao().FindCollectionByNameOrId("fleets")
	if err != nil {
		log.Printf("ERROR: CreateUserStarterFleet: Failed to find 'fleets' collection: %v", err)
		return nil, nil, fmt.Errorf("fleets collection not found: %w", err)
	}

	fleetRecord = models.NewRecord(fleetCollection)
	fleetRecord.Set("owner_id", userID)
	fleetRecord.Set("name", "Starter Fleet") // Default name
	fleetRecord.Set("current_system", systemID)
	// ETA is not applicable for a stationary fleet, so not setting it.

	if err := app.Dao().SaveRecord(fleetRecord); err != nil {
		log.Printf("ERROR: CreateUserStarterFleet: Failed to create fleet for user %s in system %s: %v", userID, systemID, err)
		return nil, nil, fmt.Errorf("failed to create fleet: %w", err)
	}
	log.Printf("INFO: CreateUserStarterFleet: Created fleet %s for user %s in system %s", fleetRecord.Id, userID, systemID)

	// 3. Create new "settler" ship record
	shipCollection, err := app.Dao().FindCollectionByNameOrId("ships")
	if err != nil {
		log.Printf("ERROR: CreateUserStarterFleet: Failed to find 'ships' collection: %v", err)
		return fleetRecord, nil, fmt.Errorf("ships collection not found: %w", err) // Return fleet as it was created
	}

	shipRecord = models.NewRecord(shipCollection)
	shipRecord.Set("fleet_id", fleetRecord.Id)
	shipRecord.Set("ship_type", settlerShipType.Id)
	shipRecord.Set("count", 1)
	shipRecord.Set("health", 100) // Default health

	if err := app.Dao().SaveRecord(shipRecord); err != nil {
		log.Printf("ERROR: CreateUserStarterFleet: Failed to create settler ship for fleet %s: %v", fleetRecord.Id, err)
		// Fleet was created, but ship failed. Caller might want to handle this (e.g., delete fleet or log inconsistency).
		return fleetRecord, nil, fmt.Errorf("failed to create settler ship: %w", err)
	}
	log.Printf("INFO: CreateUserStarterFleet: Created settler ship %s in fleet %s", shipRecord.Id, fleetRecord.Id)

	// 4. Add default starter cargo
	cargoCollection, err := app.Dao().FindCollectionByNameOrId("ship_cargo")
	if err != nil {
		log.Printf("ERROR: CreateUserStarterFleet: Failed to find 'ship_cargo' collection: %v", err)
		return fleetRecord, shipRecord, fmt.Errorf("ship_cargo collection not found: %w", err) // Fleet & ship created
	}

	starterCargoSpec := map[string]int{
		"ore":   50,
		"food":  25,
		"metal": 20,
		"fuel":  15,
	}

	// Get the resource ID -> Name map
	resourceIdToNameMap, err := resources.GetResourceTypeMap(app)
	if err != nil {
		log.Printf("ERROR: CreateUserStarterFleet: Failed to get resource type map: %v", err)
		// Decide if this is fatal. For starter fleet, it's quite important.
		return fleetRecord, shipRecord, fmt.Errorf("failed to get resource type map for starter cargo: %w", err)
	}

	// Invert the map to get Name -> ID for easier lookup from starterCargoSpec
	resourceNameToIdMap := make(map[string]string)
	for id, name := range resourceIdToNameMap {
		resourceNameToIdMap[name] = id
	}
	log.Printf("DEBUG: CreateUserStarterFleet: Inverted resource map for cargo: %+v", resourceNameToIdMap)

	for resourceName, quantity := range starterCargoSpec {
		resourceTypeID, ok := resourceNameToIdMap[resourceName]
		if !ok {
			// Already logged warning if not found by the previous loop
			continue
		}

		cargoRecord := models.NewRecord(cargoCollection)
		cargoRecord.Set("ship_id", shipRecord.Id)
		cargoRecord.Set("resource_type", resourceTypeID)
		cargoRecord.Set("quantity", quantity)

		if err := app.Dao().SaveRecord(cargoRecord); err != nil {
			log.Printf("WARN: CreateUserStarterFleet: Failed to add %s cargo (qty: %d) to ship %s: %v", resourceName, quantity, shipRecord.Id, err)
			// Continue adding other cargo items even if one fails
		} else {
			log.Printf("DEBUG: CreateUserStarterFleet: Added %d %s to ship %s", quantity, resourceName, shipRecord.Id)
		}
	}
	log.Printf("INFO: CreateUserStarterFleet: Finished adding starter cargo to ship %s", shipRecord.Id)

	return fleetRecord, shipRecord, nil
}
