package resource

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&OilGenerator{})
	entities.RegisterGenerator(&FuelGenerator{})
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

type FuelGenerator struct{}

func (g *FuelGenerator) GetWeight() float64 {
	return 8.0 // Fuel is less common and valuable
}

func (g *FuelGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeResource
}

func (g *FuelGenerator) GetSubType() string {
	return "Fuel"
}

func (g *FuelGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*100000 + rand.Intn(10000)

	// Generate name
	name := fmt.Sprintf("Fuel Deposit %d", rand.Intn(100)+1)

	// Fuel color (orange/yellow)
	resourceColor := color.RGBA{
		R: uint8(200 + rand.Intn(55)),
		G: uint8(150 + rand.Intn(80)),
		B: uint8(40 + rand.Intn(60)),
		A: 255,
	}

	// Create the resource
	resource := entities.NewResource(
		id,
		name,
		"Fuel",
		params.OrbitDistance,
		params.OrbitAngle,
		resourceColor,
	)

	// Set fuel-specific properties
	resource.Abundance = 15 + rand.Intn(40)             // 15-55% abundance
	resource.ExtractionRate = 0.4 + rand.Float64()*0.35 // 0.4-0.75 (harder to extract)
	resource.Value = 150 + rand.Intn(150)               // 150-300 credits/unit (high value for propulsion)
	resource.Rarity = "Rare"
	resource.Size = 4 + rand.Intn(5)      // 4-8 pixels
	resource.Quality = 55 + rand.Intn(40) // 55-95% quality

	return resource
}
