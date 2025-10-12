package resource

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&OilGenerator{})
	entities.RegisterGenerator(&FuelGenerator{}) // Fuel doesn't spawn naturally (weight 0.0) but needs to be registered
}

type OilGenerator struct{}

func (g *OilGenerator) GetWeight() float64 {
	return 12.0 // Oil is fairly common
}

func (g *OilGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeResource
}

func (g *OilGenerator) GetSubType() string {
	return "Oil"
}

func (g *OilGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*100000 + rand.Intn(10000)

	// Generate name
	name := fmt.Sprintf("Oil Deposit %d", rand.Intn(100)+1)

	// Oil color (dark brown/black)
	resourceColor := color.RGBA{
		R: uint8(40 + rand.Intn(60)),
		G: uint8(30 + rand.Intn(40)),
		B: uint8(20 + rand.Intn(30)),
		A: 255,
	}

	// Create the resource
	resource := entities.NewResource(
		id,
		name,
		"Oil",
		params.OrbitDistance,
		params.OrbitAngle,
		resourceColor,
	)

	// Set oil-specific properties
	resource.Abundance = 20 + rand.Intn(50)            // 20-70% abundance
	resource.ExtractionRate = 0.5 + rand.Float64()*0.3 // 0.5-0.8 (moderate extraction)
	resource.Value = 100 + rand.Intn(100)              // 100-200 credits/unit
	resource.Rarity = "Uncommon"
	resource.Size = 6 + rand.Intn(5)      // 6-10 pixels
	resource.Quality = 50 + rand.Intn(45) // 50-95% quality

	return resource
}

// FuelGenerator defines Fuel as a resource type but it doesn't spawn naturally
// Fuel is produced by refineries that convert Oil
type FuelGenerator struct{}

func (g *FuelGenerator) GetWeight() float64 {
	return 0.0 // Fuel does not spawn naturally - only produced by refineries
}

func (g *FuelGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeResource
}

func (g *FuelGenerator) GetSubType() string {
	return "Fuel"
}

func (g *FuelGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// This should never be called since weight is 0.0
	// But we need it defined so "Fuel" is a valid resource type
	id := params.SystemID*100000 + rand.Intn(10000)
	name := "Fuel"

	// Fuel color (orange/yellow)
	resourceColor := color.RGBA{
		R: 255,
		G: 165,
		B: 0,
		A: 255,
	}

	resource := entities.NewResource(
		id,
		name,
		"Fuel",
		params.OrbitDistance,
		params.OrbitAngle,
		resourceColor,
	)

	resource.Abundance = 100
	resource.ExtractionRate = 1.0
	resource.Value = 200 // High value - refined product
	resource.Rarity = "Refined"
	resource.Size = 5
	resource.Quality = 100

	return resource
}
