package star

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&BlueGiantGenerator{})
}

type BlueGiantGenerator struct{}

func (g *BlueGiantGenerator) GetWeight() float64 {
	return 5.0 // Blue giants are rare, massive, short-lived stars
}

func (g *BlueGiantGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeStar
}

func (g *BlueGiantGenerator) GetSubType() string {
	return "Blue Giant"
}

func (g *BlueGiantGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID (stars use system ID directly)
	id := params.SystemID

	// Generate name
	names := []string{"Rigel", "Spica", "Regulus", "Bellatrix", "Mintaka", "Alnilam", "Alnitak", "Shaula"}
	name := fmt.Sprintf("%s-%d", names[rand.Intn(len(names))], params.SystemID)

	// Blue giant color (brilliant blue/white tones)
	starColor := color.RGBA{
		R: uint8(180 + rand.Intn(75)),
		G: uint8(200 + rand.Intn(55)),
		B: uint8(255),
		A: 255,
	}

	// Create the star
	star := entities.NewStar(
		id,
		name,
		"Blue Giant",
		starColor,
	)

	// Set blue giant-specific properties
	star.Temperature = 20000 + rand.Intn(30000)        // 20,000-50,000K (extremely hot)
	star.Mass = 10.0 + rand.Float64()*40.0             // 10-50 solar masses (very massive)
	star.Radius = 30 + rand.Intn(15)                   // 30-44 pixels (large but not as big as red giants, will be scaled)
	star.Luminosity = 10000.0 + rand.Float64()*90000.0 // 10,000-100,000x solar luminosity (incredibly bright)
	star.Age = 0.001 + rand.Float64()*0.099            // 1-100 million years (very young, short-lived)
	star.Metallicity = 1.0 + rand.Float64()*1.0        // 1.0-2.0 (young, metal-rich stars)

	// 40% chance of having flares (very energetic and unstable)
	star.Flares = rand.Float32() < 0.40

	// 12% chance of being a binary system (massive stars often form in pairs)
	star.IsBinary = rand.Float32() < 0.12

	// Adjust color based on temperature
	if star.Temperature > 40000 {
		// Extremely hot blue giants are almost pure blue-white
		star.Color.R = uint8(200 + rand.Intn(55))
		star.Color.G = uint8(220 + rand.Intn(35))
	} else if star.Temperature < 25000 {
		// Slightly cooler blue giants have more white
		star.Color.R = uint8(220 + rand.Intn(35))
		star.Color.G = uint8(230 + rand.Intn(25))
	}

	return star
}
