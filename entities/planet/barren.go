package planet

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/entities/building"
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

	planet.Atmosphere = randomAtmosphereForType(planet.PlanetType)

	// Generate resource entities for barren worlds
	generatePlanetResources(planet, params, 1, 2) // 1-2 resource deposits

	planet.Habitability = 0
	building.EnsurePlanetHasBase(planet, params)

	return planet
}
