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

	// Population based on atmosphere (desert worlds have smaller populations)
	if planet.Atmosphere == "Thin" {
		planet.Population = int64(rand.Intn(500000000)) // Up to 500 million
	} else {
		planet.Population = int64(rand.Intn(100000000)) // Up to 100 million if toxic
	}

	// Resources typical for desert worlds
	resourcePool := []string{"Silicon", "Rare Minerals", "Solar Energy", "Sand", "Precious Stones", "Metal Ores", "Geothermal Vents"}
	numResources := 2 + rand.Intn(3) // 2-4 resources
	planet.Resources = selectRandomResources(resourcePool, numResources)

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
