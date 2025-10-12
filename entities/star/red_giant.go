package star

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&RedGiantGenerator{})
}

type RedGiantGenerator struct{}

func (g *RedGiantGenerator) GetWeight() float64 {
	return 15.0 // Red giants are less common (evolved stars)
}

func (g *RedGiantGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeStar
}

func (g *RedGiantGenerator) GetSubType() string {
	return "Red Giant"
}

func (g *RedGiantGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID (stars use system ID directly)
	id := params.SystemID

	// Generate name
	names := []string{"Arcturus", "Aldebaran", "Betelgeuse", "Antares", "Mira", "Pollux", "Capella", "Rigel"}
	name := fmt.Sprintf("%s-%d", names[rand.Intn(len(names))], params.SystemID)

	// Red giant color (bright red/orange tones, more saturated than red dwarfs)
	starColor := color.RGBA{
		R: uint8(255),
		G: uint8(120 + rand.Intn(100)),
		B: uint8(40 + rand.Intn(80)),
		A: 255,
	}

	// Create the star
	star := entities.NewStar(
		id,
		name,
		"Red Giant",
		starColor,
	)

	// Set red giant-specific properties
	star.Temperature = 3000 + rand.Intn(2000)      // 3000-5000K (cooler surface due to expansion)
	star.Mass = 0.5 + rand.Float64()*2.0           // 0.5-2.5 solar masses (lost mass during expansion)
	star.Radius = 40 + rand.Intn(20)               // 40-59 pixels (larger than main sequence, will be scaled)
	star.Luminosity = 100.0 + rand.Float64()*400.0 // 100-500x solar luminosity (very bright)
	star.Age = 8.0 + rand.Float64()*4.0            // 8-12 billion years (evolved, older stars)
	star.Metallicity = 0.3 + rand.Float64()*1.2    // 0.3-1.5 (older generation stars)

	// 15% chance of having flares (less stable than main sequence)
	star.Flares = rand.Float32() < 0.15

	// 3% chance of being a binary system (many companions would have been consumed)
	star.IsBinary = rand.Float32() < 0.03

	// Adjust color based on temperature and size
	if star.Temperature > 4000 {
		// Warmer red giants are more orange
		star.Color.G = uint8(180 + rand.Intn(60))
		star.Color.B = uint8(100 + rand.Intn(80))
	} else {
		// Cooler red giants are deeper red
		star.Color.G = uint8(100 + rand.Intn(80))
		star.Color.B = uint8(50 + rand.Intn(70))
	}

	return star
}
