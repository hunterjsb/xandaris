package planet

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&DesertGenerator{})
}

type DesertGenerator struct{}

func (g *DesertGenerator) GetWeight() float64 {
	return 9.0 // Desert planets are moderately common
}

func (g *DesertGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypePlanet
}

func (g *DesertGenerator) GetSubType() string {
	return "Desert"
}

func (g *DesertGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*1000 + rand.Intn(1000)

	// Generate name
	names := []string{"Dune", "Arid", "Sahara", "Gobi", "Mojave", "Atacama", "Kalahari"}
	name := fmt.Sprintf("%s %d", names[rand.Intn(len(names))], rand.Intn(100)+1)

	// Desert planet color (sandy/tan tones)
	planetColor := color.RGBA{
		R: uint8(220 + rand.Intn(35)),
		G: uint8(180 + rand.Intn(55)),
		B: uint8(100 + rand.Intn(50)),
		A: 255,
	}

	// Create the planet
	planet := entities.NewPlanet(
		id,
		name,
		"Desert",
		params.OrbitDistance,
		params.OrbitAngle,
		planetColor,
	)

	// Set desert-specific properties
	planet.Size = 5 + rand.Intn(2)          // 5-6 pixels (medium size)
	planet.Temperature = 30 + rand.Intn(70) // 30 to 100°C - hot and dry

	// Atmosphere options for desert worlds
	atmospheres := []string{"Thin", "Thin", "Toxic"} // Higher chance of thin atmosphere
	planet.Atmosphere = atmospheres[rand.Intn(len(atmospheres))]

	// Civilian population starts at zero; habitation will grow once colonised
	planet.Population = 0

	// Generate resource entities for desert worlds
	generatePlanetResources(planet, params, 2, 3) // 2-4 resource deposits

	// Low to moderate habitability
	planet.Habitability = calculateHabitability(planet.Temperature, planet.Atmosphere, "Desert")
	// Desert penalty
	planet.Habitability -= 15
	if planet.Habitability < 0 {
		planet.Habitability = 0
	}

	// 5% chance of rings (rare for desert worlds)
	planet.HasRings = rand.Float32() < 0.05

	return planet
}
