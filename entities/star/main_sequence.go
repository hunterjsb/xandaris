package star

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&MainSequenceGenerator{})
}

type MainSequenceGenerator struct{}

func (g *MainSequenceGenerator) GetWeight() float64 {
	return 70.0 // Main sequence stars are very common (like our Sun)
}

func (g *MainSequenceGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeStar
}

func (g *MainSequenceGenerator) GetSubType() string {
	return "Main Sequence"
}

func (g *MainSequenceGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID (stars use system ID directly)
	id := params.SystemID

	// Generate name
	names := []string{"Sol", "Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Eta", "Theta"}
	name := fmt.Sprintf("%s-%d", names[rand.Intn(len(names))], params.SystemID)

	// Main sequence star color (yellow/white tones like our Sun)
	starColor := color.RGBA{
		R: uint8(255),
		G: uint8(245 + rand.Intn(10)),
		B: uint8(200 + rand.Intn(55)),
		A: 255,
	}

	// Create the star
	star := entities.NewStar(
		id,
		name,
		"Main Sequence",
		starColor,
	)

	// Set main sequence-specific properties
	star.Temperature = 5200 + rand.Intn(1200)   // 5200-6400K (G to F class range)
	star.Mass = 0.8 + rand.Float64()*0.6        // 0.8-1.4 solar masses
	star.Radius = 20 + rand.Intn(10)            // 20-29 pixels (base size, will be scaled)
	star.Luminosity = 0.5 + rand.Float64()*2.0  // 0.5-2.5 solar luminosity
	star.Age = 1.0 + rand.Float64()*8.0         // 1-9 billion years
	star.Metallicity = 0.5 + rand.Float64()*1.0 // 0.5-1.5 (Sun = 1.0)

	// 5% chance of having flares
	star.Flares = rand.Float32() < 0.05

	// Very rare chance (1%) of being a binary system
	star.IsBinary = rand.Float32() < 0.01

	// Adjust color based on temperature
	if star.Temperature > 6000 {
		// Hotter = more white/blue
		star.Color.B = uint8(220 + rand.Intn(35))
	} else if star.Temperature < 5500 {
		// Cooler = more orange/red
		star.Color.R = uint8(255)
		star.Color.G = uint8(200 + rand.Intn(40))
		star.Color.B = uint8(150 + rand.Intn(50))
	}

	return star
}
