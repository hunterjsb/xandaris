package planet

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&LavaGenerator{})
}

type LavaGenerator struct{}

func (g *LavaGenerator) GetWeight() float64 {
	return 5.0 // Lava planets are less common
}

func (g *LavaGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypePlanet
}

func (g *LavaGenerator) GetSubType() string {
	return "Lava"
}

func (g *LavaGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*1000 + rand.Intn(1000)

	// Generate name
	name := fmt.Sprintf("Inferno %d", rand.Intn(100)+1)

	// Lava planet color (red/orange tones)
	planetColor := color.RGBA{
		R: uint8(200 + rand.Intn(55)),
		G: uint8(50 + rand.Intn(100)),
		B: uint8(20 + rand.Intn(50)),
		A: 255,
	}

	// Create the planet
	planet := entities.NewPlanet(
		id,
		name,
		"Lava",
		params.OrbitDistance,
		params.OrbitAngle,
		planetColor,
	)

	// Set lava-specific properties
	planet.Size = 4 + rand.Intn(3)            // 4-6 pixels (smaller than terrestrial)
	planet.Temperature = 800 + rand.Intn(500) // 800 to 1300°C - extremely hot
	planet.Atmosphere = "Corrosive"           // Always corrosive
	planet.Population = 0                     // Uninhabitable

	// Generate resource entities for lava planets
	resourceCount := 2 + rand.Intn(2) // 2-3 resource deposits
	resourceGenerators := entities.GetGeneratorsByType(entities.EntityTypeResource)
	if len(resourceGenerators) > 0 {
		for i := 0; i < resourceCount; i++ {
			gen := entities.SelectRandomGenerator(resourceGenerators)
			resourceParams := entities.GenerationParams{
				SystemID:      params.SystemID,
				OrbitDistance: 10.0 + float64(i)*5.0 + rand.Float64()*5.0,
				OrbitAngle:    rand.Float64() * 6.28,
				SystemSeed:    params.SystemSeed,
			}
			resource := gen.Generate(resourceParams)
			planet.Resources = append(planet.Resources, resource)
		}
	}

	// Very low habitability
	planet.Habitability = 0 // Completely uninhabitable

	// 5% chance of rings (rare)
	planet.HasRings = rand.Float32() < 0.05

	return planet
}
