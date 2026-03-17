package building

import (
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&ResearchLabGenerator{})
}

type ResearchLabGenerator struct{}

func (g *ResearchLabGenerator) GetWeight() float64                    { return 0.0 }
func (g *ResearchLabGenerator) GetEntityType() entities.EntityType    { return entities.EntityTypeBuilding }
func (g *ResearchLabGenerator) GetSubType() string                    { return entities.BuildingResearchLab }

func (g *ResearchLabGenerator) Generate(params entities.GenerationParams) entities.Entity {
	id := params.SystemID*100000 + rand.Intn(10000)
	lab := entities.NewBuilding(id, "Research Lab", entities.BuildingResearchLab, params.OrbitDistance, params.OrbitAngle,
		color.RGBA{160, 255, 180, 255})
	lab.AttachmentType = "Planet"
	lab.BuildCost = 2500
	lab.UpkeepCost = 15
	lab.Level = 1
	lab.MaxLevel = 5
	lab.IsOperational = true
	lab.Size = 8
	lab.Description = "Generates Electronics passively through research (1/interval, no inputs)"
	lab.ProductionBonus = 1.0
	lab.SetWorkersRequired(200)
	return lab
}
