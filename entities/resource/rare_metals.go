package resource

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&RareMetalsGenerator{})
}

type RareMetalsGenerator struct{}

func (g *RareMetalsGenerator) GetWeight() float64 {
	return 8.0 // Rare metals are uncommon
}

func (g *RareMetalsGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeResource
}

func (g *RareMetalsGenerator) GetSubType() string {
	return "Rare Metals"
}

func (g *RareMetalsGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*100000 + rand.Intn(10000)

	// Generate name
	names := []string{"Platinum", "Iridium", "Palladium", "Rhodium", "Gold"}
	name := fmt.Sprintf("%s Deposit %d", names[rand.Intn(len(names))], rand.Intn(100)+1)

	// Rare metals color (gold/bronze metallic)
	resourceColor := color.RGBA{
		R: uint8(200 + rand.Intn(55)),
		G: uint8(160 + rand.Intn(60)),
		B: uint8(60 + rand.Intn(80)),
		A: 255,
	}

	// Create the resource
	resource := entities.NewResource(
		id,
		name,
		"Rare Metals",
		params.OrbitDistance,
		params.OrbitAngle,
		resourceColor,
	)

	// Set rare metals-specific properties
	resource.Abundance = 10 + rand.Intn(40)            // 10-50% abundance (scarce)
	resource.ExtractionRate = 0.2 + rand.Float64()*0.4 // 0.2-0.6 (difficult to extract)
	resource.Value = 300 + rand.Intn(500)              // 300-800 credits/unit (very valuable)
	resource.Rarity = []string{"Uncommon", "Rare"}[rand.Intn(2)]
	resource.Size = 3 + rand.Intn(3)      // 3-5 pixels (smaller deposits)
	resource.Quality = 40 + rand.Intn(50) // 40-90% quality

	return resource
}
