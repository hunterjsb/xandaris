package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	_ "github.com/hunterjsb/xandaris/migrations"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: util <command>")
		fmt.Println("Commands:")
		fmt.Println("  check    - Check colony distribution")
		fmt.Println("  reset    - Reset colonies to realistic numbers")
		fmt.Println("  clean    - Clean all game data")
		fmt.Println("  setup    - Create starter fleets for existing users")
		os.Exit(1)
	}

	app := pocketbase.New()
	if err := app.Bootstrap(); err != nil {
		log.Fatal(err)
	}

	command := os.Args[1]
	switch command {
	case "check":
		checkColonies(app)
	case "reset":
		resetColonies(app)
	case "clean":
		cleanGameData(app)
	case "setup":
		setupStarterFleets(app)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func checkColonies(app *pocketbase.PocketBase) {
	fmt.Println("=== COLONY CHECK ===")
	
	// Get counts
	allPlanets, _ := app.Dao().FindRecordsByExpr("planets", nil, nil)
	users, _ := app.Dao().FindRecordsByExpr("_pb_users_auth_", nil, nil)
	populations, _ := app.Dao().FindRecordsByExpr("populations", nil, nil)
	buildings, _ := app.Dao().FindRecordsByExpr("buildings", nil, nil)

	// Count colonized planets directly in Go instead of using PocketBase filter
	colonizedCount := 0
	for _, planet := range allPlanets {
		colonizedBy := planet.GetString("colonized_by")
		if colonizedBy != "" {
			colonizedCount++
		}
	}

	fmt.Printf("Total planets: %d\n", len(allPlanets))
	fmt.Printf("Colonized planets: %d\n", colonizedCount)
	fmt.Printf("Users: %d\n", len(users))
	fmt.Printf("Population records: %d\n", len(populations))
	fmt.Printf("Building records: %d\n", len(buildings))

	if colonizedCount > 20 {
		fmt.Printf("\n⚠️  WARNING: %d colonized planets is too many!\n", colonizedCount)
		fmt.Println("💡 Run 'util reset' to fix this")
	} else {
		fmt.Println("✅ Colony distribution looks reasonable")
	}
}

func resetColonies(app *pocketbase.PocketBase) {
	fmt.Println("=== RESETTING COLONIES ===")
	rand.Seed(time.Now().UnixNano())

	// Clear existing game data
	fmt.Println("Clearing existing populations and buildings...")
	if populations, err := app.Dao().FindRecordsByExpr("populations", nil, nil); err == nil {
		for _, pop := range populations {
			app.Dao().DeleteRecord(pop)
		}
	}
	if buildings, err := app.Dao().FindRecordsByExpr("buildings", nil, nil); err == nil {
		for _, building := range buildings {
			app.Dao().DeleteRecord(building)
		}
	}

	// Reset all planets to uncolonized
	fmt.Println("Resetting planets to uncolonized...")
	if planets, err := app.Dao().FindRecordsByExpr("planets", nil, nil); err == nil {
		for _, planet := range planets {
			planet.Set("colonized_by", nil)
			planet.Set("colonized_at", nil)
			app.Dao().SaveRecord(planet)
		}
	}

	// Get users and create realistic colonies
	users, err := app.Dao().FindRecordsByExpr("users", nil, nil)
	if err != nil {
		fmt.Println("No users found")
		return
	}

	totalColonized := 0
	for _, user := range users {
		username := user.GetString("username")
		coloniesPerUser := 3 // Fixed 3 colonies per user

		// Get first few uncolonized planets
		planets, err := app.Dao().FindRecordsByFilter("planets", "colonized_by = ''", "", coloniesPerUser, 0)
		if err != nil || len(planets) == 0 {
			continue
		}

		for i := 0; i < coloniesPerUser && i < len(planets); i++ {
			planet := planets[i]
			
			// Colonize planet
			planet.Set("colonized_by", user.Id)
			planet.Set("colonized_at", time.Now())
			app.Dao().SaveRecord(planet)

			// Add population
			createPopulation(app, planet.Id, user.Id)

			// Add 2 buildings
			createBuildings(app, planet.Id, 2)
			
			totalColonized++
		}

		fmt.Printf("User %s: %d colonies\n", username, coloniesPerUser)
	}

	fmt.Printf("\n✅ Reset complete: %d total colonized planets\n", totalColonized)
}

func cleanGameData(app *pocketbase.PocketBase) {
	fmt.Println("=== CLEANING GAME DATA ===")
	fmt.Println("ℹ️  Preserving all user accounts and auth settings")
	
	// First, show how many users we're preserving
	if users, err := app.Dao().FindRecordsByExpr("users", nil, nil); err == nil {
		fmt.Printf("✅ Preserving %d user accounts\n", len(users))
		for _, user := range users {
			username := user.GetString("username")
			email := user.GetString("email")
			fmt.Printf("  - %s (%s)\n", username, email)
		}
	}
	
	fmt.Println("\n🗑️  Deleting game data...")
	
	// Delete collections in order respecting foreign key constraints
	// First delete child records, then parent records
	deleteOrder := []string{
		// Child records first
		"fleet_orders",     // references fleets and systems
		"ship_cargo",       // references ships and resource_types
		"resource_nodes",    // references planets
		"populations",       // references planets  
		"buildings",         // references planets
		"ships",            // references fleets
		"trade_routes",     // references systems
		"fleets",           // references systems
		"hyperlanes",       // references systems
		"treaties",
		"treaty_proposals", 
		"battle_logs",
		"colonies",
		// Parent records
		"planets",          // references systems and planet_types
		"systems",          // parent of planets
		// Type definitions (will be recreated by seed)
		"resource_types",
		"planet_types", 
		"building_types",
		"ship_types",
	}
	
	for _, collName := range deleteOrder {
		if records, err := app.Dao().FindRecordsByExpr(collName, nil, nil); err == nil {
			count := len(records)
			for _, record := range records {
				if err := app.Dao().DeleteRecord(record); err != nil {
					fmt.Printf("Error deleting %s record: %v\n", collName, err)
				}
			}
			if count > 0 {
				fmt.Printf("  Deleted %d %s records\n", count, collName)
			}
		}
	}

	fmt.Println("\n✅ Clean complete - all game data deleted")
	fmt.Println("✅ User accounts and authentication preserved")
	fmt.Println("💡 Run the seed command to regenerate the universe")
}

func createPopulation(app *pocketbase.PocketBase, planetID, ownerID string) {
	collection, err := app.Dao().FindCollectionByNameOrId("populations")
	if err != nil {
		return
	}

	population := models.NewRecord(collection)
	population.Set("owner_id", ownerID)
	population.Set("planet_id", planetID)
	population.Set("count", rand.Intn(100)+100)
	population.Set("happiness", rand.Intn(20)+80)
	app.Dao().SaveRecord(population)
}

func createBuildings(app *pocketbase.PocketBase, planetID string, count int) {
	buildingTypes, err := app.Dao().FindRecordsByExpr("building_types", nil, nil)
	if err != nil || len(buildingTypes) == 0 {
		return
	}

	collection, err := app.Dao().FindCollectionByNameOrId("buildings")
	if err != nil {
		return
	}

	for i := 0; i < count && i < len(buildingTypes); i++ {
		building := models.NewRecord(collection)
		building.Set("planet_id", planetID)
		building.Set("building_type", buildingTypes[i].Id)
		building.Set("level", 1)
		building.Set("active", true)
		app.Dao().SaveRecord(building)
	}
}

func setupStarterFleets(app *pocketbase.PocketBase) {
	fmt.Println("=== SETTING UP STARTER FLEETS ===")
	fmt.Println("🚀 Creating starter fleets for existing users")

	// Get all users
	users, err := app.Dao().FindRecordsByExpr("users", nil, nil)
	if err != nil {
		fmt.Printf("❌ Failed to get users: %v\n", err)
		return
	}

	fmt.Printf("📋 Found %d users\n", len(users))

	for _, user := range users {
		// Check if user already has a fleet
		existingFleets, err := app.Dao().FindRecordsByFilter("fleets", fmt.Sprintf("owner_id='%s'", user.Id), "", 1, 0)
		if err == nil && len(existingFleets) > 0 {
			fmt.Printf("⏭️  User %s already has fleet, skipping\n", user.GetString("username"))
			continue
		}

		fmt.Printf("🆕 Setting up starter resources for user: %s\n", user.GetString("username"))

		// Create starter fleet
		fleetsCollection, err := app.Dao().FindCollectionByNameOrId("fleets")
		if err != nil {
			fmt.Printf("❌ Fleets collection not found: %v\n", err)
			continue
		}

		fleet := models.NewRecord(fleetsCollection)
		fleet.Set("owner_id", user.Id)
		fleet.Set("name", "Settler Fleet")
		
		// Find a random system for starting location
		systems, err := app.Dao().FindRecordsByExpr("systems", nil, nil)
		if err != nil || len(systems) == 0 {
			fmt.Printf("❌ No systems found for starting location\n")
			continue
		}
		fleet.Set("current_system", systems[0].Id)

		if err := app.Dao().SaveRecord(fleet); err != nil {
			fmt.Printf("❌ Failed to create starter fleet: %v\n", err)
			continue
		}

		// Get settler ship type
		settlerShipType, err := app.Dao().FindFirstRecordByFilter("ship_types", "name='settler'")
		if err != nil {
			fmt.Printf("❌ Settler ship type not found: %v\n", err)
			continue
		}

		// Create settler ship
		shipsCollection, err := app.Dao().FindCollectionByNameOrId("ships")
		if err != nil {
			fmt.Printf("❌ Ships collection not found: %v\n", err)
			continue
		}

		ship := models.NewRecord(shipsCollection)
		ship.Set("fleet_id", fleet.Id)
		ship.Set("ship_type", settlerShipType.Id)
		ship.Set("count", 1)
		ship.Set("health", 100)

		if err := app.Dao().SaveRecord(ship); err != nil {
			fmt.Printf("❌ Failed to create starter ship: %v\n", err)
			continue
		}

		// Add 50 ore to ship cargo
		oreResource, err := app.Dao().FindFirstRecordByFilter("resource_types", "name='ore'")
		if err != nil {
			fmt.Printf("❌ Ore resource type not found: %v\n", err)
			continue
		}

		cargoCollection, err := app.Dao().FindCollectionByNameOrId("ship_cargo")
		if err != nil {
			fmt.Printf("❌ Ship_cargo collection not found: %v\n", err)
			continue
		}

		cargo := models.NewRecord(cargoCollection)
		cargo.Set("ship_id", ship.Id)
		cargo.Set("resource_type", oreResource.Id)
		cargo.Set("quantity", 50)

		if err := app.Dao().SaveRecord(cargo); err != nil {
			fmt.Printf("❌ Failed to create starter cargo: %v\n", err)
			continue
		}

		fmt.Printf("✅ Created starter fleet '%s' with settler ship and 50 ore for user %s\n", 
			fleet.GetString("name"), user.GetString("username"))
	}

	fmt.Println("🎉 Starter fleet setup completed!")
}