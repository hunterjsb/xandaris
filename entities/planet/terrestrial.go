package planet

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&TerrestrialGenerator{})
}

type TerrestrialGenerator struct{}

func (g *TerrestrialGenerator) GetWeight() float64 {
	return 15.0 // Terrestrial planets are common
}

func (g *TerrestrialGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypePlanet
}

func (g *TerrestrialGenerator) GetSubType() string {
	return "Terrestrial"
}

func (g *TerrestrialGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*1000 + rand.Intn(1000)

	// Generate name
	name := fmt.Sprintf("Planet %d", rand.Intn(100)+1)

	// Terrestrial planet color (earth-like tones)
	planetColor := color.RGBA{
		R: uint8(50 + rand.Intn(100)),
		G: uint8(100 + rand.Intn(100)),
		B: uint8(50 + rand.Intn(80)),
		A: 255,
	}

	// Create the planet
	planet := entities.NewPlanet(
		id,
		name,
		"Terrestrial",
		params.OrbitDistance,
		params.OrbitAngle,
		planetColor,
	)

	// Set terrestrial-specific properties
	planet.Size = 5 + rand.Intn(3)           // 5-7 pixels
	planet.Temperature = -20 + rand.Intn(60) // -20 to 40°C

	// Atmosphere options for terrestrial planets
	atmospheres := []string{"Breathable", "Toxic", "Thin"}
	planet.Atmosphere = atmospheres[rand.Intn(len(atmospheres))]

	// Population based on atmosphere
	if planet.Atmosphere == "Breathable" {
		planet.Population = int64(rand.Intn(2000000000))
	} else if planet.Atmosphere == "Thin" {
		planet.Population = int64(rand.Intn(500000000))
	} else {
		planet.Population = int64(rand.Intn(100000000))
	}

	// Resources typical for terrestrial planets
	resourcePool := []string{"Iron", "Water", "Food", "Rare Metals", "Minerals", "Timber"}
	numResources := 2 + rand.Intn(3) // 2-4 resources
	planet.Resources = selectRandomResources(resourcePool, numResources)

	// Calculate habitability
	planet.Habitability = calculateHabitability(planet.Temperature, planet.Atmosphere, "Terrestrial")

	// 10% chance of rings
	planet.HasRings = rand.Float32() < 0.10

	return planet
}

// selectRandomResources picks random resources from a pool
func selectRandomResources(pool []string, count int) []string {
	if count > len(pool) {
		count = len(pool)
	}

	result := make([]string, 0, count)
	used := make(map[int]bool)

	for len(result) < count {
		idx := rand.Intn(len(pool))
		if !used[idx] {
			result = append(result, pool[idx])
			used[idx] = true
		}
	}

	return result
}

// calculateHabitability calculates a habitability score 0-100
func calculateHabitability(temperature int, atmosphere string, planetType string) int {
	score := 50 // Base score

	// Temperature scoring
	if temperature >= -10 && temperature <= 30 {
		score += 30 // Ideal temperature
	} else if temperature >= -30 && temperature <= 50 {
		score += 15 // Acceptable temperature
	} else {
		score -= 20 // Poor temperature
	}

	// Atmosphere scoring
	switch atmosphere {
	case "Breathable":
		score += 20
	case "Thin":
		score += 5
	case "Dense":
		score -= 5
	case "Toxic":
		score -= 15
	case "Corrosive":
		score -= 30
	case "None":
		score -= 25
	}

	// Planet type bonus
	if planetType == "Terrestrial" {
		score += 10
	}

	// Ensure score is between 0 and 100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}
