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

	if err := resetColoniesToRealisticNumbers(app); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Colony reset completed successfully!")
}

func resetColoniesToRealisticNumbers(app *pocketbase.PocketBase) error {
	rand.Seed(time.Now().UnixNano())

	// Get all users
	users, err := app.Dao().FindRecordsByExpr("users", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	fmt.Printf("Found %d users\n", len(users))

	// Reset all planets to uncolonized first
	fmt.Println("Resetting all planets to uncolonized...")
	allPlanets, err := app.Dao().FindRecordsByExpr("planets", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get all planets: %w", err)
	}

	for _, planet := range allPlanets {
		planet.Set("colonized_by", nil)
		planet.Set("colonized_at", nil)
		if err := app.Dao().SaveRecord(planet); err != nil {
			return fmt.Errorf("failed to reset planet %s: %w", planet.Id, err)
		}
	}

	fmt.Printf("Reset %d planets to uncolonized\n", len(allPlanets))

	// Clear all existing populations
	fmt.Println("Clearing all populations...")
	populations, err := app.Dao().FindRecordsByExpr("populations", nil, nil)
	if err == nil {
		for _, pop := range populations {
			app.Dao().DeleteRecord(pop)
		}
		fmt.Printf("Cleared %d population records\n", len(populations))
	}

	// Clear all existing buildings
	fmt.Println("Clearing all buildings...")
	buildings, err := app.Dao().FindRecordsByExpr("buildings", nil, nil)
	if err == nil {
		for _, building := range buildings {
			app.Dao().DeleteRecord(building)
		}
		fmt.Printf("Cleared %d building records\n", len(buildings))
	}

	// For each user, colonize only 2-4 planets
	for _, user := range users {
		coloniesPerUser := rand.Intn(3) + 2 // 2-4 colonies per user
		
		// Get random uncolonized planets
		uncolonizedPlanets, err := app.Dao().FindRecordsByFilter("planets", "colonized_by = ''", "", coloniesPerUser*2, 0)
		if err != nil {
			continue
		}

		if len(uncolonizedPlanets) < coloniesPerUser {
			coloniesPerUser = len(uncolonizedPlanets)
		}

		// Colonize random planets for this user
		for i := 0; i < coloniesPerUser; i++ {
			planet := uncolonizedPlanets[i]
			planet.Set("colonized_by", user.Id)
			planet.Set("colonized_at", time.Now())

			if err := app.Dao().SaveRecord(planet); err != nil {
				fmt.Printf("Failed to colonize planet %s for user %s: %v\n", planet.Id, user.GetString("username"), err)
				continue
			}

			// Add population to this planet
			if err := addPopulationToPlanet(app, planet, user.Id); err != nil {
				fmt.Printf("Failed to add population to planet %s: %v\n", planet.Id, err)
			}

			// Add 1-2 buildings to this planet
			if err := addBuildingsToPlanet(app, planet); err != nil {
				fmt.Printf("Failed to add buildings to planet %s: %v\n", planet.Id, err)
			}
		}

		fmt.Printf("User %s colonized %d planets\n", user.GetString("username"), coloniesPerUser)
	}

	// Count final colonized planets
	finalColonizedPlanets, err := app.Dao().FindRecordsByFilter("planets", "colonized_by != ''", "", 0, 0)
	if err != nil {
		return err
	}

	fmt.Printf("\nFinal result: %d colonized planets (realistic for 4X gameplay)\n", len(finalColonizedPlanets))
	return nil
}

func addPopulationToPlanet(app *pocketbase.PocketBase, planet *models.Record, ownerID string) error {
	populationCollection, err := app.Dao().FindCollectionByNameOrId("populations")
	if err != nil {
		return err
	}

	population := models.NewRecord(populationCollection)
	population.Set("owner_id", ownerID)
	population.Set("planet_id", planet.Id)
	population.Set("count", rand.Intn(150)+50)  // 50-200 population
	population.Set("happiness", rand.Intn(20)+80) // 80-100 happiness

	return app.Dao().SaveRecord(population)
}

func addBuildingsToPlanet(app *pocketbase.PocketBase, planet *models.Record) error {
	// Get building types
	buildingTypes, err := app.Dao().FindRecordsByExpr("building_types", nil, nil)
	if err != nil {
		return err
	}

	if len(buildingTypes) == 0 {
		return nil
	}

	buildingCollection, err := app.Dao().FindCollectionByNameOrId("buildings")
	if err != nil {
		return err
	}

	// Add 1-2 random buildings per planet
	buildingCount := rand.Intn(2) + 1
	usedTypes := make(map[string]bool)

	for i := 0; i < buildingCount && len(usedTypes) < len(buildingTypes); i++ {
		// Pick a random building type we haven't used
		var buildingType *models.Record
		attempts := 0
		for attempts < 10 {
			buildingType = buildingTypes[rand.Intn(len(buildingTypes))]
			if !usedTypes[buildingType.Id] {
				usedTypes[buildingType.Id] = true
				break
			}
			attempts++
		}

		if attempts >= 10 {
			break // Avoid infinite loop
		}

		building := models.NewRecord(buildingCollection)
		building.Set("planet_id", planet.Id)
		building.Set("building_type", buildingType.Id)
		building.Set("level", rand.Intn(2)+1) // Level 1-2
		building.Set("active", true)

		if err := app.Dao().SaveRecord(building); err != nil {
			return err
		}
	}

	return nil
}