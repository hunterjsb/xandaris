package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <user_id>")
	}
	
	userID := os.Args[1]
	
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: "./pb_data",
	})
	
	if err := app.Bootstrap(); err != nil {
		log.Fatal("Failed to bootstrap app:", err)
	}

	if err := createStartingFleet(app, userID); err != nil {
		log.Fatal("Failed to create starting fleet:", err)
	}
	
	fmt.Printf("✅ Successfully created starting fleet for user %s\n", userID)
}

func createStartingFleet(app *pocketbase.PocketBase, userID string) error {
	rand.Seed(time.Now().UnixNano())
	
	systems, err := app.Dao().FindRecordsByExpr("systems", nil, nil)
	if err != nil {
		return fmt.Errorf("error finding systems: %v", err)
	}
	if len(systems) == 0 {
		return fmt.Errorf("no systems available for starting location")
	}

	startingSystem := systems[rand.Intn(len(systems))]

	settlerShipType, err := app.Dao().FindFirstRecordByFilter("ship_types", "name='settler'")
	if err != nil {
		return fmt.Errorf("settler ship type not found: %v", err)
	}

	fleetCollection, err := app.Dao().FindCollectionByNameOrId("fleets")
	if err != nil {
		return err
	}

	fleet := models.NewRecord(fleetCollection)
	fleet.Set("owner_id", userID)
	fleet.Set("name", "Test Fleet")
	fleet.Set("current_system", startingSystem.Id)
	
	if err := app.Dao().SaveRecord(fleet); err != nil {
		return fmt.Errorf("failed to create fleet: %v", err)
	}

	shipCollection, err := app.Dao().FindCollectionByNameOrId("ships")
	if err != nil {
		return err
	}

	ship := models.NewRecord(shipCollection)
	ship.Set("fleet_id", fleet.Id)
	ship.Set("ship_type", settlerShipType.Id)
	ship.Set("count", 1)
	ship.Set("health", 100)

	if err := app.Dao().SaveRecord(ship); err != nil {
		return fmt.Errorf("failed to create settler ship: %v", err)
	}

	log.Printf("Created fleet %s for user %s at system %s", fleet.Id, userID, startingSystem.Id)
	return nil
}