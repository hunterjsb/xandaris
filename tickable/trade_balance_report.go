package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&TradeBalanceReportSystem{
		BaseSystem: NewBaseSystem("TradeBalanceReport", 128),
	})
}

// TradeBalanceReportSystem tracks each faction's imports vs exports
// and announces trade balance status. A positive trade balance means
// the faction exports more than it imports — they're a net producer.
//
// Calculates from recent trade history:
//   Exports = total credits from selling
//   Imports = total credits spent buying
//   Balance = Exports - Imports
//
// Status:
//   Surplus (balance > +5000):  healthy exporter
//   Balanced (±5000):           self-sufficient
//   Deficit (balance < -5000):  depends on imports
//   Critical deficit (<-20000): unsustainable, bleeding credits
type TradeBalanceReportSystem struct {
	*BaseSystem
	nextReport int64
}

func (tbrs *TradeBalanceReportSystem) OnTick(tick int64) {
	ctx := tbrs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tbrs.nextReport == 0 {
		tbrs.nextReport = tick + 5000 + int64(rand.Intn(5000))
	}
	if tick < tbrs.nextReport {
		return
	}
	tbrs.nextReport = tick + 8000 + int64(rand.Intn(5000))

	te := game.GetTradeExecutor()
	if te == nil {
		return
	}

	players := ctx.GetPlayers()
	history := te.GetHistory(100)

	// Aggregate per faction
	type balance struct {
		exports int
		imports int
	}
	factionBalance := make(map[string]*balance)

	for _, record := range history {
		if factionBalance[record.Player] == nil {
			factionBalance[record.Player] = &balance{}
		}
		if record.Action == "sell" {
			factionBalance[record.Player].exports += record.Total
		} else {
			factionBalance[record.Player].imports += record.Total
		}
	}

	if len(factionBalance) == 0 {
		return
	}

	msg := "📊 Trade Balance: "
	hasData := false

	for _, player := range players {
		if player == nil {
			continue
		}
		bal := factionBalance[player.Name]
		if bal == nil {
			continue
		}

		net := bal.exports - bal.imports
		status := "balanced"
		emoji := "⚖️"
		if net > 5000 {
			status = "surplus"
			emoji = "📈"
		} else if net < -20000 {
			status = "CRITICAL deficit"
			emoji = "🚨"
		} else if net < -5000 {
			status = "deficit"
			emoji = "📉"
		}

		msg += fmt.Sprintf("%s %s %s(net:%+dcr) ", emoji, player.Name, status, net)
		hasData = true
	}

	if hasData {
		game.LogEvent("intel", "", msg)
	}
}
