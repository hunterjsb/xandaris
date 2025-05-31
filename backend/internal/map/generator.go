package mapgen

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// GenerateMap creates a new galaxy with planets
func GenerateMap(app *pocketbase.PocketBase, planetCount int) error {
	rand.Seed(time.Now().UnixNano())

	// Clear existing planets
	if err := clearExistingPlanets(app); err != nil {
		return fmt.Errorf("failed to clear existing planets: %w", err)
	}

	// Generate planets
	planets := generatePlanets(planetCount)

	// Save planets to database
	collection, err := app.Dao().FindCollectionByNameOrId("planets")
	if err != nil {
		return fmt.Errorf("planets collection not found: %w", err)
	}

	for _, p := range planets {
		record := models.NewRecord(collection)
		record.Set("x", p.X)
		record.Set("y", p.Y)
		record.Set("richness", p.Richness)
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
			return fmt.Errorf("failed to save planet: %w", err)
		}
	}

	return nil
}

type Planet struct {
	X        int
	Y        int
	Richness int
}

func generatePlanets(count int) []Planet {
	planets := make([]Planet, 0, count)

	// Use a simple grid with some randomization
	gridSize := int(math.Ceil(math.Sqrt(float64(count))))
	spacing := 200 // Distance between planets

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

		planets = append(planets, Planet{
			X:        x,
			Y:        y,
			Richness: richness,
		})
	}

	return planets
}

func clearExistingPlanets(app *pocketbase.PocketBase) error {
	// Try to find and delete existing planets
	// We'll just skip if the collection doesn't exist yet
	planets, err := app.Dao().FindRecordsByFilter("planets", "", "", 50, 0)
	if err != nil {
		// If planets collection doesn't exist, that's fine
		return nil
	}

	for _, planet := range planets {
		if err := app.Dao().DeleteRecord(planet); err != nil {
			return fmt.Errorf("failed to delete planet %s: %w", planet.Id, err)
		}
	}

	return nil
}

// GetMapData returns the current map state for the frontend
func GetMapData(app *pocketbase.PocketBase) (map[string]interface{}, error) {
	// Use a simple query to get all planets
	query := app.Dao().RecordQuery("planets")
	planets := []*models.Record{}

	if err := query.All(&planets); err != nil {
		return nil, fmt.Errorf("failed to fetch planets: %w", err)
	}

	planetsData := make([]map[string]interface{}, len(planets))
	for i, planet := range planets {
		planetsData[i] = map[string]interface{}{
			"id":       planet.Id,
			"x":        planet.GetInt("x"),
			"y":        planet.GetInt("y"),
			"richness": planet.GetInt("richness"),
			"owner_id": planet.GetString("owner_id"),
			"pop":      planet.GetInt("pop"),
			"morale":   planet.GetInt("morale"),
			"food":     planet.GetInt("food"),
			"ore":      planet.GetInt("ore"),
			"goods":    planet.GetInt("goods"),
			"fuel":     planet.GetInt("fuel"),
			"hab_lvl":  planet.GetInt("hab_lvl"),
			"farm_lvl": planet.GetInt("farm_lvl"),
			"mine_lvl": planet.GetInt("mine_lvl"),
			"fac_lvl":  planet.GetInt("fac_lvl"),
			"yard_lvl": planet.GetInt("yard_lvl"),
		}
	}

	// Generate lanes (connections between nearby planets)
	lanes := generateLanes(planetsData)

	return map[string]interface{}{
		"planets": planetsData,
		"lanes":   lanes,
	}, nil
}

func generateLanes(planets []map[string]interface{}) []map[string]interface{} {
	lanes := make([]map[string]interface{}, 0)
	maxDistance := 300.0 // Maximum distance for lane connections

	for i, p1 := range planets {
		x1 := float64(p1["x"].(int))
		y1 := float64(p1["y"].(int))

		for j, p2 := range planets {
			if i >= j {
				continue // Avoid duplicate and self-connections
			}

			x2 := float64(p2["x"].(int))
			y2 := float64(p2["y"].(int))

			distance := math.Sqrt((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1))

			if distance <= maxDistance {
				lanes = append(lanes, map[string]interface{}{
					"from":     p1["id"],
					"to":       p2["id"],
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