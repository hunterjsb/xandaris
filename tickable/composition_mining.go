package tickable

import (
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&CompositionMiningSystem{
		BaseSystem: NewBaseSystem("CompositionMining", 17),
	})
}

// CompositionMiningSystem applies composition-based bonuses to
// resource extraction. Planets with high iron composition extract
// more iron per tick. Water-rich planets extract more water. etc.
//
// Bonus rate: composition_fraction × base_rate × 0.5
//
// Example: A planet with 30% Iron composition and a Mine produces
// Iron at base rate + (0.30 × base × 0.5) = +15% bonus.
//
// This creates natural specialization: some planets are better
// for mining iron, others for water. Trade becomes necessary
// because no planet excels at everything.
//
// Priority 17: runs right after resource accumulation (16).
type CompositionMiningSystem struct {
	*BaseSystem
}

func (cms *CompositionMiningSystem) OnTick(tick int64) {
	if tick%100 != 0 {
		return
	}

	ctx := cms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.Mass == 0 {
				continue // skip legacy planets
			}

			comp := planet.Comp

			// For each resource, apply composition bonus
			type bonus struct {
				res      string
				fraction float64
			}
			bonuses := []bonus{
				{entities.ResIron, comp.Iron},
				{entities.ResWater, comp.Water},
				{entities.ResOil, comp.Organics},
				{entities.ResRareMetals, comp.RareEarth * 10}, // amplify rare earth
				{entities.ResHelium3, comp.Gas * 0.5},
			}

			for _, b := range bonuses {
				if b.fraction < 0.05 {
					continue
				}

				// Check if planet has a mine producing this resource
				hasMine := false
				for _, be := range planet.Buildings {
					if bld, ok := be.(*entities.Building); ok && bld.BuildingType == entities.BuildingMine && bld.IsOperational {
						// Check if mine is on a matching resource
						for _, re := range planet.Resources {
							if r, ok := re.(*entities.Resource); ok && r.ResourceType == b.res && bld.ResourceNodeID == r.GetID() {
								hasMine = true
								break
							}
						}
					}
					if hasMine {
						break
					}
				}

				if !hasMine {
					continue
				}

				// Composition bonus: inject extra resources proportional to fraction
				// 0.30 fraction = 30% of base rate as bonus ≈ 1 extra unit per 100 ticks
				bonusQty := int(b.fraction * 3)
				if bonusQty < 1 && rand.Intn(3) == 0 {
					bonusQty = 1
				}
				if bonusQty > 0 {
					planet.AddStoredResource(b.res, bonusQty)
				}
			}
		}
	}
}
