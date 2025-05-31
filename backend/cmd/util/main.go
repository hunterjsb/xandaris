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
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func checkColonies(app *pocketbase.PocketBase) {
	fmt.Println("=== COLONY CHECK ===")
	
	// Get counts
	allPlanets, _ := app.Dao().FindRecordsByExpr("planets", nil, nil)
	users, _ := app.Dao().FindRecordsByExpr("users", nil, nil)
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
	fmt.Println("=== CLEANING ALL GAME DATA ===")
	
	collections := []string{"populations", "buildings", "ships", "fleets", "trade_routes"}
	for _, collName := range collections {
		if records, err := app.Dao().FindRecordsByExpr(collName, nil, nil); err == nil {
			for _, record := range records {
				app.Dao().DeleteRecord(record)
			}
			fmt.Printf("Cleared %d %s records\n", len(records), collName)
		}
	}

	// Reset planets
	if planets, err := app.Dao().FindRecordsByExpr("planets", nil, nil); err == nil {
		for _, planet := range planets {
			planet.Set("colonized_by", nil)
			planet.Set("colonized_at", nil)
			app.Dao().SaveRecord(planet)
		}
		fmt.Printf("Reset %d planets to uncolonized\n", len(planets))
	}

	fmt.Println("✅ Clean complete")
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