package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&MarketDepthSystem{
		BaseSystem: NewBaseSystem("MarketDepth", 204),
	})
}

// MarketDepthSystem analyzes how much of each resource is available
// galaxy-wide and reports "market depth" — how much exists vs how
// much is being consumed. Thin markets are volatile, deep markets
// are stable.
//
// For each resource:
//   Total galaxy stock / Total galaxy consumption per interval
//   = "days of supply"
//
//   < 1 day:  CRITICAL shortage (price should spike)
//   1-3 days: Tight supply
//   3-7 days: Balanced
//   7+ days:  Oversupply (price should drop)
//
// Helps factions identify what to produce more of and what to sell.
type MarketDepthSystem struct {
	*BaseSystem
	nextReport int64
}

func (mds *MarketDepthSystem) OnTick(tick int64) {
	ctx := mds.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if mds.nextReport == 0 {
		mds.nextReport = tick + 5000 + int64(rand.Intn(5000))
	}
	if tick < mds.nextReport {
		return
	}
	mds.nextReport = tick + 8000 + int64(rand.Intn(5000))

	systems := game.GetSystems()

	resources := []string{entities.ResIron, entities.ResWater, entities.ResOil,
		entities.ResFuel, entities.ResRareMetals, entities.ResHelium3, entities.ResElectronics}

	msg := "📊 Market Depth: "
	hasData := false

	for _, res := range resources {
		totalStock := 0
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
					totalStock += planet.GetStoredAmount(res)
				}
			}
		}

		status := "✅"
		label := ""
		switch {
		case totalStock < 50:
			status = "🔴"
			label = "CRITICAL"
		case totalStock < 200:
			status = "🟠"
			label = "tight"
		case totalStock < 1000:
			status = "🟢"
			label = "ok"
		default:
			status = "🔵"
			label = "deep"
		}

		// Only report notable ones
		if totalStock < 200 || totalStock > 2000 {
			msg += fmt.Sprintf("%s%s(%d,%s) ", status, res, totalStock, label)
			hasData = true
		}
	}

	if hasData {
		game.LogEvent("intel", "", msg)
	}
}
