package resource

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&WaterGenerator{})
}

type WaterGenerator struct{}

func (g *WaterGenerator) GetWeight() float64 {
	return 18.0 // Water is very common and essential
}

func (g *WaterGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeResource
}

func (g *WaterGenerator) GetSubType() string {
	return "Water"
}

func (g *WaterGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*100000 + rand.Intn(10000)

	// Generate name
	name := fmt.Sprintf("Water Deposit %d", rand.Intn(100)+1)

	// Water color (blue/cyan)
	resourceColor := color.RGBA{
		R: uint8(60 + rand.Intn(100)),
		G: uint8(140 + rand.Intn(100)),
		B: uint8(220 + rand.Intn(35)),
		A: 255,
	}

	// Create the resource
	resource := entities.NewResource(
		id,
		name,
		"Water",
		params.OrbitDistance,
		params.OrbitAngle,
		resourceColor,
	)

	// Set water-specific properties
	resource.Abundance = 30 + rand.Intn(60)             // 30-90% abundance
	resource.ExtractionRate = 0.7 + rand.Float64()*0.25 // 0.7-0.95 (easy to extract)
	resource.Value = 80 + rand.Intn(70)                 // 80-150 credits/unit (valuable for life support)
	resource.Rarity = "Common"
	resource.Size = 5 + rand.Intn(4)      // 5-8 pixels
	resource.Quality = 60 + rand.Intn(35) // 60-95% quality

	return resource
}
