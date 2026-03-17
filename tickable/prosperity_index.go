package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ProsperityIndexSystem{
		BaseSystem: NewBaseSystem("ProsperityIndex", 203),
	})
}

// ProsperityIndexSystem computes a single "Prosperity Index" per
// faction that combines all quality-of-life metrics into one number.
//
// PI = (avg happiness * 40) + (avg power ratio * 20) +
//      (resource diversity * 5) + (tech level * 10) +
//      (pop growth trend * 10)
//
// Scale:
//   80+: Flourishing civilization
//   60-79: Prosperous
//   40-59: Developing
//   20-39: Struggling
//   <20: Failed state
//
// The one number that answers "how well is my empire doing overall?"
type ProsperityIndexSystem struct {
	*BaseSystem
	prevPop    map[string]int64
	nextReport int64
}

func (pis *ProsperityIndexSystem) OnTick(tick int64) {
	ctx := pis.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if pis.prevPop == nil {
		pis.prevPop = make(map[string]int64)
	}

	if pis.nextReport == 0 {
		pis.nextReport = tick + 5000
	}
	if tick < pis.nextReport {
		return
	}
	pis.nextReport = tick + 8000 + int64(rand.Intn(5000))

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, player := range players {
		if player == nil {
			continue
		}

		totalHappiness := 0.0
		totalPower := 0.0
		totalPop := int64(0)
		totalTech := 0.0
		planetCount := 0
		resourceTypes := make(map[string]bool)

		for _, sys := range systems {
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner != player.Name {
					continue
				}
				planetCount++
				totalHappiness += planet.Happiness
				totalPower += planet.GetPowerRatio()
				totalPop += planet.Population
				totalTech += planet.TechLevel

				for _, res := range []string{entities.ResIron, entities.ResWater, entities.ResOil,
					entities.ResFuel, entities.ResRareMetals, entities.ResHelium3, entities.ResElectronics} {
					if planet.GetStoredAmount(res) > 0 {
						resourceTypes[res] = true
					}
				}
			}
		}

		if planetCount == 0 {
			continue
		}

		avgHappiness := totalHappiness / float64(planetCount)
		avgPower := totalPower / float64(planetCount)
		if avgPower > 1.0 {
			avgPower = 1.0
		}
		avgTech := totalTech / float64(planetCount)

		// Pop growth trend
		popGrowth := 0.0
		prev := pis.prevPop[player.Name]
		if prev > 0 {
			popGrowth = float64(totalPop-prev) / float64(prev)
			if popGrowth > 0.1 {
				popGrowth = 0.1
			}
			if popGrowth < -0.1 {
				popGrowth = -0.1
			}
		}
		pis.prevPop[player.Name] = totalPop

		pi := avgHappiness*40 + avgPower*20 + float64(len(resourceTypes))*5 +
			avgTech*10 + (popGrowth+0.1)*50

		status := "Failed state"
		emoji := "💀"
		switch {
		case pi >= 80:
			status = "Flourishing"
			emoji = "🌟"
		case pi >= 60:
			status = "Prosperous"
			emoji = "😊"
		case pi >= 40:
			status = "Developing"
			emoji = "📊"
		case pi >= 20:
			status = "Struggling"
			emoji = "😟"
		}

		game.LogEvent("intel", player.Name,
			fmt.Sprintf("%s %s Prosperity: %.0f (%s) | happy:%.0f%% power:%.0f%% tech:%.1f resources:%d pop:%d",
				emoji, player.Name, pi, status, avgHappiness*100, avgPower*100,
				avgTech, len(resourceTypes), totalPop))
	}
}
