package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "github.com/hunterjsb/xandaris/migrations"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

func main() {
	app := pocketbase.New()

	if err := app.Bootstrap(); err != nil {
		log.Fatal(err)
	}

	if err := seedNewSchema(app); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Seeding completed successfully!")
}

func seedNewSchema(app *pocketbase.PocketBase) error {
	rand.Seed(time.Now().UnixNano())

	// 1. Seed resource types
	if err := seedResourceTypes(app); err != nil {
		return fmt.Errorf("failed to seed resource types: %w", err)
	}

	// 2. Seed planet types
	if err := seedPlanetTypes(app); err != nil {
		return fmt.Errorf("failed to seed planet types: %w", err)
	}

	// 3. Seed building types
	if err := seedBuildingTypes(app); err != nil {
		return fmt.Errorf("failed to seed building types: %w", err)
	}

	// 4. Seed ship types
	if err := seedShipTypes(app); err != nil {
		return fmt.Errorf("failed to seed ship types: %w", err)
	}

	// 5. Generate systems
	if err := seedSystems(app, 50); err != nil {
		return fmt.Errorf("failed to seed systems: %w", err)
	}

	// 6. Generate planets for each system
	if err := seedPlanets(app); err != nil {
		return fmt.Errorf("failed to seed planets: %w", err)
	}

	// 7. Generate resource nodes for planets
	if err := seedResourceNodes(app); err != nil {
		return fmt.Errorf("failed to seed resource nodes: %w", err)
	}

	// 8. Create sample user and colonize some planets
	if err := seedSampleUserAndColonies(app); err != nil {
		return fmt.Errorf("failed to seed sample user and colonies: %w", err)
	}

	return nil
}

func seedSampleUserAndColonies(app *pocketbase.PocketBase) error {
	// Get all existing users
	allUsers, err := app.Dao().FindRecordsByExpr("users", nil, nil)
	if err != nil {
		return err
	}

	if len(allUsers) == 0 {
		fmt.Println("No users found in the database. Please create a user first.")
		return nil
	}

	fmt.Printf("Found %d users to give colonies\n", len(allUsers))

	// Get all planets and filter uncolonized ones in Go
	allPlanets, err := app.Dao().FindRecordsByExpr("planets", nil, nil)
	if err != nil {
		return err
	}

	// Filter uncolonized planets
	var uncolonizedPlanets []*models.Record
	for _, planet := range allPlanets {
		if planet.GetString("colonized_by") == "" {
			uncolonizedPlanets = append(uncolonizedPlanets, planet)
		}
	}

	if len(uncolonizedPlanets) < len(allUsers)*3 {
		fmt.Printf("Warning: Not enough uncolonized planets (%d) for all users to get 3 colonies each\n", len(uncolonizedPlanets))
	}

	planetIndex := 0
	coloniesPerUser := 3

	// Give each user some colonies
	for _, user := range allUsers {
		username := user.GetString("username")
		email := user.GetString("email")

		// Check how many planets this user already has colonized
		existingColonies, err := app.Dao().FindRecordsByFilter("planets", "colonized_by = '"+user.Id+"'", "", 0, 0)
		if err != nil {
			return err
		}

		if len(existingColonies) >= coloniesPerUser {
			fmt.Printf("  %s (%s) already has %d colonies\n", username, email, len(existingColonies))
			continue
		}

		// Colonize planets for this user
		planetsToColonize := coloniesPerUser - len(existingColonies)
		colonized := 0

		for j := 0; j < planetsToColonize && planetIndex < len(uncolonizedPlanets); j++ {
			planet := uncolonizedPlanets[planetIndex]
			planetIndex++

			// Colonize planet
			planet.Set("colonized_by", user.Id)
			planet.Set("colonized_at", time.Now())

			if err := app.Dao().SaveRecord(planet); err != nil {
				fmt.Printf("    Error colonizing planet: %v\n", err)
				continue
			}

			// Add population to this planet
			if err := seedPopulationForPlanet(app, planet, user.Id); err != nil {
				fmt.Printf("    Error adding population: %v\n", err)
			}

			// Add some buildings
			if err := seedBuildingsForPlanet(app, planet); err != nil {
				fmt.Printf("    Error adding buildings: %v\n", err)
			}

			colonized++
		}

		totalColonies := len(existingColonies) + colonized
		fmt.Printf("  %s (%s) now has %d colonies\n", username, email, totalColonies)
	}
	return nil
}

func seedPopulationForPlanet(app *pocketbase.PocketBase, planet *models.Record, ownerID string) error {
	populationCollection, err := app.Dao().FindCollectionByNameOrId("populations")
	if err != nil {
		return err
	}

	population := models.NewRecord(populationCollection)
	population.Set("owner_id", ownerID)
	population.Set("planet_id", planet.Id)
	population.Set("count", rand.Intn(200)+50)    // 50-250 population
	population.Set("happiness", rand.Intn(30)+70) // 70-100 happiness

	return app.Dao().SaveRecord(population)
}

func seedBuildingsForPlanet(app *pocketbase.PocketBase, planet *models.Record) error {
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

	// Add 1-3 random buildings per planet
	buildingCount := rand.Intn(3) + 1
	usedTypes := make(map[string]bool)

	for i := 0; i < buildingCount && len(usedTypes) < len(buildingTypes); i++ {
		// Pick a random building type we haven't used
		var buildingType *models.Record
		for {
			buildingType = buildingTypes[rand.Intn(len(buildingTypes))]
			if !usedTypes[buildingType.Id] {
				usedTypes[buildingType.Id] = true
				break
			}
		}

		building := models.NewRecord(buildingCollection)
		building.Set("planet_id", planet.Id)
		building.Set("building_type", buildingType.Id)
		building.Set("level", rand.Intn(3)+1) // Level 1-3
		building.Set("active", true)

		if err := app.Dao().SaveRecord(building); err != nil {
			return err
		}
	}

	return nil
}

func seedResourceTypes(app *pocketbase.PocketBase) error {
	collection, err := app.Dao().FindCollectionByNameOrId("resource_types")
	if err != nil {
		return err
	}

	resources := []map[string]interface{}{
		{"name": "food", "description": "Essential for population growth and happiness", "is_consumable": true},
		{"name": "ore", "description": "Raw materials for construction and manufacturing", "is_consumable": false},
		{"name": "goods", "description": "Manufactured products for trade and population", "is_consumable": true},
		{"name": "fuel", "description": "Energy source for ships and industry", "is_consumable": true},
		{"name": "water", "description": "Life-sustaining resource", "is_consumable": true},
		{"name": "rare_metals", "description": "Advanced materials for high-tech construction", "is_consumable": false},
	}

	for _, resource := range resources {
		record := models.NewRecord(collection)
		record.Set("name", resource["name"])
		record.Set("description", resource["description"])
		record.Set("is_consumable", resource["is_consumable"])
		if err := app.Dao().SaveRecord(record); err != nil {
			return err
		}
	}

	return nil
}

func seedPlanetTypes(app *pocketbase.PocketBase) error {
	collection, err := app.Dao().FindCollectionByNameOrId("planet_types")
	if err != nil {
		return err
	}

	planetTypes := []map[string]interface{}{
		{"name": "terran", "base_max_population": 1000, "habitability": 1.0},
		{"name": "arid", "base_max_population": 600, "habitability": 0.7},
		{"name": "ocean", "base_max_population": 800, "habitability": 0.8},
		{"name": "arctic", "base_max_population": 400, "habitability": 0.5},
		{"name": "volcanic", "base_max_population": 300, "habitability": 0.4},
		{"name": "gas_giant", "base_max_population": 0, "habitability": 0.0},
	}

	for _, planetType := range planetTypes {
		record := models.NewRecord(collection)
		record.Set("name", planetType["name"])
		record.Set("base_max_population", planetType["base_max_population"])
		record.Set("habitability", planetType["habitability"])
		if err := app.Dao().SaveRecord(record); err != nil {
			return err
		}
	}

	return nil
}

func seedBuildingTypes(app *pocketbase.PocketBase) error {
	collection, err := app.Dao().FindCollectionByNameOrId("building_types")
	if err != nil {
		return err
	}

	buildingTypes := []map[string]interface{}{
		{"name": "farm", "cost": 100, "worker_capacity": 50, "max_level": 10},
		{"name": "mine", "cost": 150, "worker_capacity": 30, "max_level": 8},
		{"name": "factory", "cost": 200, "worker_capacity": 40, "max_level": 12},
		{"name": "power_plant", "cost": 300, "worker_capacity": 20, "max_level": 6},
		{"name": "spaceport", "cost": 500, "worker_capacity": 100, "max_level": 5},
		{"name": "research_lab", "cost": 400, "worker_capacity": 25, "max_level": 15},
	}

	for _, building := range buildingTypes {
		record := models.NewRecord(collection)
		record.Set("name", building["name"])
		record.Set("cost", building["cost"])
		record.Set("worker_capacity", building["worker_capacity"])
		record.Set("max_level", building["max_level"])
		if err := app.Dao().SaveRecord(record); err != nil {
			return err
		}
	}

	return nil
}

func seedShipTypes(app *pocketbase.PocketBase) error {
	collection, err := app.Dao().FindCollectionByNameOrId("ship_types")
	if err != nil {
		return err
	}

	shipTypes := []map[string]interface{}{
		{"name": "scout", "cost": 50, "strength": 1, "cargo_capacity": 10},
		{"name": "fighter", "cost": 100, "strength": 5, "cargo_capacity": 5},
		{"name": "frigate", "cost": 200, "strength": 15, "cargo_capacity": 20},
		{"name": "transport", "cost": 150, "strength": 2, "cargo_capacity": 100},
		{"name": "cruiser", "cost": 500, "strength": 50, "cargo_capacity": 30},
		{"name": "battleship", "cost": 1000, "strength": 100, "cargo_capacity": 50},
	}

	for _, ship := range shipTypes {
		record := models.NewRecord(collection)
		record.Set("name", ship["name"])
		record.Set("cost", ship["cost"])
		record.Set("strength", ship["strength"])
		record.Set("cargo_capacity", ship["cargo_capacity"])
		if err := app.Dao().SaveRecord(record); err != nil {
			return err
		}
	}

	return nil
}

func seedSystems(app *pocketbase.PocketBase, count int) error {
	collection, err := app.Dao().FindCollectionByNameOrId("systems")
	if err != nil {
		return err
	}

	// Generate systems in a grid pattern with some randomization
	gridSize := int(float64(count) * 0.8) // Not quite square for variety
	spacing := 200

	for i := 0; i < count; i++ {
		x := (i % gridSize) * spacing
		y := (i / gridSize) * spacing

		// Add some randomization
		x += rand.Intn(100) - 50
		y += rand.Intn(100) - 50

		record := models.NewRecord(collection)
		record.Set("name", fmt.Sprintf("System-%d", i+1))
		record.Set("x", x)
		record.Set("y", y)

		if err := app.Dao().SaveRecord(record); err != nil {
			return err
		}
	}

	return nil
}

func seedPlanets(app *pocketbase.PocketBase) error {
	// Get all systems
	systems, err := app.Dao().FindRecordsByExpr("systems", nil, nil)
	if err != nil {
		return err
	}

	// Get planet types
	planetTypes, err := app.Dao().FindRecordsByExpr("planet_types", nil, nil)
	if err != nil {
		return err
	}

	planetCollection, err := app.Dao().FindCollectionByNameOrId("planets")
	if err != nil {
		return err
	}

	planetCounter := 1

	for _, system := range systems {
		// Each system has 1-4 planets
		planetCount := rand.Intn(4) + 1

		for j := 0; j < planetCount; j++ {
			// Random planet type
			planetType := planetTypes[rand.Intn(len(planetTypes))]

			record := models.NewRecord(planetCollection)
			record.Set("name", fmt.Sprintf("Planet-%d", planetCounter))
			record.Set("system_id", system.Id)
			record.Set("planet_type", planetType.Id)
			record.Set("size", rand.Intn(5)+1) // Size 1-5

			if err := app.Dao().SaveRecord(record); err != nil {
				return err
			}

			planetCounter++
		}
	}

	return nil
}

func seedResourceNodes(app *pocketbase.PocketBase) error {
	// Get all planets
	planets, err := app.Dao().FindRecordsByExpr("planets", nil, nil)
	if err != nil {
		return err
	}

	// Get resource types
	resourceTypes, err := app.Dao().FindRecordsByExpr("resource_types", nil, nil)
	if err != nil {
		return err
	}

	resourceNodeCollection, err := app.Dao().FindCollectionByNameOrId("resource_nodes")
	if err != nil {
		return err
	}

	for _, planet := range planets {
		planetSize := planet.GetInt("size")

		// Larger planets have more resource nodes
		nodeCount := planetSize + rand.Intn(3)

		for j := 0; j < nodeCount; j++ {
			// Random resource type
			resourceType := resourceTypes[rand.Intn(len(resourceTypes))]

			record := models.NewRecord(resourceNodeCollection)
			record.Set("planet_id", planet.Id)
			record.Set("resource_type", resourceType.Id)
			record.Set("richness", rand.Intn(10)+1) // Richness 1-10
			record.Set("exhausted", false)

			if err := app.Dao().SaveRecord(record); err != nil {
				return err
			}
		}
	}

	return nil
}
