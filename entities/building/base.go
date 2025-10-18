package building

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&BaseGenerator{})
}

// BaseGenerator exists primarily so the base entity is discoverable in the registry, though bases
// are spawned explicitly for each planet and not through random generation.
type BaseGenerator struct{}

func (g *BaseGenerator) GetWeight() float64 {
	return 0.0 // Bases are created explicitly per planet
}

func (g *BaseGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeBuilding
}

func (g *BaseGenerator) GetSubType() string {
	return "Base"
}

func (g *BaseGenerator) Generate(params entities.GenerationParams) entities.Entity {
	id := params.SystemID*1000000 + rand.Intn(100000)

	buildingColor := color.RGBA{
		R: 180,
		G: 200,
		B: 220,
		A: 255,
	}

	base := entities.NewBuilding(
		id,
		fmt.Sprintf("Planetary Base %d", rand.Intn(900)+100),
		"Base",
		params.OrbitDistance,
		params.OrbitAngle,
		buildingColor,
	)

	base.AttachmentType = "Planet"
	base.ProductionBonus = 1.0
	base.PopulationCapacity = 0
	base.BuildCost = 0
	base.UpkeepCost = 0
	base.MaxLevel = 1
	base.Size = 5
	base.Description = "Primary colony infrastructure that makes a planet habitable."

	return base
}

// EnsurePlanetHasBase creates or updates the planetary base building for the provided planet.
func EnsurePlanetHasBase(planet *entities.Planet, params entities.GenerationParams) {
	if planet == nil {
		return
	}

	base := planet.GetBaseBuilding()
	if base == nil {
		entity := (&BaseGenerator{}).Generate(params)
		var ok bool
		base, ok = entity.(*entities.Building)
		if !ok {
			return
		}
		planet.Buildings = append(planet.Buildings, base)
	}

	base.AttachmentType = "Planet"
	base.AttachedTo = fmt.Sprintf("%d", planet.GetID())

	base.PopulationCapacity = calculateBaseHousing(planet)
	if planet.Habitability <= 0 {
		base.PopulationCapacity = 0
		base.IsOperational = false
	} else {
		base.IsOperational = true
	}

	base.Name = fmt.Sprintf("%s Base", planet.Name)
}

func calculateBaseHousing(planet *entities.Planet) int64 {
	if planet == nil {
		return 0
	}

	if planet.Habitability <= 0 {
		return 0
	}

	sizeFactor := float64(planet.Size)
	habitabilityFactor := float64(planet.Habitability) / 100.0

	capacity := int64(sizeFactor * sizeFactor * 2000.0 * habitabilityFactor)
	if capacity < 1000 {
		capacity = 1000
	}

	return capacity
}
