package resource

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&Helium3Generator{})
}

type Helium3Generator struct{}

func (g *Helium3Generator) GetWeight() float64 {
	return 6.0 // Helium-3 is rare and valuable
}

func (g *Helium3Generator) GetEntityType() entities.EntityType {
	return entities.EntityTypeResource
}

func (g *Helium3Generator) GetSubType() string {
	return "Helium-3"
}

func (g *Helium3Generator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*100000 + rand.Intn(10000)

	// Generate name
	name := fmt.Sprintf("Helium-3 Deposit %d", rand.Intn(100)+1)

	// Helium-3 color (light blue/cyan for gas)
	resourceColor := color.RGBA{
		R: uint8(150 + rand.Intn(80)),
		G: uint8(200 + rand.Intn(55)),
		B: uint8(240 + rand.Intn(15)),
		A: 255,
	}

	// Create the resource
	resource := entities.NewResource(
		id,
		name,
		"Helium-3",
		params.OrbitDistance,
		params.OrbitAngle,
		resourceColor,
	)

	// Set Helium-3-specific properties
	resource.Abundance = 15 + rand.Intn(35)            // 15-50% abundance (rare)
	resource.ExtractionRate = 0.3 + rand.Float64()*0.3 // 0.3-0.6 (moderate difficulty, gas extraction)
	resource.Value = 400 + rand.Intn(600)              // 400-1000 credits/unit (fusion fuel is very valuable)
	resource.Rarity = []string{"Rare", "Very Rare"}[rand.Intn(2)]
	resource.Size = 4 + rand.Intn(4)      // 4-7 pixels
	resource.Quality = 50 + rand.Intn(45) // 50-95% quality

	return resource
}
