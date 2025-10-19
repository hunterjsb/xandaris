package building

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&ShipyardGenerator{})
}

type ShipyardGenerator struct{}

func (g *ShipyardGenerator) GetWeight() float64 {
	return 0.0 // Buildings are not auto-generated, they're built by players
}

func (g *ShipyardGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeBuilding
}

func (g *ShipyardGenerator) GetSubType() string {
	return "Shipyard"
}

func (g *ShipyardGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*1000000 + rand.Intn(100000)

	// Generate name
	names := []string{"Orbital Shipyard", "Construction Dock", "Assembly Station", "Forge"}
	name := fmt.Sprintf("%s %d", names[rand.Intn(len(names))], rand.Intn(100)+1)

	// Shipyard color (metallic silver/blue industrial)
	buildingColor := color.RGBA{
		R: uint8(140 + rand.Intn(60)),
		G: uint8(150 + rand.Intn(60)),
		B: uint8(180 + rand.Intn(60)),
		A: 255,
	}

	// Create the building
	building := entities.NewBuilding(
		id,
		name,
		"Shipyard",
		params.OrbitDistance,
		params.OrbitAngle,
		buildingColor,
	)

	// Set shipyard-specific properties
	building.AttachmentType = "Planet" // Shipyards are built on planets
	building.ProductionBonus = 2.0     // +100% ship construction speed
	building.PopulationCapacity = 0    // No housing
	building.BuildCost = 2000          // 2000 credits to build (expensive)
	building.UpkeepCost = 20           // 20 credits/sec upkeep (high maintenance)
	building.MaxLevel = 5              // Can upgrade to level 5
	building.Size = 6                  // 6 pixels (large industrial structure)
	building.Description = "Constructs and repairs ships. Enables fleet production."
	building.SetWorkersRequired(400)

	return building
}

// CreateShipyard is a helper function to create a shipyard on a specific planet
func CreateShipyard(planetID string, owner string, params entities.GenerationParams) *entities.Building {
	gen := &ShipyardGenerator{}
	building := gen.Generate(params).(*entities.Building)
	building.AttachedTo = planetID
	building.Owner = owner
	return building
}
