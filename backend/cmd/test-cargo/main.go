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
		// Test the ship cargo and building system
		return testCargoSystem(app)
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func testCargoSystem(app *pocketbase.PocketBase) error {
	fmt.Println("🧪 Testing Ship Cargo and Building System")

	// Get the test user (use existing user ID from database)
	user, err := app.Dao().FindRecordById("users", "0opov99t9085loi")
	if err != nil {
		return fmt.Errorf("test user not found: %v", err)
	}
	fmt.Println("✅ Found test user")

	// Get the test fleet
	fleets, err := app.Dao().FindRecordsByFilter("fleets", fmt.Sprintf("owner_id='%s'", user.Id), "", 1, 0)
	if err != nil || len(fleets) == 0 {
		return fmt.Errorf("no fleets found for user")
	}
	fleet := fleets[0]
	fmt.Printf("✅ Found test fleet: %s\n", fleet.GetString("name"))

	// Get ships in fleet
	ships, err := app.Dao().FindRecordsByFilter("ships", fmt.Sprintf("fleet_id='%s'", fleet.Id), "", 0, 0)
	if err != nil || len(ships) == 0 {
		return fmt.Errorf("no ships found in fleet")
	}
	ship := ships[0]
	fmt.Printf("✅ Found ship in fleet\n")

	// Check current cargo
	cargo, err := app.Dao().FindRecordsByFilter("ship_cargo", fmt.Sprintf("ship_id='%s'", ship.Id), "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to get ship cargo: %v", err)
	}

	fmt.Printf("🏭 Current cargo:\n")
	totalCargo := 0
	for _, c := range cargo {
		resourceType, _ := app.Dao().FindRecordById("resource_types", c.GetString("resource_type"))
		resourceName := "unknown"
		if resourceType != nil {
			resourceName = resourceType.GetString("name")
		}
		quantity := c.GetInt("quantity")
		totalCargo += quantity
		fmt.Printf("  - %s: %d\n", resourceName, quantity)
	}
	fmt.Printf("  Total cargo: %d\n", totalCargo)

	// Get a planet owned by the user
	planets, err := app.Dao().FindRecordsByFilter("planets", fmt.Sprintf("colonized_by='%s'", user.Id), "", 1, 0)
	if err != nil || len(planets) == 0 {
		return fmt.Errorf("no planets found for user")
	}
	planet := planets[0]
	fmt.Printf("✅ Found planet: %s in system %s\n", planet.GetString("name"), planet.GetString("system_id"))

	// Move fleet to same system as planet
	fleet.Set("current_system", planet.GetString("system_id"))
	if err := app.Dao().SaveRecord(fleet); err != nil {
		return fmt.Errorf("failed to move fleet: %v", err)
	}
	fmt.Printf("✅ Moved fleet to planet system\n")

	// Test building construction
	fmt.Println("🏗️ Testing building construction...")

	// Get ore resource type
	oreResource, err := app.Dao().FindFirstRecordByFilter("resource_types", "name='ore'")
	if err != nil {
		return fmt.Errorf("ore resource type not found: %v", err)
	}

	// Get mine building type
	mineBuildingType, err := app.Dao().FindFirstRecordByFilter("building_types", "name='mine'")
	if err != nil {
		return fmt.Errorf("mine building type not found: %v", err)
	}

	fmt.Printf("📋 Building costs: %d %s\n", 
		mineBuildingType.GetInt("cost_quantity"), 
		"ore")

	// Check if we have enough ore
	oreCargoRecord, err := app.Dao().FindFirstRecordByFilter("ship_cargo", 
		fmt.Sprintf("ship_id='%s' && resource_type='%s'", ship.Id, oreResource.Id))
	if err != nil {
		return fmt.Errorf("no ore found in ship cargo: %v", err)
	}

	currentOre := oreCargoRecord.GetInt("quantity")
	requiredOre := mineBuildingType.GetInt("cost_quantity")

	fmt.Printf("💰 Available ore: %d, Required: %d\n", currentOre, requiredOre)

	if currentOre < requiredOre {
		return fmt.Errorf("insufficient ore for building")
	}

	// Simulate building construction
	fmt.Println("🔨 Constructing mine...")

	// Deduct ore from cargo
	newOreQuantity := currentOre - requiredOre
	if newOreQuantity <= 0 {
		if err := app.Dao().DeleteRecord(oreCargoRecord); err != nil {
			return fmt.Errorf("failed to delete empty cargo: %v", err)
		}
		fmt.Println("  - Removed empty ore cargo")
	} else {
		oreCargoRecord.Set("quantity", newOreQuantity)
		if err := app.Dao().SaveRecord(oreCargoRecord); err != nil {
			return fmt.Errorf("failed to update cargo: %v", err)
		}
		fmt.Printf("  - Reduced ore cargo to %d\n", newOreQuantity)
	}

	// Create building
	buildingsCollection, err := app.Dao().FindCollectionByNameOrId("buildings")
	if err != nil {
		return fmt.Errorf("buildings collection not found: %v", err)
	}

	building := models.NewRecord(buildingsCollection)
	building.Set("planet_id", planet.Id)
	building.Set("building_type", mineBuildingType.Id)
	building.Set("level", 1)
	building.Set("active", true)

	if err := app.Dao().SaveRecord(building); err != nil {
		return fmt.Errorf("failed to create building: %v", err)
	}

	fmt.Printf("✅ Successfully built mine on planet %s!\n", planet.GetString("name"))
	fmt.Printf("🏭 Building ID: %s\n", building.Id)

	// Final cargo check
	finalCargo, _ := app.Dao().FindRecordsByFilter("ship_cargo", fmt.Sprintf("ship_id='%s'", ship.Id), "", 0, 0)
	fmt.Printf("\n📦 Final cargo:\n")
	finalTotal := 0
	for _, c := range finalCargo {
		resourceType, _ := app.Dao().FindRecordById("resource_types", c.GetString("resource_type"))
		resourceName := "unknown"
		if resourceType != nil {
			resourceName = resourceType.GetString("name")
		}
		quantity := c.GetInt("quantity")
		finalTotal += quantity
		fmt.Printf("  - %s: %d\n", resourceName, quantity)
	}
	fmt.Printf("  Total cargo: %d\n", finalTotal)

	fmt.Println("\n🎉 Ship cargo and building system test completed successfully!")

	os.Exit(0)
	return nil
}