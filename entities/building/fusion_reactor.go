package building

import (
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&FusionReactorGenerator{})
}

type FusionReactorGenerator struct{}

func (g *FusionReactorGenerator) GetWeight() float64        { return 0.0 }
func (g *FusionReactorGenerator) GetEntityType() entities.EntityType { return entities.EntityTypeBuilding }
func (g *FusionReactorGenerator) GetSubType() string         { return entities.BuildingFusionReactor }

func (g *FusionReactorGenerator) Generate(params entities.GenerationParams) entities.Entity {
	id := params.SystemID*100000 + rand.Intn(10000)
	reactor := entities.NewBuilding(id, "Fusion Reactor", entities.BuildingFusionReactor, params.OrbitDistance, params.OrbitAngle,
		color.RGBA{100, 220, 255, 255})
	reactor.AttachmentType = "Planet"
	reactor.BuildCost = 3000
	reactor.UpkeepCost = 15
	reactor.Level = 1
	reactor.MaxLevel = 5
	reactor.IsOperational = true
	reactor.Size = 10
	reactor.Description = "Helium-3 fusion produces 200 MW of clean power"
	reactor.ProductionBonus = 1.0
	reactor.SetWorkersRequired(200)
	return reactor
}
