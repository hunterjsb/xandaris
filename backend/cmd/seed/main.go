package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	mapgen "github.com/hunterjsb/xandaris/internal/map"
	"github.com/hunterjsb/xandaris/internal/worldgen"
	_ "github.com/hunterjsb/xandaris/migrations"
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

	// 5. Generate spiral galaxy with systems and planets
	if err := mapgen.GenerateMap(app, 200); err != nil {
		return fmt.Errorf("failed to generate galaxy map: %w", err)
	}

	// 7. Generate resource nodes for planets
	if err := seedResourceNodes(app); err != nil {
		return fmt.Errorf("failed to seed resource nodes: %w", err)
	}

	// 8. Create sample user and colonize some planets (optional)
	if err := seedSampleUserAndColonies(app); err != nil {
		return fmt.Errorf("failed to seed sample user and colonies: %w", err)
	}

	fmt.Println("✅ Universe generation complete:")
	fmt.Printf("  - Resource types, planet types, building types, ship types created\n")
	fmt.Printf("  - Galaxy map generated with systems and planets\n")
	fmt.Printf("  - Resource nodes distributed across planets\n")

	return nil
}



func seedSampleUserAndColonies(app *pocketbase.PocketBase) error {
	// Get all existing users
	allUsers, err := app.Dao().FindRecordsByExpr("_pb_users_auth_", nil, nil)
	if err != nil {
		return err
	}

	if len(allUsers) == 0 {
		fmt.Println("No users found in the database. Skipping colony creation.")
		fmt.Println("✅ Universe seeded successfully without user colonies")
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
		userID := user.Id
		userEmail := user.GetString("email")
		if userEmail == "" {
			userEmail = "no-email"
		}

		// Check how many planets this user already has colonized
		existingColonies, err := app.Dao().FindRecordsByFilter("planets", "colonized_by = '"+user.Id+"'", "", 0, 0)
		if err != nil {
			return err
		}

		if len(existingColonies) >= coloniesPerUser {
			fmt.Printf("  %s (%s) already has %d colonies\n", userID, userEmail, len(existingColonies))
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
		fmt.Printf("  %s (%s) now has %d colonies\n", userID, userEmail, totalColonies)
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
		{"name": "food", "description": "Sustains population growth and survival", "produced_in": "farm"},
		{"name": "ore", "description": "Basic raw material used in metal production and construction.", "produced_in": "mine"},
		{"name": "fuel", "description": "Energy source for ships and industry", "produced_in": "refinery"},
		{"name": "titanium", "description": "High-grade material for advanced construction and ships", "produced_in": "deep_mine"},
		{"name": "metal", "description": "Building material for ships and construction", "produced_in": "refinery"},
		{"name": "oil", "description": "Essential for fuel production and ship manufacturing", "produced_in": "oil_rig"},
		{"name": "xanium", "description": "Most lit material", "produced_in": "Shady Alley"},
		{"name": "credits", "description": "Standard currency used exclusively for player-to-player trading", "produced_in": "crypto_server"},
		{"name": "all", "description": "placeholder for storage", "produced_in": "none"},
	}

	for _, resource := range resources {
		record := models.NewRecord(collection)
		record.Set("name", resource["name"])
		record.Set("description", resource["description"])
		record.Set("produced_in", resource["produced_in"])
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
		{"name": "Highlands", "spawn_prob": 0.038, "icon": "/icons/terrain.svg"},
		{"name": "Abundant", "spawn_prob": 0.013, "icon": "/icons/eco.svg"},
		{"name": "Fertile", "spawn_prob": 0.05, "icon": "/icons/grass.svg"},
		{"name": "Mountain", "spawn_prob": 0.05, "icon": "/icons/landscape.svg"},
		{"name": "Desert", "spawn_prob": 0.025, "icon": "/icons/wb_sunny.svg"},
		{"name": "Volcanic", "spawn_prob": 0.025, "icon": "/icons/whatshot.svg"},
		{"name": "Swamp", "spawn_prob": 0.038, "icon": "/icons/water.svg"},
		{"name": "Barren", "spawn_prob": 0.005, "icon": "/icons/circle.svg"},
		{"name": "Radiant", "spawn_prob": 0.005, "icon": "/icons/flare.svg"},
		{"name": "Barred", "spawn_prob": 0.001, "icon": "/icons/block.svg"},
	}

	for _, planetType := range planetTypes {
		record := models.NewRecord(collection)
		record.Set("name", planetType["name"])
		record.Set("spawn_prob", planetType["spawn_prob"])
		record.Set("icon", planetType["icon"])
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

	// Get resource types for relationships
	resourceTypes, err := app.Dao().FindRecordsByExpr("resource_types", nil, nil)
	if err != nil {
		return err
	}

	// Create a map for easy lookup
	resourceMap := make(map[string]string)
	for _, resource := range resourceTypes {
		resourceMap[resource.GetString("name")] = resource.Id
	}

	buildingTypes := []map[string]interface{}{
		{
			"name": "farm",
			"cost": 100,
			"strength": "weak",
			"power_consumption": 10,
			"res1_type": "food",
			"res1_quantity": 5,
			"res1_capacity": 100,
			"res2_quantity": 0,
			"res2_capacity": 0,
			"description": "",
			"node_requirement": "",
		},
		{
			"name": "mine",
			"cost": 100,
			"strength": "strong",
			"power_consumption": 20,
			"res1_type": "ore",
			"res1_quantity": 12,
			"res1_capacity": 100,
			"res2_quantity": 0,
			"res2_capacity": 0,
			"description": "",
			"node_requirement": "",
		},
		{
			"name": "metal_refinery",
			"cost": 200,
			"strength": "na",
			"power_consumption": 50,
			"res1_type": "metal",
			"res1_quantity": 18,
			"res1_capacity": 100,
			"res2_type": "ore",
			"res2_quantity": -24,
			"res2_capacity": 100,
			"description": "",
			"node_requirement": "",
		},
		{
			"name": "power_plant",
			"cost": 500,
			"strength": "na",
			"power_consumption": -100,
			"res1_type": "fuel",
			"res1_quantity": -1,
			"res1_capacity": 100,
			"res2_quantity": 0,
			"res2_capacity": 0,
			"description": "",
			"node_requirement": "",
		},
		{
			"name": "spaceport",
			"cost": 1000,
			"strength": "na",
			"power_consumption": 100,
			"res1_quantity": 0,
			"res1_capacity": 0,
			"res2_quantity": 0,
			"res2_capacity": 0,
			"description": "",
			"node_requirement": "",
		},
		{
			"name": "oil_rig",
			"cost": 200,
			"strength": "weak",
			"power_consumption": 25,
			"res1_type": "oil",
			"res1_quantity": 5,
			"res1_capacity": 100,
			"res2_quantity": 0,
			"res2_capacity": 0,
			"description": "",
			"node_requirement": "",
		},
		{
			"name": "deep_mine",
			"cost": 500,
			"strength": "na",
			"power_consumption": 50,
			"res1_type": "titanium",
			"res1_quantity": 5,
			"res1_capacity": 100,
			"res2_type": "oil",
			"res2_quantity": -1,
			"res2_capacity": 100,
			"description": "",
			"node_requirement": "",
		},
		{
			"name": "crypto_server",
			"cost": 1000,
			"strength": "na",
			"power_consumption": 50,
			"res1_type": "credits",
			"res1_quantity": 1,
			"res1_capacity": 10000,
			"res2_quantity": 0,
			"res2_capacity": 0,
			"description": "",
			"node_requirement": "",
		},
		{
			"name": "shady_alley",
			"cost": 5000,
			"strength": "weak",
			"power_consumption": 200,
			"res1_type": "xanium",
			"res1_quantity": 1,
			"res1_capacity": 20,
			"res2_type": "food",
			"res2_quantity": -20,
			"res2_capacity": 200,
			"description": "",
			"node_requirement": "",
		},
		{
			"name": "oil_refinery",
			"cost": 200,
			"strength": "na",
			"power_consumption": 50,
			"res1_type": "fuel",
			"res1_quantity": 18,
			"res1_capacity": 100,
			"res2_type": "oil",
			"res2_quantity": -24,
			"res2_capacity": 100,
			"description": "",
			"node_requirement": "",
		},
		{
			"name": "base",
			"cost": 0,
			"strength": "na",
			"power_consumption": -100,
			"res1_type": "all",
			"res1_quantity": 0,
			"res1_capacity": 100,
			"res2_type": "xanium",
			"res2_quantity": 0,
			"res2_capacity": 5,
			"description": "",
			"node_requirement": "",
		},
		{
			"name": "storage_depo",
			"cost": 200,
			"strength": "na",
			"power_consumption": 0,
			"res1_type": "all",
			"res1_quantity": 0,
			"res1_capacity": 1000,
			"res2_quantity": 0,
			"res2_capacity": 0,
			"description": "",
			"node_requirement": "",
		},
	}

	for _, building := range buildingTypes {
		record := models.NewRecord(collection)
		record.Set("name", building["name"])
		record.Set("cost", building["cost"])
		record.Set("strength", building["strength"])
		record.Set("power_consumption", building["power_consumption"])
		
		if res1Type, ok := building["res1_type"]; ok {
			if resourceID, exists := resourceMap[res1Type.(string)]; exists {
				record.Set("res1_type", resourceID)
			}
		}
		if res1Qty, ok := building["res1_quantity"]; ok {
			record.Set("res1_quantity", res1Qty)
		}
		if res1Cap, ok := building["res1_capacity"]; ok {
			record.Set("res1_capacity", res1Cap)
		}
		
		if res2Type, ok := building["res2_type"]; ok {
			if resourceID, exists := resourceMap[res2Type.(string)]; exists {
				record.Set("res2_type", resourceID)
			}
		}
		if res2Qty, ok := building["res2_quantity"]; ok {
			record.Set("res2_quantity", res2Qty)
		}
		if res2Cap, ok := building["res2_capacity"]; ok {
			record.Set("res2_capacity", res2Cap)
		}
		
		record.Set("description", building["description"])
		record.Set("node_requirement", building["node_requirement"])
		
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
		{"name": "settler", "cost": 100, "strength": 1, "cargo_capacity": 50},
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
	fmt.Println("🌍 Generating enhanced resource nodes using world generation system...")
	
	// Use worldgen to generate all resource nodes based on planet types
	if err := worldgen.GenerateResourceNodesForAllPlanets(app); err != nil {
		return fmt.Errorf("worldgen resource generation failed: %w", err)
	}
	
	fmt.Println("✅ Enhanced resource node generation complete")
	return nil
}
