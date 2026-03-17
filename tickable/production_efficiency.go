package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ProductionEfficiencySystem{
		BaseSystem: NewBaseSystem("ProductionEfficiency", 146),
	})
}

// ProductionEfficiencySystem rewards planets that maintain high
// operational efficiency. A planet where all buildings are staffed,
// powered, and operational gets a production bonus.
//
// Efficiency = (operational buildings / total buildings) * staffing * power
//
// Bonuses at high efficiency:
//   90%+: +10% resource production
//   95%+: +20% resource production + "Model Colony" event
//   100%: +30% + small credit bonus
//
// Low efficiency (<50%) triggers a warning with specific issues.
type ProductionEfficiencySystem struct {
	*BaseSystem
	nextReport int64
}

func (pes *ProductionEfficiencySystem) OnTick(tick int64) {
	if tick%3000 != 0 {
		return
	}

	ctx := pes.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if pes.nextReport == 0 {
		pes.nextReport = tick + 5000
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.Population < 500 {
				continue
			}

			totalBuildings := 0
			operational := 0
			totalStaffing := 0.0

			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					totalBuildings++
					if b.IsOperational {
						operational++
						totalStaffing += b.GetStaffingRatio()
					}
				}
			}

			if totalBuildings < 3 {
				continue
			}

			opRatio := float64(operational) / float64(totalBuildings)
			avgStaffing := totalStaffing / float64(totalBuildings)
			powerRatio := planet.GetPowerRatio()
			if powerRatio > 1.0 {
				powerRatio = 1.0
			}

			efficiency := opRatio * avgStaffing * powerRatio

			// Apply bonuses for high efficiency
			if efficiency >= 0.95 {
				// Small resource bonus
				for _, re := range planet.Resources {
					if r, ok := re.(*entities.Resource); ok && r.Abundance > 0 {
						planet.AddStoredResource(r.ResourceType, 1)
					}
				}

				// Credit bonus for perfect efficiency
				if efficiency >= 1.0 {
					for _, p := range players {
						if p != nil && p.Name == planet.Owner {
							p.Credits += 25
							break
						}
					}
				}
			}

			// Report high/low efficiency periodically
			if tick >= pes.nextReport && rand.Intn(5) == 0 {
				if efficiency >= 0.95 {
					game.LogEvent("logistics", planet.Owner,
						fmt.Sprintf("⭐ %s is a Model Colony! %.0f%% efficiency (%.0f%% operational, %.0f%% staffed, %.0f%% powered)",
							planet.Name, efficiency*100, opRatio*100, avgStaffing*100, planet.GetPowerRatio()*100))
				} else if efficiency < 0.5 && totalBuildings >= 3 {
					game.LogEvent("alert", planet.Owner,
						fmt.Sprintf("📉 %s: low efficiency (%.0f%%). Issues: %.0f%% buildings operational, %.0f%% staffed, %.0f%% power",
							planet.Name, efficiency*100, opRatio*100, avgStaffing*100, planet.GetPowerRatio()*100))
				}
			}
		}
	}

	if tick >= pes.nextReport {
		pes.nextReport = tick + 10000 + int64(rand.Intn(5000))
	}
}
