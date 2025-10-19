package building

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&MineGenerator{})
}

type MineGenerator struct{}

func (g *MineGenerator) GetWeight() float64 {
	return 0.0 // Buildings are not auto-generated, they're built by players
}

func (g *MineGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeBuilding
}

func (g *MineGenerator) GetSubType() string {
	return "Mine"
}

func (g *MineGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*1000000 + rand.Intn(100000)

	// Generate name
	name := fmt.Sprintf("Mine %d", rand.Intn(100)+1)

	// Mine color (industrial gray/brown)
	buildingColor := color.RGBA{
		R: uint8(120 + rand.Intn(40)),
		G: uint8(100 + rand.Intn(40)),
		B: uint8(80 + rand.Intn(40)),
		A: 255,
	}

	// Create the building
	building := entities.NewBuilding(
		id,
		name,
		"Mine",
		params.OrbitDistance,
		params.OrbitAngle,
		buildingColor,
	)

	// Set mine-specific properties
	building.AttachmentType = "Resource" // Mines are built on resources
	building.ProductionBonus = 1.5       // +50% resource extraction
	building.BuildCost = 500             // 500 credits to build
	building.UpkeepCost = 5              // 5 credits/sec upkeep
	building.MaxLevel = 5                // Can upgrade to level 5
	building.Size = 4                    // 4 pixels
	building.Description = "Extracts resources from deposits. Increases extraction rate."
	building.SetWorkersRequired(80)

	return building
}

// CreateMine is a helper function to create a mine on a specific resource
func CreateMine(resourceID string, owner string, params entities.GenerationParams) *entities.Building {
	gen := &MineGenerator{}
	building := gen.Generate(params).(*entities.Building)
	building.AttachedTo = resourceID
	building.Owner = owner
	return building
}
