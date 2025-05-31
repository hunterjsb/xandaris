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
	sysCollection, err := app.Dao().FindCollectionByNameOrId("systems")
	if err != nil {
		return fmt.Errorf("systems collection not found: %w", err)
	}
	planetCollection, err := app.Dao().FindCollectionByNameOrId("planets")
	if err != nil {
		return fmt.Errorf("planets collection not found: %w", err)
	}

	for i, sys := range systems {
		systemRecord := models.NewRecord(sysCollection)
		systemRecord.Set("x", sys.X)
		systemRecord.Set("y", sys.Y)
		systemRecord.Set("richness", sys.Richness)

		if err := app.Dao().SaveRecord(systemRecord); err != nil {
			return fmt.Errorf("failed to save system: %w", err)
		}

		// create a default planet for each system
		planet := models.NewRecord(planetCollection)
		planet.Set("name", fmt.Sprintf("Planet-%d", i+1))
		planet.Set("system_id", systemRecord.Id)
		planet.Set("type_id", "") // default type will be assigned later
		if err := app.Dao().SaveRecord(planet); err != nil {
			return fmt.Errorf("failed to save planet: %w", err)
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
	// delete planets first
	planets, _ := app.Dao().FindRecordsByFilter("planets", "", "", 100, 0)
	for _, p := range planets {
		_ = app.Dao().DeleteRecord(p)
	}

	// delete systems
	systems, err := app.Dao().FindRecordsByFilter("systems", "", "", 50, 0)
	if err != nil {
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
		}
	}

	// fetch planets
	planetRecords, _ := app.Dao().FindRecordsByFilter("planets", "", "", 0, 0)
	planetsData := make([]map[string]interface{}, len(planetRecords))
	for i, p := range planetRecords {
		planetsData[i] = map[string]interface{}{
			"id":        p.Id,
			"name":      p.GetString("name"),
			"system_id": p.GetString("system_id"),
			"type_id":   p.GetString("type_id"),
		}
	}

	// Generate lanes (connections between nearby systems)
	lanes := generateLanes(systemsData)

	return map[string]interface{}{
		"systems": systemsData,
		"planets": planetsData,
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
