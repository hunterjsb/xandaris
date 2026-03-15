package building

import (
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&GeneratorGenerator{})
}

type GeneratorGenerator struct{}

func (g *GeneratorGenerator) GetWeight() float64        { return 0.0 }
func (g *GeneratorGenerator) GetEntityType() entities.EntityType { return entities.EntityTypeBuilding }
func (g *GeneratorGenerator) GetSubType() string         { return "Generator" }

func (g *GeneratorGenerator) Generate(params entities.GenerationParams) entities.Entity {
	id := params.SystemID*100000 + rand.Intn(10000)
	gen := entities.NewBuilding(id, "Fuel Generator", "Generator", params.OrbitDistance, params.OrbitAngle,
		color.RGBA{255, 180, 50, 255})
	gen.AttachmentType = "Planet"
	gen.BuildCost = 1000
	gen.UpkeepCost = 8
	gen.Level = 1
	gen.MaxLevel = 5
	gen.IsOperational = true
	gen.Size = 7
	gen.Description = "Burns Fuel to generate 50 MW of power for the colony"
	gen.ProductionBonus = 1.0
	gen.SetWorkersRequired(100)
	return gen
}
