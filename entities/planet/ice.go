package planet

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&IceGenerator{})
}

type IceGenerator struct{}

func (g *IceGenerator) GetWeight() float64 {
	return 8.0 // Ice worlds are moderately common
}

func (g *IceGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypePlanet
}

func (g *IceGenerator) GetSubType() string {
	return "Ice"
}

func (g *IceGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*1000 + rand.Intn(1000)

	// Generate name
	names := []string{"Frost", "Glacier", "Tundra", "Icarus", "Borea", "Cryo"}
	name := fmt.Sprintf("%s %d", names[rand.Intn(len(names))], rand.Intn(100)+1)

	// Ice world color (blue/white icy tones)
	planetColor := color.RGBA{
		R: uint8(200 + rand.Intn(55)),
		G: uint8(220 + rand.Intn(35)),
		B: uint8(240 + rand.Intn(15)),
		A: 255,
	}

	// Create the planet
	planet := entities.NewPlanet(
		id,
		name,
		"Ice",
		params.OrbitDistance,
		params.OrbitAngle,
		planetColor,
	)

	// Set ice world-specific properties
	planet.Size = 4 + rand.Intn(3)           // 4-6 pixels
	planet.Temperature = -80 + rand.Intn(40) // -80 to -40°C - very cold

	// Atmosphere options for ice worlds
	atmospheres := []string{"Thin", "None"}
	planet.Atmosphere = atmospheres[rand.Intn(len(atmospheres))]

	// Small population (research stations, mining colonies)
	if planet.Atmosphere == "Thin" {
		planet.Population = int64(rand.Intn(50000000))
	} else {
		planet.Population = int64(rand.Intn(10000000))
	}

	// Generate resource entities for ice worlds
	resourceCount := 2 + rand.Intn(3) // 2-4 resource deposits
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

	// Low to moderate habitability
	planet.Habitability = calculateHabitability(planet.Temperature, planet.Atmosphere, "Ice")

	// 15% chance of rings
	planet.HasRings = rand.Float32() < 0.15

	return planet
}
