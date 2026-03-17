package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TerraformingSystem{
		BaseSystem: NewBaseSystem("Terraforming", 35),
	})
}

// TerraformingSystem slowly improves planet habitability over time.
// Planets with population and Water supply gradually terraform:
// - Habitability increases +1 per 5000 ticks (~8 min) if Water > 100
// - Higher population speeds terraforming (more workers)
// - Extremely high habitability (>80) can generate new Water deposits
//
// This means barren planets colonized early can become lush paradises
// over time — rewarding long-term investment in a colony.
type TerraformingSystem struct {
	*BaseSystem
}

func (ts *TerraformingSystem) OnTick(tick int64) {
	if tick%5000 != 0 {
		return
	}

	ctx := ts.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	for _, sys := range game.GetSystems() {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.Population <= 0 {
				continue
			}

			// Need Water for terraforming
			water := planet.GetStoredAmount(entities.ResWater)
			if water < 100 {
				continue
			}

			// Habitability cap at 95
			if planet.Habitability >= 95 {
				continue
			}

			// Consume some Water for terraforming
			planet.RemoveStoredResource(entities.ResWater, 50)

			// Pop bonus: larger colonies terraform faster
			popBonus := 1
			if planet.Population > 10000 {
				popBonus = 2
			}
			if planet.Population > 50000 {
				popBonus = 3
			}

			planet.Habitability += popBonus
			if planet.Habitability > 95 {
				planet.Habitability = 95
			}

			// At high habitability, chance to generate a Water deposit
			if planet.Habitability > 80 && rand.Intn(5) == 0 {
				for _, res := range planet.Resources {
					if r, ok := res.(*entities.Resource); ok && r.ResourceType == entities.ResWater {
						r.Abundance += 5
						game.LogEvent("event", planet.Owner,
							fmt.Sprintf("🌍 Terraforming on %s created new Water springs! (+5 abundance, hab=%d%%)",
								planet.Name, planet.Habitability))
						break
					}
				}
			}

			if popBonus > 1 {
				game.LogEvent("event", planet.Owner,
					fmt.Sprintf("🌍 %s terraforming progress: habitability now %d%%",
						planet.Name, planet.Habitability))
			}
		}
	}
}
