package building

import (
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&TradingPostGenerator{})
}

// TradingPostGenerator creates Trading Post buildings
type TradingPostGenerator struct{}

func (g *TradingPostGenerator) GetWeight() float64 {
	return 0.0 // Player-built only
}

func (g *TradingPostGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeBuilding
}

func (g *TradingPostGenerator) GetSubType() string {
	return "Trading Post"
}

func (g *TradingPostGenerator) Generate(params entities.GenerationParams) entities.Entity {
	id := params.SystemID*1000000 + rand.Intn(100000)

	name := "Trading Post"

	// Warm accent color to differentiate trade structures
	buildingColor := color.RGBA{
		R: 210,
		G: 175,
		B: 95,
		A: 255,
	}

	building := entities.NewBuilding(
		id,
		name,
		"Trading Post",
		params.OrbitDistance,
		params.OrbitAngle,
		buildingColor,
	)

	building.AttachmentType = "Planet"
	building.ProductionBonus = 1.0
	building.PopulationCapacity = 0
	building.BuildCost = 1200
	building.UpkeepCost = 10
	building.MaxLevel = 5
	building.Size = 6
	building.Description = "Establishes a commercial foothold and links the planet to interstellar trade routes."
	building.SetWorkersRequired(150)

	return building
}
