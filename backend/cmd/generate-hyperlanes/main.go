package main

import (
	"fmt"
	"log"
	"math"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

func main() {
	app := pocketbase.New()

	// Initialize the app
	if err := app.Bootstrap(); err != nil {
		log.Fatal(err)
	}

	// Generate hyperlanes
	if err := generateHyperlanes(app); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Hyperlanes generated successfully!")
}

func generateHyperlanes(app core.App) error {
	// Get all systems
	systems, err := app.Dao().FindRecordsByFilter("systems", "id != ''", "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch systems: %w", err)
	}

	fmt.Printf("Generating hyperlanes for %d systems...\n", len(systems))

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

	const maxDistance = 800.0
	const minDistance = 100.0  // Prevent too many short connections
	hyperlaneCount := 0

	// Generate hyperlanes using strategic network topology
	// Phase 1: Create main highway connections (longer distances)
	for i := 0; i < len(systems); i++ {
		system1 := systems[i]
		x1 := system1.GetFloat("x")
		y1 := system1.GetFloat("y")

		// Find strategic long-distance connections first
		var longCandidates []struct {
			system   *models.Record
			distance float64
			angle    float64
		}

		for j := 0; j < len(systems); j++ {
			if i == j {
				continue
			}

			system2 := systems[j]
			x2 := system2.GetFloat("x")
			y2 := system2.GetFloat("y")

			deltaX := x2 - x1
			deltaY := y2 - y1
			distance := math.Sqrt(deltaX*deltaX + deltaY*deltaY)
			angle := math.Atan2(deltaY, deltaX)

			// Focus on medium to long distance connections for highways
			if distance >= 400 && distance <= maxDistance {
				longCandidates = append(longCandidates, struct {
					system   *models.Record
					distance float64
					angle    float64
				}{system2, distance, angle})
			}
		}

		// Sort by distance (prefer medium-long distances)
		for i := 0; i < len(longCandidates)-1; i++ {
			for j := i + 1; j < len(longCandidates); j++ {
				if longCandidates[i].distance > longCandidates[j].distance {
					longCandidates[i], longCandidates[j] = longCandidates[j], longCandidates[i]
				}
			}
		}

		// Create 1-2 strategic long connections per system
		longConnections := 0
		maxLongConnections := 2

		for _, candidate := range longCandidates {
			if longConnections >= maxLongConnections {
				break
			}

			// Check if hyperlane already exists
			exists, err := hyperlaneExists(app, system1.Id, candidate.system.Id)
			if err != nil {
				return fmt.Errorf("failed to check hyperlane existence: %w", err)
			}

			if !exists {
				hyperlane := models.NewRecord(hyperlanesCollection)
				hyperlane.Set("from_system", system1.Id)
				hyperlane.Set("to_system", candidate.system.Id)
				hyperlane.Set("distance", candidate.distance)

				if err := app.Dao().SaveRecord(hyperlane); err != nil {
					return fmt.Errorf("failed to save hyperlane: %w", err)
				}

				hyperlaneCount++
				longConnections++
			}
		}
	}

	// Phase 2: Fill in local connections for connectivity
	for i := 0; i < len(systems); i++ {
		system1 := systems[i]
		x1 := system1.GetFloat("x")
		y1 := system1.GetFloat("y")

		// Count existing connections
		existingConnections := countSystemConnections(app, system1.Id)
		if existingConnections >= 3 {
			continue // System already well connected
		}

		// Find short-range candidates for local connectivity
		var localCandidates []struct {
			system   *models.Record
			distance float64
		}

		for j := 0; j < len(systems); j++ {
			if i == j {
				continue
			}

			system2 := systems[j]
			x2 := system2.GetFloat("x")
			y2 := system2.GetFloat("y")

			deltaX := x2 - x1
			deltaY := y2 - y1
			distance := math.Sqrt(deltaX*deltaX + deltaY*deltaY)

			// Focus on shorter connections for local networks
			if distance >= minDistance && distance <= 400 {
				localCandidates = append(localCandidates, struct {
					system   *models.Record
					distance float64
				}{system2, distance})
			}
		}

		// Sort by distance (closest first for local connections)
		for i := 0; i < len(localCandidates)-1; i++ {
			for j := i + 1; j < len(localCandidates); j++ {
				if localCandidates[i].distance > localCandidates[j].distance {
					localCandidates[i], localCandidates[j] = localCandidates[j], localCandidates[i]
				}
			}
		}

		// Add local connections to ensure minimum connectivity
		neededConnections := 3 - existingConnections
		localConnections := 0

		for _, candidate := range localCandidates {
			if localConnections >= neededConnections {
				break
			}

			exists, err := hyperlaneExists(app, system1.Id, candidate.system.Id)
			if err != nil {
				return fmt.Errorf("failed to check hyperlane existence: %w", err)
			}

			if !exists {
				hyperlane := models.NewRecord(hyperlanesCollection)
				hyperlane.Set("from_system", system1.Id)
				hyperlane.Set("to_system", candidate.system.Id)
				hyperlane.Set("distance", candidate.distance)

				if err := app.Dao().SaveRecord(hyperlane); err != nil {
					return fmt.Errorf("failed to save hyperlane: %w", err)
				}

				hyperlaneCount++
				localConnections++
			}
		}
	}

	fmt.Printf("Generated %d hyperlanes\n", hyperlaneCount)
	return nil
}

func hyperlaneExists(app core.App, system1Id, system2Id string) (bool, error) {
	// Check both directions since hyperlanes are bidirectional
	filter1 := fmt.Sprintf("from_system='%s' && to_system='%s'", system1Id, system2Id)
	filter2 := fmt.Sprintf("from_system='%s' && to_system='%s'", system2Id, system1Id)

	records1, err := app.Dao().FindRecordsByFilter("hyperlanes", filter1, "", 1, 0)
	if err != nil {
		return false, err
	}

	records2, err := app.Dao().FindRecordsByFilter("hyperlanes", filter2, "", 1, 0)
	if err != nil {
		return false, err
	}

	return len(records1) > 0 || len(records2) > 0, nil
}

func countSystemConnections(app core.App, systemId string) int {
	filter := fmt.Sprintf("from_system='%s' || to_system='%s'", systemId, systemId)
	records, err := app.Dao().FindRecordsByFilter("hyperlanes", filter, "", 0, 0)
	if err != nil {
		return 0
	}
	return len(records)
}

func wouldCrossExisting(app core.App, system1, system2 *models.Record) bool {
	// Get all existing hyperlanes
	existingHyperlanes, err := app.Dao().FindRecordsByFilter("hyperlanes", "id != ''", "", 0, 0)
	if err != nil {
		return false
	}

	x1, y1 := system1.GetFloat("x"), system1.GetFloat("y")
	x2, y2 := system2.GetFloat("x"), system2.GetFloat("y")

	// Check if the proposed hyperlane would cross existing ones
	for _, existing := range existingHyperlanes {
		// Get systems for existing hyperlane
		fromSystem, err1 := app.Dao().FindRecordById("systems", existing.GetString("from_system"))
		toSystem, err2 := app.Dao().FindRecordById("systems", existing.GetString("to_system"))
		
		if err1 != nil || err2 != nil {
			continue
		}

		x3, y3 := fromSystem.GetFloat("x"), fromSystem.GetFloat("y")
		x4, y4 := toSystem.GetFloat("x"), toSystem.GetFloat("y")

		// Check if lines intersect using cross product method
		if linesIntersect(x1, y1, x2, y2, x3, y3, x4, y4) {
			return true
		}
	}

	return false
}

func linesIntersect(x1, y1, x2, y2, x3, y3, x4, y4 float64) bool {
	// Calculate cross products
	d1 := direction(x3, y3, x4, y4, x1, y1)
	d2 := direction(x3, y3, x4, y4, x2, y2)
	d3 := direction(x1, y1, x2, y2, x3, y3)
	d4 := direction(x1, y1, x2, y2, x4, y4)

	// Lines intersect if they straddle each other
	if ((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) && 
	   ((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0)) {
		return true
	}

	// Skip collinear cases to keep it simple
	return false
}

func direction(px, py, qx, qy, rx, ry float64) float64 {
	return (qx-px)*(ry-py) - (qy-py)*(rx-px)
}