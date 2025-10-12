package resource

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&IronGenerator{})
}

type IronGenerator struct{}

func (g *IronGenerator) GetWeight() float64 {
	return 20.0 // Iron is very common
}

func (g *IronGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeResource
}

func (g *IronGenerator) GetSubType() string {
	return "Iron"
}

func (g *IronGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*100000 + rand.Intn(10000)

	// Generate name
	name := fmt.Sprintf("Iron Deposit %d", rand.Intn(100)+1)

	// Iron color (gray/metallic)
	resourceColor := color.RGBA{
		R: uint8(140 + rand.Intn(60)),
		G: uint8(140 + rand.Intn(60)),
		B: uint8(140 + rand.Intn(60)),
		A: 255,
	}

	// Create the resource
	resource := entities.NewResource(
		id,
		name,
		"Iron",
		params.OrbitDistance,
		params.OrbitAngle,
		resourceColor,
	)

	// Set iron-specific properties
	resource.Abundance = 40 + rand.Intn(50)            // 40-90% abundance
	resource.ExtractionRate = 0.6 + rand.Float64()*0.3 // 0.6-0.9 (fairly easy to extract)
	resource.Value = 50 + rand.Intn(50)                // 50-100 credits/unit
	resource.Rarity = "Common"
	resource.Size = 4 + rand.Intn(3)      // 4-6 pixels
	resource.Quality = 50 + rand.Intn(40) // 50-90% quality

	return resource
}
