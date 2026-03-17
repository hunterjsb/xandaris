package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&TradeEfficiencyRatingSystem{
		BaseSystem: NewBaseSystem("TradeEfficiencyRating", 155),
	})
}

// TradeEfficiencyRatingSystem rates how efficiently each faction
// converts credits spent on trade into actual value. A faction that
// buys at 50cr and sells at 30cr has terrible efficiency. A faction
// that buys at 20cr and sells at 50cr has great efficiency.
//
// Efficiency = (total sell revenue) / (total buy spending) * 100
//   150%+ = Excellent trader (selling higher than buying)
//   100-150% = Break-even to profitable
//   50-100% = Losing money on trades
//   <50% = Terrible trader (losing >50% on every trade cycle)
//
// Announced per faction every ~8000 ticks with grade.
type TradeEfficiencyRatingSystem struct {
	*BaseSystem
	nextReport int64
}

func (ters *TradeEfficiencyRatingSystem) OnTick(tick int64) {
	ctx := ters.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ters.nextReport == 0 {
		ters.nextReport = tick + 5000 + int64(rand.Intn(5000))
	}
	if tick < ters.nextReport {
		return
	}
	ters.nextReport = tick + 8000 + int64(rand.Intn(5000))

	te := game.GetTradeExecutor()
	if te == nil {
		return
	}

	history := te.GetHistory(100)

	// Aggregate per faction
	type tradeStats struct {
		buySpend  int
		sellEarn  int
		buyCount  int
		sellCount int
	}
	factions := make(map[string]*tradeStats)

	for _, r := range history {
		if factions[r.Player] == nil {
			factions[r.Player] = &tradeStats{}
		}
		if r.Action == "buy" {
			factions[r.Player].buySpend += r.Total
			factions[r.Player].buyCount++
		} else {
			factions[r.Player].sellEarn += r.Total
			factions[r.Player].sellCount++
		}
	}

	for name, stats := range factions {
		if stats.buyCount == 0 && stats.sellCount == 0 {
			continue
		}

		efficiency := 0.0
		if stats.buySpend > 0 {
			efficiency = float64(stats.sellEarn) / float64(stats.buySpend) * 100
		} else if stats.sellEarn > 0 {
			efficiency = 200 // pure seller, great efficiency
		}

		grade := "F"
		switch {
		case efficiency >= 150:
			grade = "A+"
		case efficiency >= 120:
			grade = "A"
		case efficiency >= 100:
			grade = "B"
		case efficiency >= 75:
			grade = "C"
		case efficiency >= 50:
			grade = "D"
		}

		// Only report notable grades
		if stats.buyCount+stats.sellCount < 3 {
			continue
		}

		if efficiency < 80 || efficiency > 130 {
			emoji := "📉"
			if efficiency > 100 {
				emoji = "📈"
			}
			game.LogEvent("logistics", name,
				fmt.Sprintf("%s %s trade efficiency: %.0f%% (grade %s) — bought %d times for %dcr, sold %d times for %dcr",
					emoji, name, efficiency, grade, stats.buyCount, stats.buySpend, stats.sellCount, stats.sellEarn))
		}
	}
}
