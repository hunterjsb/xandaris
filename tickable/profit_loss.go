package tickable

import (
	"fmt"
)

func init() {
	RegisterSystem(&ProfitLossSystem{
		BaseSystem: NewBaseSystem("ProfitLoss", 71),
	})
}

// ProfitLossSystem tracks credit flow per faction and alerts when a
// faction is losing money faster than earning it. This is the financial
// early warning system.
//
// Every 2000 ticks, it snapshots each faction's credits and compares
// to the previous snapshot. If credits decreased, it calculates the
// burn rate and estimates how long until bankruptcy.
//
// Alerts:
//   - "Burning 500cr/interval — bankrupt in ~20 minutes"
//   - "Credits growing +200cr/interval — economy healthy"
//   - "Break-even — income matches expenses"
//
// This helps LLM agents stop bad strategies (like Llama's losing trades)
// and helps human players understand their economic health.
type ProfitLossSystem struct {
	*BaseSystem
	snapshots map[string][]int // playerName → last 5 credit snapshots
	lastCheck int64
}

func (pls *ProfitLossSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := pls.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if pls.snapshots == nil {
		pls.snapshots = make(map[string][]int)
	}

	players := ctx.GetPlayers()

	for _, player := range players {
		if player == nil {
			continue
		}

		// Record snapshot
		if pls.snapshots[player.Name] == nil {
			pls.snapshots[player.Name] = []int{}
		}
		pls.snapshots[player.Name] = append(pls.snapshots[player.Name], player.Credits)
		if len(pls.snapshots[player.Name]) > 5 {
			pls.snapshots[player.Name] = pls.snapshots[player.Name][1:]
		}

		snaps := pls.snapshots[player.Name]
		if len(snaps) < 2 {
			continue
		}

		current := snaps[len(snaps)-1]
		previous := snaps[len(snaps)-2]
		delta := current - previous

		// Only report significant changes or danger
		if delta < -1000 {
			// Losing money fast
			burnRate := -delta // positive number
			minutesLeft := 0
			if burnRate > 0 && current > 0 {
				intervalsLeft := current / burnRate
				minutesLeft = intervalsLeft * 2000 / 600 // 2000 ticks per interval, 600 ticks per minute
			}

			msg := fmt.Sprintf("💸 %s: losing %dcr per interval (credits: %d",
				player.Name, burnRate, current)
			if minutesLeft > 0 && minutesLeft < 120 {
				msg += fmt.Sprintf(", bankrupt in ~%d min", minutesLeft)
			}
			msg += "). Review expenses!"
			game.LogEvent("alert", player.Name, msg)
		} else if delta > 5000 {
			// Only log big gains (not every small +)
			game.LogEvent("logistics", player.Name,
				fmt.Sprintf("💰 %s: credits growing +%dcr per interval (total: %d). Economy healthy!",
					player.Name, delta, current))
		}

		// Trend analysis: if all 5 snapshots are declining, extra warning
		if len(snaps) >= 5 {
			allDecline := true
			for i := 1; i < len(snaps); i++ {
				if snaps[i] >= snaps[i-1] {
					allDecline = false
					break
				}
			}
			if allDecline {
				totalLoss := snaps[0] - snaps[len(snaps)-1]
				game.LogEvent("alert", player.Name,
					fmt.Sprintf("⚠️ %s: sustained credit decline! Lost %dcr over last 5 intervals. Immediate action needed!",
						player.Name, totalLoss))
			}
		}
	}
}
