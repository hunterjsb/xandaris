package main

import (
	"fmt"
	"log"
	"math"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	mapgen "github.com/hunterjsb/xandaris/internal/map"
	_ "github.com/hunterjsb/xandaris/migrations"
)

func main() {
	app := pocketbase.New()

	if err := app.Bootstrap(); err != nil {
		log.Fatal(err)
	}

	if err := convertLanesToHyperlanes(app); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully converted map lanes to hyperlanes!")
}

func convertLanesToHyperlanes(app *pocketbase.PocketBase) error {
	// Get map data which includes the generated lanes
	mapData, err := mapgen.GetMapData(app)
	if err != nil {
		return fmt.Errorf("failed to get map data: %w", err)
	}

	lanes, ok := mapData["lanes"].([]map[string]interface{})
	if !ok {
		return fmt.Errorf("lanes not found in map data")
	}

	fmt.Printf("Found %d lanes to convert to hyperlanes\n", len(lanes))

	// Clear existing hyperlanes
	existingHyperlanes, err := app.Dao().FindRecordsByFilter("hyperlanes", "id != ''", "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch existing hyperlanes: %w", err)
	}

	for _, hyperlane := range existingHyperlanes {
		if err := app.Dao().DeleteRecord(hyperlane); err != nil {
			return fmt.Errorf("failed to delete existing hyperlane: %w", err)
		}
	}

	// Get hyperlanes collection
	hyperlanesCollection, err := app.Dao().FindCollectionByNameOrId("hyperlanes")
	if err != nil {
		return fmt.Errorf("failed to find hyperlanes collection: %w", err)
	}

	// Convert each lane to a hyperlane
	hyperlaneCount := 0
	for _, lane := range lanes {
		fromXInt, ok1 := lane["fromX"].(int)
		fromYInt, ok2 := lane["fromY"].(int)
		toXInt, ok3 := lane["toX"].(int)
		toYInt, ok4 := lane["toY"].(int)

		if !ok1 || !ok2 || !ok3 || !ok4 {
			continue // Skip invalid lane data
		}

		fromX := float64(fromXInt)
		fromY := float64(fromYInt)
		toX := float64(toXInt)
		toY := float64(toYInt)

		// Find systems at these coordinates
		fromSystem, err := findSystemAtCoordinates(app, fromX, fromY)
		if err != nil {
			continue
		}

		toSystem, err := findSystemAtCoordinates(app, toX, toY)
		if err != nil {
			continue
		}

		// Calculate distance
		deltaX := toX - fromX
		deltaY := toY - fromY
		distance := math.Sqrt(deltaX*deltaX + deltaY*deltaY)

		// Create hyperlane record
		hyperlane := models.NewRecord(hyperlanesCollection)
		hyperlane.Set("from_system", fromSystem.Id)
		hyperlane.Set("to_system", toSystem.Id)
		hyperlane.Set("distance", distance)

		if err := app.Dao().SaveRecord(hyperlane); err != nil {
			return fmt.Errorf("failed to save hyperlane: %w", err)
		}

		hyperlaneCount++
	}

	fmt.Printf("Successfully converted %d lanes to hyperlanes\n", hyperlaneCount)
	return nil
}

func findSystemAtCoordinates(app *pocketbase.PocketBase, x, y float64) (*models.Record, error) {
	systems, err := app.Dao().FindRecordsByFilter("systems", "id != ''", "", 0, 0)
	if err != nil {
		return nil, err
	}

	// Find system with matching coordinates (with small tolerance for floating point)
	tolerance := 1.0
	for _, system := range systems {
		sysX := system.GetFloat("x")
		sysY := system.GetFloat("y")

		if math.Abs(sysX-x) < tolerance && math.Abs(sysY-y) < tolerance {
			return system, nil
		}
	}

	return nil, fmt.Errorf("no system found at coordinates (%f, %f)", x, y)
}