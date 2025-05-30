package mapgen

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// GenerateMap creates a new galaxy with systems
func GenerateMap(app *pocketbase.PocketBase, systemCount int) error {
	rand.Seed(time.Now().UnixNano())

	// Clear existing systems
	if err := clearExistingSystems(app); err != nil {
		return fmt.Errorf("failed to clear existing systems: %w", err)
	}

	// Generate systems
	systems := generateSystems(systemCount)

	// Save systems to database
	collection, err := app.Dao().FindCollectionByNameOrId("systems")
	if err != nil {
		return fmt.Errorf("systems collection not found: %w", err)
	}

	for _, sys := range systems {
		record := models.NewRecord(collection)
		record.Set("x", sys.X)
		record.Set("y", sys.Y)
		record.Set("richness", sys.Richness)
		record.Set("pop", 0)
		record.Set("morale", 0)
		record.Set("food", 0)
		record.Set("ore", 0)
		record.Set("goods", 0)
		record.Set("fuel", 0)
		record.Set("hab_lvl", 0)
		record.Set("farm_lvl", 0)
		record.Set("mine_lvl", 0)
		record.Set("fac_lvl", 0)
		record.Set("yard_lvl", 0)

		if err := app.Dao().SaveRecord(record); err != nil {
			return fmt.Errorf("failed to save system: %w", err)
		}
	}

	return nil
}

type System struct {
	X        int
	Y        int
	Richness int
}

func generateSystems(count int) []System {
	systems := make([]System, 0, count)
	
	// Use a simple grid with some randomization
	gridSize := int(math.Ceil(math.Sqrt(float64(count))))
	spacing := 200 // Distance between systems

	for i := 0; i < count; i++ {
		gridX := i % gridSize
		gridY := i / gridSize

		// Add randomization to grid positions
		offsetX := rand.Intn(spacing/2) - spacing/4
		offsetY := rand.Intn(spacing/2) - spacing/4

		x := gridX*spacing + offsetX
		y := gridY*spacing + offsetY

		// Generate richness (1-10, with bias toward middle values)
		richness := 3 + rand.Intn(5) // 3-7 base
		if rand.Float32() < 0.1 {
			richness = 1 + rand.Intn(2) // 10% chance of poor (1-2)
		} else if rand.Float32() < 0.1 {
			richness = 8 + rand.Intn(3) // 10% chance of rich (8-10)
		}

		systems = append(systems, System{
			X:        x,
			Y:        y,
			Richness: richness,
		})
	}

	return systems
}

func clearExistingSystems(app *pocketbase.PocketBase) error {
	// Try to find and delete existing systems
	// We'll just skip if the collection doesn't exist yet
	systems, err := app.Dao().FindRecordsByFilter("systems", "", "", 50, 0)
	if err != nil {
		// If systems collection doesn't exist, that's fine
		return nil
	}

	for _, system := range systems {
		if err := app.Dao().DeleteRecord(system); err != nil {
			return fmt.Errorf("failed to delete system %s: %w", system.Id, err)
		}
	}

	return nil
}

// GetMapData returns the current map state for the frontend
func GetMapData(app *pocketbase.PocketBase) (map[string]interface{}, error) {
	// Use a simple query to get all systems
	query := app.Dao().RecordQuery("systems")
	systems := []*models.Record{}
	
	if err := query.All(&systems); err != nil {
		return nil, fmt.Errorf("failed to fetch systems: %w", err)
	}

	systemsData := make([]map[string]interface{}, len(systems))
	for i, system := range systems {
		systemsData[i] = map[string]interface{}{
			"id":       system.Id,
			"x":        system.GetInt("x"),
			"y":        system.GetInt("y"),
			"richness": system.GetInt("richness"),
			"owner_id": system.GetString("owner_id"),
			"pop":      system.GetInt("pop"),
			"morale":   system.GetInt("morale"),
			"food":     system.GetInt("food"),
			"ore":      system.GetInt("ore"),
			"goods":    system.GetInt("goods"),
			"fuel":     system.GetInt("fuel"),
			"hab_lvl":  system.GetInt("hab_lvl"),
			"farm_lvl": system.GetInt("farm_lvl"),
			"mine_lvl": system.GetInt("mine_lvl"),
			"fac_lvl":  system.GetInt("fac_lvl"),
			"yard_lvl": system.GetInt("yard_lvl"),
		}
	}

	// Generate lanes (connections between nearby systems)
	lanes := generateLanes(systemsData)

	return map[string]interface{}{
		"systems": systemsData,
		"lanes":   lanes,
	}, nil
}

func generateLanes(systems []map[string]interface{}) []map[string]interface{} {
	lanes := make([]map[string]interface{}, 0)
	maxDistance := 300.0 // Maximum distance for lane connections

	for i, sys1 := range systems {
		x1 := float64(sys1["x"].(int))
		y1 := float64(sys1["y"].(int))

		for j, sys2 := range systems {
			if i >= j {
				continue // Avoid duplicate and self-connections
			}

			x2 := float64(sys2["x"].(int))
			y2 := float64(sys2["y"].(int))

			distance := math.Sqrt((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1))

			if distance <= maxDistance {
				lanes = append(lanes, map[string]interface{}{
					"from":     sys1["id"],
					"to":       sys2["id"],
					"fromX":    int(x1),
					"fromY":    int(y1),
					"toX":      int(x2),
					"toY":      int(y2),
					"distance": int(distance),
				})
			}
		}
	}

	return lanes
}