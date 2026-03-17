package tickable

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ProspectingSystem{
		BaseSystem: NewBaseSystem("Prospecting", 57),
	})
}

// ProspectingSystem lets planets discover new resource deposits over time.
// As resource nodes deplete, prospecting provides a way to find replacements
// without sending Colony ships to new systems.
//
// Mechanics:
//   - Planets with a Mine and tech level 2.0+ have a small chance to discover
//     new resource deposits every ~5000 ticks
//   - Higher tech level = better chance of rare resource discovery
//   - Research Labs double the prospecting chance
//   - Discovered resources start at low abundance (10-30) and grow naturally
//
// Resource discovery probabilities (per check):
//   Base: 5% chance
//   With Research Lab: 10% chance
//   Tech 3.0+: +5% chance for Rare Metals or Helium-3
//
// This creates long-term planetary development: mature planets slowly
// gain more resource variety, reducing dependence on imports.
type ProspectingSystem struct {
	*BaseSystem
	nextCheck map[int]int64 // planetID → next prospect tick
}

func (ps *ProspectingSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := ps.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ps.nextCheck == nil {
		ps.nextCheck = make(map[int]int64)
	}

	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.TechLevel < 2.0 {
				continue
			}

			pid := planet.GetID()
			if ps.nextCheck[pid] == 0 {
				ps.nextCheck[pid] = tick + 3000 + int64(rand.Intn(5000))
			}
			if tick < ps.nextCheck[pid] {
				continue
			}
			ps.nextCheck[pid] = tick + 5000 + int64(rand.Intn(8000))

			// Must have at least one mine (mining infrastructure)
			hasMine := false
			hasResearchLab := false
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					if b.BuildingType == entities.BuildingMine && b.IsOperational {
						hasMine = true
					}
					if b.BuildingType == entities.BuildingResearchLab && b.IsOperational {
						hasResearchLab = true
					}
				}
			}
			if !hasMine {
				continue
			}

			// Calculate discovery chance
			chance := 5
			if hasResearchLab {
				chance += 5
			}
			if planet.TechLevel >= 3.0 {
				chance += 5
			}

			if rand.Intn(100) >= chance {
				continue // no discovery this time
			}

			// Discover a new resource!
			ps.discoverResource(planet, game)
		}
	}
}

func (ps *ProspectingSystem) discoverResource(planet *entities.Planet, game GameProvider) {
	// Determine what resources the planet already has
	existing := make(map[string]bool)
	for _, re := range planet.Resources {
		if r, ok := re.(*entities.Resource); ok {
			existing[r.ResourceType] = true
		}
	}

	// Build candidate list (prefer resources the planet doesn't have)
	type candidate struct {
		resType string
		weight  int
	}
	var candidates []candidate

	allResources := []struct {
		resType  string
		baseWeight int
		techReq  float64
	}{
		{entities.ResIron, 30, 0},
		{entities.ResWater, 25, 0},
		{entities.ResOil, 20, 0},
		{entities.ResRareMetals, 10, 2.0},
		{entities.ResHelium3, 8, 2.5},
	}

	for _, r := range allResources {
		if planet.TechLevel < r.techReq {
			continue
		}
		weight := r.baseWeight
		if existing[r.resType] {
			weight /= 5 // much less likely to discover a resource you already have
		}
		if weight > 0 {
			candidates = append(candidates, candidate{r.resType, weight})
		}
	}

	if len(candidates) == 0 {
		return
	}

	// Weighted random selection
	totalWeight := 0
	for _, c := range candidates {
		totalWeight += c.weight
	}
	roll := rand.Intn(totalWeight)
	selected := candidates[0].resType
	for _, c := range candidates {
		roll -= c.weight
		if roll < 0 {
			selected = c.resType
			break
		}
	}

	// Create the new resource deposit
	abundance := 10 + rand.Intn(20) // starts small
	newRes := &entities.Resource{
		BaseEntity: entities.BaseEntity{
			ID:   rand.Intn(900000000) + 100000000,
			Name: fmt.Sprintf("%s Deposit", selected),
			Type: entities.EntityTypeResource,
			Color: resourceColor(selected),
		},
		ResourceType:   selected,
		Abundance:      abundance,
		ExtractionRate: 0.5 + rand.Float64()*0.3,
		Value:          resourceValue(selected),
		Rarity:         prospectRarity(selected),
		Owner:          planet.Owner,
	}

	planet.Resources = append(planet.Resources, newRes)

	game.LogEvent("explore", planet.Owner,
		fmt.Sprintf("⛏️ Prospectors on %s discovered a new %s deposit! (abundance: %d)",
			planet.Name, selected, abundance))
}

func resourceColor(res string) color.RGBA {
	switch res {
	case entities.ResIron:
		return color.RGBA{150, 150, 150, 255}
	case entities.ResWater:
		return color.RGBA{50, 100, 200, 255}
	case entities.ResOil:
		return color.RGBA{40, 40, 40, 255}
	case entities.ResRareMetals:
		return color.RGBA{200, 150, 50, 255}
	case entities.ResHelium3:
		return color.RGBA{150, 220, 255, 255}
	default:
		return color.RGBA{128, 128, 128, 255}
	}
}

func resourceValue(res string) int {
	switch res {
	case entities.ResIron:
		return 10
	case entities.ResWater:
		return 15
	case entities.ResOil:
		return 20
	case entities.ResRareMetals:
		return 50
	case entities.ResHelium3:
		return 80
	default:
		return 10
	}
}

func prospectRarity(res string) string {
	switch res {
	case entities.ResIron, entities.ResWater:
		return "Common"
	case entities.ResOil:
		return "Uncommon"
	case entities.ResRareMetals:
		return "Rare"
	case entities.ResHelium3:
		return "Very Rare"
	default:
		return "Common"
	}
}
