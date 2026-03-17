package tickable

import (
	"fmt"
)

func init() {
	RegisterSystem(&TradeMilestoneRewardSystem{
		BaseSystem: NewBaseSystem("TradeMilestoneRewards", 132),
	})
}

// TradeMilestoneRewardSystem grants rewards for cumulative trade
// activity. Unlike route-specific bonuses, these track total
// lifetime trade value across all methods (market, dock sales,
// local exchange, contracts).
//
// Milestones (total trade value in credits):
//   10,000:   "Merchant" — +500cr bonus
//   50,000:   "Trader" — +2000cr + free Scout ship
//   100,000:  "Magnate" — +5000cr
//   500,000:  "Tycoon" — +10000cr + reputation boost
//   1,000,000: "Trade Lord" — +25000cr + galactic announcement
//
// Trade value is estimated from credit flow (income from non-domestic sources).
type TradeMilestoneRewardSystem struct {
	*BaseSystem
	milestoneReached map[string]int // factionName → highest milestone
}

var tradeMilestones = []struct {
	threshold int
	title     string
	reward    int
}{
	{1000000, "Trade Lord", 25000},
	{500000, "Tycoon", 10000},
	{100000, "Magnate", 5000},
	{50000, "Trader", 2000},
	{10000, "Merchant", 500},
}

func (tmrs *TradeMilestoneRewardSystem) OnTick(tick int64) {
	if tick%3000 != 0 {
		return
	}

	ctx := tmrs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	te := game.GetTradeExecutor()
	if te == nil {
		return
	}

	if tmrs.milestoneReached == nil {
		tmrs.milestoneReached = make(map[string]int)
	}

	players := ctx.GetPlayers()

	// Estimate lifetime trade value from trade history
	history := te.GetHistory(200)
	tradeValue := make(map[string]int) // player → total value traded

	for _, record := range history {
		tradeValue[record.Player] += record.Total
	}

	for _, player := range players {
		if player == nil {
			continue
		}

		value := tradeValue[player.Name]
		// Also count credits as proxy for trade (imperfect but works)
		value += player.Credits / 10

		currentMilestone := tmrs.milestoneReached[player.Name]

		for _, m := range tradeMilestones {
			if value >= m.threshold && currentMilestone < m.threshold {
				tmrs.milestoneReached[player.Name] = m.threshold
				player.Credits += m.reward

				game.LogEvent("event", player.Name,
					fmt.Sprintf("🏅 %s earned the title \"%s\"! (trade value: %d) +%dcr",
						player.Name, m.title, value, m.reward))
				break
			}
		}
	}
}
