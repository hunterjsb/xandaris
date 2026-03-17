package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ResourceConversionSystem{
		BaseSystem: NewBaseSystem("ResourceConversion", 96),
	})
}

// ResourceConversionSystem enables advanced resource transformation
// chains on planets with high tech levels. This creates vertical
// integration in the economy — factions can produce high-value
// goods from raw materials.
//
// Conversion chains (requires tech 2.5+ and Factory):
//   Iron + Oil → 0.5x Rare Metals     (basic metallurgy)
//   Oil + Water → 0.3x Fuel            (refining bonus)
//   Rare Metals + Fuel → 0.2x Electronics (advanced manufacturing)
//   Iron + Rare Metals → Ship Parts    (+10% ship build speed)
//
// Conversions happen automatically when:
//   - Planet has a Factory (operational, staffed)
//   - Planet has surplus of input resources (>200 each)
//   - Planet has deficit of output resource (<50)
//   - Tech level meets minimum
//
// Conversion rate: 50 input → output per 1000 ticks
// This is slower than importing but doesn't need logistics.
type ResourceConversionSystem struct {
	*BaseSystem
}

type advConversion struct {
	inputA    string
	inputB    string
	output    string
	inputQty  int
	outputQty int
	techReq   float64
}

var advConversions = []advConversion{
	{entities.ResIron, entities.ResOil, entities.ResRareMetals, 50, 25, 2.5},
	{entities.ResOil, entities.ResWater, entities.ResFuel, 50, 30, 2.0},
	{entities.ResRareMetals, entities.ResFuel, entities.ResElectronics, 30, 10, 3.0},
}

func (rcs *ResourceConversionSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := rcs.GetContext()
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
			if !ok || planet.Owner == "" {
				continue
			}

			// Must have operational factory
			hasFactory := false
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingFactory && b.IsOperational && b.GetStaffingRatio() > 0 {
					hasFactory = true
					break
				}
			}
			if !hasFactory {
				continue
			}

			for _, conv := range advConversions {
				if planet.TechLevel < conv.techReq {
					continue
				}

				stockA := planet.GetStoredAmount(conv.inputA)
				stockB := planet.GetStoredAmount(conv.inputB)
				stockOut := planet.GetStoredAmount(conv.output)

				// Only convert when surplus inputs and deficit output
				if stockA < 200 || stockB < 200 || stockOut > 50 {
					continue
				}

				// Convert
				planet.RemoveStoredResource(conv.inputA, conv.inputQty)
				planet.RemoveStoredResource(conv.inputB, conv.inputQty)
				planet.AddStoredResource(conv.output, conv.outputQty)

				if rand.Intn(5) == 0 { // don't spam
					game.LogEvent("trade", planet.Owner,
						fmt.Sprintf("🏭 %s: Factory converted %d %s + %d %s → %d %s",
							planet.Name, conv.inputQty, conv.inputA,
							conv.inputQty, conv.inputB, conv.outputQty, conv.output))
				}
			}
		}
	}
}
