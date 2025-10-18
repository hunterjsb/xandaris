package planet

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&OceanGenerator{})
}

type OceanGenerator struct{}

func (g *OceanGenerator) GetWeight() float64 {
	return 12.0 // Ocean worlds are fairly common and valuable
}

func (g *OceanGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypePlanet
}

func (g *OceanGenerator) GetSubType() string {
	return "Ocean"
}

func (g *OceanGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*1000 + rand.Intn(1000)

	// Generate name
	names := []string{"Aqua", "Marina", "Oceanus", "Poseidon", "Nautilus", "Coral", "Atlantis"}
	name := fmt.Sprintf("%s %d", names[rand.Intn(len(names))], rand.Intn(100)+1)

	// Ocean world color (blue/teal tones)
	planetColor := color.RGBA{
		R: uint8(20 + rand.Intn(60)),
		G: uint8(100 + rand.Intn(100)),
		B: uint8(180 + rand.Intn(75)),
		A: 255,
	}

	// Create the planet
	planet := entities.NewPlanet(
		id,
		name,
		"Ocean",
		params.OrbitDistance,
		params.OrbitAngle,
		planetColor,
	)

	// Set ocean world-specific properties
	planet.Size = 5 + rand.Intn(3)         // 5-7 pixels (similar to terrestrial)
	planet.Temperature = 0 + rand.Intn(40) // 0 to 40°C - temperate water worlds

	// Atmosphere options for ocean worlds
	atmospheres := []string{"Breathable", "Breathable", "Toxic"} // Higher chance of breathable
	planet.Atmosphere = atmospheres[rand.Intn(len(atmospheres))]

	// Civilian population now starts at zero; growth systems will populate habitable worlds later
	planet.Population = 0

	// Generate resource entities for ocean worlds
	generatePlanetResources(planet, params, 3, 3) // 3-5 resource deposits (ocean worlds are resource-rich)

	// High habitability (water is life)
	planet.Habitability = calculateHabitability(planet.Temperature, planet.Atmosphere, "Ocean")
	// Bonus for ocean worlds
	if planet.Atmosphere == "Breathable" {
		planet.Habitability += 15
		if planet.Habitability > 100 {
			planet.Habitability = 100
		}
	}

	// 20% chance of rings
	planet.HasRings = rand.Float32() < 0.20

	return planet
}
