package tickable

import (
	"fmt"
	"math"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticMarketIndexSystem{
		BaseSystem: NewBaseSystem("GalacticMarketIndex", 87),
	})
}

// GalacticMarketIndexSystem computes a single number representing the
// overall health of the galactic economy — the GMI. Like a stock market
// index, the GMI combines multiple economic indicators into one score.
//
// GMI components (each 0-200, weighted):
//   Trade Volume (30%):  based on market trade activity
//   Credit Flow (25%):   total credits across all factions
//   Resource Health (20%): average resource diversity across planets
//   Fleet Activity (15%): cargo ships in transit / total cargo ships
//   Population Growth (10%): total population trend
//
// GMI = 100 is "normal". >120 = booming. <80 = recession.
// >150 = bubble (crash risk). <50 = depression.
//
// The GMI is announced every ~5000 ticks with trend direction.
// It's the single most important number for understanding the economy.
type GalacticMarketIndexSystem struct {
	*BaseSystem
	history    []float64 // last 10 GMI values
	nextReport int64
	lastPop    int64 // total pop at last snapshot
}

func (gmi *GalacticMarketIndexSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := gmi.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if gmi.nextReport == 0 {
		gmi.nextReport = tick + 3000
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	market := game.GetMarketEngine()

	// Calculate components
	tradeScore := 100.0
	if market != nil {
		// Use price stability as proxy for trade health
		resources := []string{entities.ResIron, entities.ResWater, entities.ResOil}
		avgPrice := 0.0
		for _, res := range resources {
			avgPrice += market.GetSellPrice(res)
		}
		avgPrice /= float64(len(resources))
		// Normalize: 20cr avg = 100, higher = more trade
		tradeScore = math.Min(200, avgPrice*5)
	}

	// Credit flow: total credits normalized
	totalCredits := 0
	for _, p := range players {
		if p != nil {
			totalCredits += p.Credits
		}
	}
	creditScore := math.Min(200, float64(totalCredits)/100000*100)

	// Resource health: average diversity
	totalDiversity := 0
	planetCount := 0
	totalPop := int64(0)
	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}
			planetCount++
			totalPop += planet.Population
			diversity := 0
			for _, res := range []string{entities.ResIron, entities.ResWater, entities.ResOil,
				entities.ResFuel, entities.ResRareMetals, entities.ResHelium3, entities.ResElectronics} {
				if planet.GetStoredAmount(res) > 0 {
					diversity++
				}
			}
			totalDiversity += diversity
		}
	}
	resourceScore := 100.0
	if planetCount > 0 {
		avgDiversity := float64(totalDiversity) / float64(planetCount)
		resourceScore = math.Min(200, avgDiversity/7*200) // 7 resources = 200
	}

	// Fleet activity
	totalCargo := 0
	activeCargo := 0
	for _, p := range players {
		if p == nil {
			continue
		}
		for _, ship := range p.OwnedShips {
			if ship != nil && ship.ShipType == entities.ShipTypeCargo {
				totalCargo++
				if ship.Status == entities.ShipStatusMoving || ship.GetTotalCargo() > 0 {
					activeCargo++
				}
			}
		}
	}
	fleetScore := 100.0
	if totalCargo > 0 {
		fleetScore = float64(activeCargo) / float64(totalCargo) * 200
	}

	// Population trend
	popScore := 100.0
	if gmi.lastPop > 0 {
		growth := float64(totalPop-gmi.lastPop) / float64(gmi.lastPop)
		popScore = 100 + growth*1000 // amplify small changes
		popScore = math.Max(0, math.Min(200, popScore))
	}
	gmi.lastPop = totalPop

	// Weighted GMI
	index := tradeScore*0.30 + creditScore*0.25 + resourceScore*0.20 +
		fleetScore*0.15 + popScore*0.10

	// Record history
	gmi.history = append(gmi.history, index)
	if len(gmi.history) > 10 {
		gmi.history = gmi.history[1:]
	}

	// Report
	if tick >= gmi.nextReport {
		gmi.nextReport = tick + 5000

		trend := "→"
		if len(gmi.history) >= 2 {
			prev := gmi.history[len(gmi.history)-2]
			if index > prev+5 {
				trend = "📈"
			} else if index < prev-5 {
				trend = "📉"
			}
		}

		status := "Normal"
		switch {
		case index > 150:
			status = "BUBBLE ⚠️"
		case index > 120:
			status = "Booming"
		case index < 50:
			status = "Depression"
		case index < 80:
			status = "Recession"
		}

		game.LogEvent("intel", "",
			fmt.Sprintf("📊 Galactic Market Index: %.0f %s (%s) | Trade:%.0f Credit:%.0f Resources:%.0f Fleet:%.0f Pop:%.0f",
				index, trend, status, tradeScore, creditScore, resourceScore, fleetScore, popScore))
	}
}

// GetGMI returns the current Galactic Market Index value.
func (gmi *GalacticMarketIndexSystem) GetGMI() float64 {
	if len(gmi.history) == 0 {
		return 100
	}
	return gmi.history[len(gmi.history)-1]
}
