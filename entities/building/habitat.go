package building

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&HabitatGenerator{})
}

type HabitatGenerator struct{}

func (g *HabitatGenerator) GetWeight() float64 {
	return 0.0 // Buildings are not auto-generated, they're built by players
}

func (g *HabitatGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeBuilding
}

func (g *HabitatGenerator) GetSubType() string {
	return "Habitat"
}

func (g *HabitatGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*1000000 + rand.Intn(100000)

	// Generate name
	name := fmt.Sprintf("Habitat %d", rand.Intn(100)+1)

	// Habitat color (blue/green living space)
	buildingColor := color.RGBA{
		R: uint8(80 + rand.Intn(60)),
		G: uint8(140 + rand.Intn(80)),
		B: uint8(180 + rand.Intn(60)),
		A: 255,
	}

	// Create the building
	building := entities.NewBuilding(
		id,
		name,
		"Habitat",
		params.OrbitDistance,
		params.OrbitAngle,
		buildingColor,
	)

	// Set habitat-specific properties
	building.AttachmentType = "Planet"     // Habitats are built on planets
	building.ProductionBonus = 1.0         // No production bonus (housing focused)
	building.PopulationCapacity = 10000000 // 10 million population capacity
	building.BuildCost = 800               // 800 credits to build
	building.UpkeepCost = 8                // 8 credits/sec upkeep
	building.MaxLevel = 10                 // Can upgrade to level 10 (more housing)
	building.Size = 5                      // 5 pixels (larger than mine)
	building.Description = "Provides housing for population. Increases planet capacity."
	building.SetWorkersRequired(200)

	return building
}

// CreateHabitat is a helper function to create a habitat on a specific planet
func CreateHabitat(planetID string, owner string, params entities.GenerationParams) *entities.Building {
	gen := &HabitatGenerator{}
	building := gen.Generate(params).(*entities.Building)
	building.AttachedTo = planetID
	building.Owner = owner
	return building
}
