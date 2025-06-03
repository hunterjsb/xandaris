package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

func main() {
	app := pocketbase.New()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// Setup starter resources for new players
		return setupStarterResources(app)
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func setupStarterResources(app *pocketbase.PocketBase) error {
	fmt.Println("🚀 Setting up starter resources for new players")

	// Get all users without fleets (new players)
	users, err := app.Dao().FindRecordsByExpr("users", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get users: %v", err)
	}

	for _, user := range users {
		// Check if user already has a fleet
		existingFleets, err := app.Dao().FindRecordsByFilter("fleets", fmt.Sprintf("owner_id='%s'", user.Id), "", 1, 0)
		if err == nil && len(existingFleets) > 0 {
			fmt.Printf("⏭️ User %s already has fleet, skipping\n", user.GetString("username"))
			continue
		}

		fmt.Printf("🆕 Setting up starter resources for user: %s\n", user.GetString("username"))

		// Create starter fleet
		fleetsCollection, err := app.Dao().FindCollectionByNameOrId("fleets")
		if err != nil {
			return fmt.Errorf("fleets collection not found: %v", err)
		}

		fleet := models.NewRecord(fleetsCollection)
		fleet.Set("owner_id", user.Id)
		fleet.Set("name", "Settler Fleet")
		
		// Find a random system for starting location
		systems, err := app.Dao().FindRecordsByFilter("systems", "", "", 1, 0)
		if err != nil || len(systems) == 0 {
			return fmt.Errorf("no systems found for starting location")
		}
		fleet.Set("current_system", systems[0].Id)

		if err := app.Dao().SaveRecord(fleet); err != nil {
			return fmt.Errorf("failed to create starter fleet: %v", err)
		}

		// Get settler ship type
		settlerShipType, err := app.Dao().FindFirstRecordByFilter("ship_types", "name='settler'")
		if err != nil {
			return fmt.Errorf("settler ship type not found: %v", err)
		}

		// Create settler ship
		shipsCollection, err := app.Dao().FindCollectionByNameOrId("ships")
		if err != nil {
			return fmt.Errorf("ships collection not found: %v", err)
		}

		ship := models.NewRecord(shipsCollection)
		ship.Set("fleet_id", fleet.Id)
		ship.Set("ship_type", settlerShipType.Id)
		ship.Set("count", 1)
		ship.Set("health", 100)

		if err := app.Dao().SaveRecord(ship); err != nil {
			return fmt.Errorf("failed to create starter ship: %v", err)
		}

		// Add 30 ore to ship cargo
		oreResource, err := app.Dao().FindFirstRecordByFilter("resource_types", "name='ore'")
		if err != nil {
			return fmt.Errorf("ore resource type not found: %v", err)
		}

		cargoCollection, err := app.Dao().FindCollectionByNameOrId("ship_cargo")
		if err != nil {
			return fmt.Errorf("ship_cargo collection not found: %v", err)
		}

		cargo := models.NewRecord(cargoCollection)
		cargo.Set("ship_id", ship.Id)
		cargo.Set("resource_type", oreResource.Id)
		cargo.Set("quantity", 30)

		if err := app.Dao().SaveRecord(cargo); err != nil {
			return fmt.Errorf("failed to create starter cargo: %v", err)
		}

		fmt.Printf("✅ Created starter fleet '%s' with settler ship and 30 ore for user %s\n", 
			fleet.GetString("name"), user.GetString("username"))
	}

	fmt.Println("🎉 Starter resource setup completed!")
	os.Exit(0)
	return nil
}