package building

import (
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&FactoryGenerator{})
}

type FactoryGenerator struct{}

func (g *FactoryGenerator) GetWeight() float64 {
	return 0.0 // Factories are only built by players
}

func (g *FactoryGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeBuilding
}

func (g *FactoryGenerator) GetSubType() string {
	return "Factory"
}

func (g *FactoryGenerator) Generate(params entities.GenerationParams) entities.Entity {
	id := params.SystemID*100000 + rand.Intn(10000)

	factoryColor := color.RGBA{
		R: 180,
		G: 130,
		B: 255,
		A: 255,
	}

	factory := entities.NewBuilding(
		id,
		"Electronics Factory",
		"Factory",
		params.OrbitDistance,
		params.OrbitAngle,
		factoryColor,
	)

	factory.AttachmentType = "Planet"
	factory.BuildCost = 2000
	factory.UpkeepCost = 12
	factory.Level = 1
	factory.MaxLevel = 5
	factory.IsOperational = true
	factory.Size = 8
	factory.Description = "Converts Rare Metals + Iron into Electronics"
	factory.ProductionBonus = 1.0
	factory.SetWorkersRequired(300)

	return factory
}
