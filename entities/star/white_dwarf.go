package star

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&WhiteDwarfGenerator{})
}

type WhiteDwarfGenerator struct{}

func (g *WhiteDwarfGenerator) GetWeight() float64 {
	return 8.0 // White dwarfs are moderately common (stellar remnants)
}

func (g *WhiteDwarfGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeStar
}

func (g *WhiteDwarfGenerator) GetSubType() string {
	return "White Dwarf"
}

func (g *WhiteDwarfGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID (stars use system ID directly)
	id := params.SystemID

	// Generate name
	names := []string{"Sirius B", "Procyon B", "Van Maanen", "Wolf", "GD", "WD", "PG", "HZ"}
	name := fmt.Sprintf("%s-%d", names[rand.Intn(len(names))], params.SystemID)

	// White dwarf color (brilliant white/blue-white, very hot surface)
	starColor := color.RGBA{
		R: uint8(240 + rand.Intn(15)),
		G: uint8(245 + rand.Intn(10)),
		B: uint8(255),
		A: 255,
	}

	// Create the star
	star := entities.NewStar(
		id,
		name,
		"White Dwarf",
		starColor,
	)

	// Set white dwarf-specific properties
	star.Temperature = 5000 + rand.Intn(45000)    // 5,000-50,000K (very hot surface)
	star.Mass = 0.5 + rand.Float64()*0.9          // 0.5-1.4 solar masses (Chandrasekhar limit)
	star.Radius = 8 + rand.Intn(5)                // 8-12 pixels (small but still larger than planets, will be scaled)
	star.Luminosity = 0.001 + rand.Float64()*0.01 // 0.001-0.01 solar luminosity (dim despite heat)
	star.Age = 1.0 + rand.Float64()*12.0          // 1-13 billion years (cooling remnants)
	star.Metallicity = 0.8 + rand.Float64()*0.4   // 0.8-1.2 (from original main sequence star)

	// 5% chance of having flares (generally stable)
	star.Flares = rand.Float32() < 0.05

	// 15% chance of being a binary system (many white dwarfs are in binaries)
	star.IsBinary = rand.Float32() < 0.15

	// Adjust color based on temperature
	if star.Temperature > 25000 {
		// Very hot white dwarfs are blue-white
		star.Color.R = uint8(220 + rand.Intn(35))
		star.Color.G = uint8(235 + rand.Intn(20))
	} else if star.Temperature < 10000 {
		// Cooler white dwarfs are more yellow-white
		star.Color.R = uint8(255)
		star.Color.G = uint8(250 + rand.Intn(5))
		star.Color.B = uint8(230 + rand.Intn(25))
	}

	return star
}
