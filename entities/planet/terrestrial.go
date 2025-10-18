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

	planet.Atmosphere = randomAtmosphereForType(planet.PlanetType)

	// Civilian population starts at zero; future growth will depend on habitability and housing
	planet.Population = 0

	// Generate resource entities for terrestrial planets
	generatePlanetResources(planet, params, 2, 3) // 2-4 resource deposits

	// Calculate habitability
	planet.Habitability = calculateHabitability(planet.Temperature, planet.Atmosphere, "Terrestrial")

	// 10% chance of rings
	planet.HasRings = rand.Float32() < 0.10

	planet.RecalculateBasePopulationCapacity()

	return planet
}

// generatePlanetResources generates resource nodes for a planet with proper distribution
func generatePlanetResources(planet *entities.Planet, params entities.GenerationParams, minResources, maxResourcesRange int) {
	// Max 6 resource nodes per planet (to ensure space for mines and visual clarity)
	maxResources := 6
	resourceCount := minResources + rand.Intn(maxResourcesRange)
	if resourceCount > maxResources {
		resourceCount = maxResources
	}

	resourceGenerators := entities.GetGeneratorsByType(entities.EntityTypeResource)
	if len(resourceGenerators) > 0 {
		// Distribute resource nodes evenly around the planet
		angleStep := 6.28318 / float64(maxResources) // 2π divided by max nodes
		for i := 0; i < resourceCount; i++ {
			gen := entities.SelectRandomGenerator(resourceGenerators)
			// Assign evenly distributed angles for node positions
			nodeAngle := float64(i)*angleStep + rand.Float64()*0.3 // Small random offset
			resourceParams := entities.GenerationParams{
				SystemID:      params.SystemID,
				OrbitDistance: 10.0 + float64(i)*5.0 + rand.Float64()*5.0,
				OrbitAngle:    nodeAngle, // This will become NodePosition
				SystemSeed:    params.SystemSeed,
			}
			resource := gen.Generate(resourceParams)
			planet.Resources = append(planet.Resources, resource)
		}
	}
}

// calculateHabitability calculates a habitability score (0-100) using shared rules
func calculateHabitability(temperature int, atmosphere string, planetType string) int {
	score := 40 // Base score shared across types

	profile, ok := planetTemperatureProfiles[planetType]
	if !ok {
		profile = planetTemperatureProfiles["default"]
	}

	score += temperatureScore(temperature, profile)

	if modifier, ok := atmosphereHabitabilityModifiers[atmosphere]; ok {
		score += modifier
	} else {
		score -= 20
	}

	if modifier, ok := planetTypeHabitabilityModifiers[planetType]; ok {
		score += modifier
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}
