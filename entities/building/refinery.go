package building

import (
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&RefineryGenerator{})
}

type RefineryGenerator struct{}

func (g *RefineryGenerator) GetWeight() float64 {
	return 0.0 // Refineries are only built by players
}

func (g *RefineryGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeBuilding
}

func (g *RefineryGenerator) GetSubType() string {
	return "Refinery"
}

func (g *RefineryGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*100000 + rand.Intn(10000)

	// Generate name
	name := "Oil Refinery"

	// Refinery color (orange - for petroleum products)
	refineryColor := color.RGBA{
		R: 255,
		G: 150,
		B: 100,
		A: 255,
	}

	// Create the refinery building
	refinery := entities.NewBuilding(
		id,
		name,
		"Refinery",
		params.OrbitDistance,
		params.OrbitAngle,
		refineryColor,
	)

	// Configure refinery properties
	refinery.AttachmentType = "Planet"
	refinery.BuildCost = 1500
	refinery.UpkeepCost = 15
	refinery.Level = 1
	refinery.MaxLevel = 5
	refinery.IsOperational = true
	refinery.Size = 8
	refinery.Description = "Converts Oil into Fuel for spacecraft propulsion"
	refinery.ProductionBonus = 1.0 // Base efficiency

	return refinery
}
