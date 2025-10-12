package planet

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&BarrenGenerator{})
}

type BarrenGenerator struct{}

func (g *BarrenGenerator) GetWeight() float64 {
	return 10.0 // common while testing
}

func (g *BarrenGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypePlanet
}

func (g *BarrenGenerator) GetSubType() string {
	return "Barren"
}

func (g *BarrenGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*1000 + rand.Intn(1000)

	// Generate name
	names := []string{"Void", "Null", "Empty", "Nil", "NaN", "None"}
	name := fmt.Sprintf("%s %d", names[rand.Intn(len(names))], rand.Intn(100)+1)

	// Generate grey color for barren planet
	planetColor := color.RGBA{
		R: uint8(100 + rand.Intn(50)),
		G: uint8(100 + rand.Intn(50)),
		B: uint8(100 + rand.Intn(50)),
		A: 255,
	}

	// Create the planet
	planet := entities.NewPlanet(
		id,
		name,
		"Barren",
		params.OrbitDistance,
		params.OrbitAngle,
		planetColor,
	)

	// Set desert-specific properties
	planet.Size = 5 + rand.Intn(2)           // 5-6 pixels (medium size)
	planet.Temperature = 30 - rand.Intn(100) // probably cold

	// Atmosphere options for desert worlds
	atmospheres := []string{"Thin", "Thin", "Toxic"} // Higher chance of thin atmosphere
	planet.Atmosphere = atmospheres[rand.Intn(len(atmospheres))]

	// Generate resource entities for barren worlds
	resourceCount := 1 + rand.Intn(2) // 1-2 resource deposits
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

	planet.Habitability = 0

	return planet
}
