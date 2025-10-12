package planet

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&GasGiantGenerator{})
}

type GasGiantGenerator struct{}

func (g *GasGiantGenerator) GetWeight() float64 {
	return 10.0 // Gas giants are fairly common
}

func (g *GasGiantGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypePlanet
}

func (g *GasGiantGenerator) GetSubType() string {
	return "Gas Giant"
}

func (g *GasGiantGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*1000 + rand.Intn(1000)

	// Generate name
	names := []string{"Goliath", "Titan", "Colossus", "Behemoth", "Giant", "Leviathan"}
	name := fmt.Sprintf("%s %d", names[rand.Intn(len(names))], rand.Intn(100)+1)

	// Gas giant colors (varied: Jupiter-like browns, Saturn-like yellows, Neptune-like blues)
	colorTypes := []color.RGBA{
		{R: 180, G: 140, B: 100, A: 255}, // Jupiter-like brown/orange
		{R: 220, G: 200, B: 140, A: 255}, // Saturn-like yellow
		{R: 80, G: 120, B: 200, A: 255},  // Neptune-like blue
		{R: 160, G: 180, B: 200, A: 255}, // Uranus-like cyan
	}
	planetColor := colorTypes[rand.Intn(len(colorTypes))]

	// Create the planet
	planet := entities.NewPlanet(
		id,
		name,
		"Gas Giant",
		params.OrbitDistance,
		params.OrbitAngle,
		planetColor,
	)

	// Set gas giant-specific properties
	planet.Size = 8 + rand.Intn(4)            // 8-11 pixels (much larger than terrestrial)
	planet.Temperature = -150 + rand.Intn(50) // -150 to -100°C - very cold
	planet.Atmosphere = "Dense"               // Always dense atmosphere
	planet.Population = 0                     // No surface, but could have floating cities in future

	// Generate resource entities for gas giants
	generatePlanetResources(planet, params, 2, 3) // 2-4 resource deposits

	// Very low habitability (no solid surface)
	planet.Habitability = 5 + rand.Intn(10) // 5-15% (potential for floating stations)

	// 40% chance of rings (gas giants often have rings)
	planet.HasRings = rand.Float32() < 0.40

	return planet
}
