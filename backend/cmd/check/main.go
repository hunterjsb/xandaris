package main

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	_ "github.com/hunterjsb/xandaris/migrations"
)

func main() {
	app := pocketbase.New()

	if err := app.Bootstrap(); err != nil {
		log.Fatal(err)
	}

	if err := checkColonies(app); err != nil {
		log.Fatal(err)
	}
}

func checkColonies(app *pocketbase.PocketBase) error {
	// Get all planets
	allPlanets, err := app.Dao().FindRecordsByExpr("planets", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get all planets: %w", err)
	}

	// Get colonized planets
	colonizedPlanets, err := app.Dao().FindRecordsByFilter("planets", "colonized_by != ''", "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to get colonized planets: %w", err)
	}

	// Get all users
	users, err := app.Dao().FindRecordsByExpr("users", nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	fmt.Printf("=== COLONY DISTRIBUTION ===\n")
	fmt.Printf("Total planets: %d\n", len(allPlanets))
	fmt.Printf("Colonized planets: %d\n", len(colonizedPlanets))
	fmt.Printf("Uncolonized planets: %d\n", len(allPlanets)-len(colonizedPlanets))
	fmt.Printf("Total users: %d\n", len(users))
	fmt.Printf("\n")

	// Count colonies per user
	colonyCount := make(map[string]int)
	userNames := make(map[string]string)

	for _, user := range users {
		userNames[user.Id] = user.GetString("username")
		colonyCount[user.Id] = 0
	}

	for _, planet := range colonizedPlanets {
		ownerID := planet.GetString("colonized_by")
		if ownerID != "" {
			colonyCount[ownerID]++
		}
	}

	fmt.Printf("=== COLONIES PER USER ===\n")
	for userID, count := range colonyCount {
		username := userNames[userID]
		if username == "" {
			username = "Unknown"
		}
		fmt.Printf("User %s: %d colonies\n", username, count)
	}

	// Check populations
	populations, err := app.Dao().FindRecordsByExpr("populations", nil, nil)
	if err == nil {
		fmt.Printf("\nTotal population records: %d\n", len(populations))
	}

	// Check buildings
	buildings, err := app.Dao().FindRecordsByExpr("buildings", nil, nil)
	if err == nil {
		fmt.Printf("Total building records: %d\n", len(buildings))
	}

	fmt.Printf("\n=== RECOMMENDATION ===\n")
	if len(colonizedPlanets) > 20 {
		fmt.Printf("⚠️  WARNING: %d colonized planets is too many for good gameplay!\n", len(colonizedPlanets))
		fmt.Printf("💡 Recommend: Reset to 2-5 colonies per user for realistic 4X gameplay\n")
	} else {
		fmt.Printf("✅ Colony count looks reasonable for 4X gameplay\n")
	}

	return nil
}