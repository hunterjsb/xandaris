package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	_ "github.com/hunterjsb/xandaris/migrations"
)

func main() {
	app := pocketbase.New()

	if err := app.Bootstrap(); err != nil {
		log.Fatal(err)
	}

	if err := fixColonies(app); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Colony fix completed successfully!")
}

func fixColonies(app *pocketbase.PocketBase) error {
	rand.Seed(time.Now().UnixNano())

	// Step 1: Clear all broken colony data
	fmt.Println("Step 1: Clearing all existing colony data...")
	
	// Delete all populations
	populations, err := app.Dao().FindRecordsByExpr("populations", nil, nil)
	if err == nil {
		for _, pop := range populations {
			app.Dao().DeleteRecord(pop)
		}
		fmt.Printf("Deleted %d population records\n", len(populations))
	}

	// Delete all buildings
	buildings, err := app.Dao().FindRecordsByExpr("buildings", nil, nil)
	if err == nil {
		for _, building := range buildings {
			app.Dao().DeleteRecord(building)
		}
		fmt.Printf("Deleted %d building records\n", len(buildings))
	}

	// Reset all planets to truly uncolonized
	allPlanets, err := app.Dao().FindRecordsByExpr("planets", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get planets: %w", err)
	}

	fmt.Printf("Resetting %d planets to uncolonized...\n", len(allPlanets))
	for _, planet := range allPlanets {
		planet.Set("colonized_by", "")
		planet.Set("colonized_at", "")
		if err := app.Dao().SaveRecord(planet); err != nil {
			return fmt.Errorf("failed to reset planet: %w", err)
		}
	}

	// Step 2: Get users
	users, err := app.Dao().FindRecordsByExpr("users", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	fmt.Printf("Found %d users\n", len(users))

	// Step 3: Create realistic colonies for each user
	totalColonized := 0
	for _, user := range users {
		username := user.GetString("username")
		fmt.Printf("\nStep 3: Creating colonies for user %s...\n", username)

		// Each user gets 3-5 colonies
		coloniesForUser := rand.Intn(3) + 3

		// Get random planets (limit to first 100 for performance)
		candidatePlanets, err := app.Dao().FindRecordsByFilter("planets", "colonized_by = ''", "", 100, 0)
		if err != nil {
			fmt.Printf("Failed to get planets for user %s: %v\n", username, err)
			continue
		}

		if len(candidatePlanets) < coloniesForUser {
			coloniesForUser = len(candidatePlanets)
		}

		// Colonize random planets
		actualColonized := 0
		for i := 0; i < coloniesForUser && i < len(candidatePlanets); i++ {
			planet := candidatePlanets[i]
			
			// Colonize planet
			planet.Set("colonized_by", user.Id)
			planet.Set("colonized_at", time.Now())
			if err := app.Dao().SaveRecord(planet); err != nil {
				fmt.Printf("Failed to colonize planet: %v\n", err)
				continue
			}

			// Add population
			if err := createPopulation(app, planet.Id, user.Id); err != nil {
				fmt.Printf("Failed to add population: %v\n", err)
			}

			// Add buildings
			if err := createBuildings(app, planet.Id); err != nil {
				fmt.Printf("Failed to add buildings: %v\n", err)
			}

			actualColonized++
			totalColonized++
		}

		fmt.Printf("User %s colonized %d planets\n", username, actualColonized)
	}

	// Step 4: Verify results
	fmt.Printf("\n=== FINAL RESULTS ===\n")
	finalColonized, err := app.Dao().FindRecordsByFilter("planets", "colonized_by != ''", "", 0, 0)
	if err == nil {
		fmt.Printf("Total colonized planets: %d\n", len(finalColonized))
	}

	finalPopulations, err := app.Dao().FindRecordsByExpr("populations", nil, nil)
	if err == nil {
		fmt.Printf("Total population records: %d\n", len(finalPopulations))
	}

	finalBuildings, err := app.Dao().FindRecordsByExpr("buildings", nil, nil)
	if err == nil {
		fmt.Printf("Total building records: %d\n", len(finalBuildings))
	}

	fmt.Printf("\n✅ Realistic 4X colony setup complete!\n")
	return nil
}

func createPopulation(app *pocketbase.PocketBase, planetID, ownerID string) error {
	collection, err := app.Dao().FindCollectionByNameOrId("populations")
	if err != nil {
		return err
	}

	population := models.NewRecord(collection)
	population.Set("owner_id", ownerID)
	population.Set("planet_id", planetID)
	population.Set("count", rand.Intn(100)+100) // 100-200 population
	population.Set("happiness", rand.Intn(20)+75) // 75-95 happiness

	return app.Dao().SaveRecord(population)
}

func createBuildings(app *pocketbase.PocketBase, planetID string) error {
	// Get building types
	buildingTypes, err := app.Dao().FindRecordsByExpr("building_types", nil, nil)
	if err != nil || len(buildingTypes) == 0 {
		return err
	}

	collection, err := app.Dao().FindCollectionByNameOrId("buildings")
	if err != nil {
		return err
	}

	// Add 2-3 buildings per planet
	buildingCount := rand.Intn(2) + 2
	for i := 0; i < buildingCount && i < len(buildingTypes); i++ {
		buildingType := buildingTypes[i]

		building := models.NewRecord(collection)
		building.Set("planet_id", planetID)
		building.Set("building_type", buildingType.Id)
		building.Set("level", rand.Intn(2)+1) // Level 1-2
		building.Set("active", true)

		if err := app.Dao().SaveRecord(building); err != nil {
			return err
		}
	}

	return nil
}