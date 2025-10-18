package planet

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/entities/building"
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

	planet.Atmosphere = randomAtmosphereForType(planet.PlanetType)

	// Civilian population starts at zero; any presence represents later colonisation
	planet.Population = 0

	// Generate resource entities for ice worlds
	generatePlanetResources(planet, params, 2, 3) // 2-4 resource deposits

	// Low to moderate habitability
	planet.Habitability = calculateHabitability(planet.Temperature, planet.Atmosphere, "Ice")

	building.EnsurePlanetHasBase(planet, params)

	// 15% chance of rings
	planet.HasRings = rand.Float32() < 0.15

	return planet
}
