package star

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&RedDwarfGenerator{})
}

type RedDwarfGenerator struct{}

func (g *RedDwarfGenerator) GetWeight() float64 {
	return 75.0 // Red dwarfs are the most common stars in the universe
}

func (g *RedDwarfGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeStar
}

func (g *RedDwarfGenerator) GetSubType() string {
	return "Red Dwarf"
}

func (g *RedDwarfGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID (stars use system ID directly)
	id := params.SystemID

	// Generate name
	names := []string{"Proxima", "Wolf", "Ross", "Gliese", "Lacaille", "Luyten", "Kapteyn", "Barnard"}
	name := fmt.Sprintf("%s-%d", names[rand.Intn(len(names))], params.SystemID)

	// Red dwarf color (deep red/orange tones)
	starColor := color.RGBA{
		R: uint8(255),
		G: uint8(100 + rand.Intn(100)),
		B: uint8(50 + rand.Intn(80)),
		A: 255,
	}

	// Create the star
	star := entities.NewStar(
		id,
		name,
		"Red Dwarf",
		starColor,
	)

	// Set red dwarf-specific properties
	star.Temperature = 2500 + rand.Intn(1300)      // 2500-3800K (M-class range)
	star.Mass = 0.08 + rand.Float64()*0.37         // 0.08-0.45 solar masses (minimum for fusion)
	star.Radius = 10 + rand.Intn(8)                // 10-17 pixels (smaller than main sequence, will be scaled)
	star.Luminosity = 0.0001 + rand.Float64()*0.08 // Very dim: 0.0001-0.08 solar luminosity
	star.Age = 1.0 + rand.Float64()*12.0           // 1-13 billion years (can be very old)
	star.Metallicity = 0.1 + rand.Float64()*0.8    // 0.1-0.9 (generally metal-poor)

	// 25% chance of having flares (red dwarfs are known for flares)
	star.Flares = rand.Float32() < 0.25

	// 8% chance of being a binary system
	star.IsBinary = rand.Float32() < 0.08

	// Adjust color based on temperature
	if star.Temperature > 3200 {
		// Warmer red dwarfs are more orange
		star.Color.G = uint8(150 + rand.Intn(80))
		star.Color.B = uint8(80 + rand.Intn(70))
	} else {
		// Cooler red dwarfs are deeper red
		star.Color.G = uint8(80 + rand.Intn(60))
		star.Color.B = uint8(40 + rand.Intn(60))
	}

	return star
}
