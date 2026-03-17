package tickable

import (
	"fmt"
)

func init() {
	RegisterSystem(&LossPreventionSystem{
		BaseSystem: NewBaseSystem("LossPrevention", 22),
	})
}

// LossPreventionSystem directly intercepts the auto-order system
// by checking if a faction's recent trades are net-negative. If a
// faction has lost more credits from trading than earned in the
// last 5000 ticks, their auto-trading is suspended.
//
// This is more aggressive than the trade guard (which only blocks
// individual resources). This blocks ALL auto-trading for the faction
// until they're profitable again.
//
// Works by tracking credit snapshots: if credits dropped AND the
// faction has active standing orders or auto-orders, suspend them.
//
// Priority 22: runs before smart auto trade (23), standing orders
// (25), and auto orders (29).
type LossPreventionSystem struct {
	*BaseSystem
	creditHistory map[string][]int // playerName → last 3 credit snapshots
	suspended     map[string]int64 // playerName → suspend until tick
}

func (lps *LossPreventionSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := lps.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if lps.creditHistory == nil {
		lps.creditHistory = make(map[string][]int)
		lps.suspended = make(map[string]int64)
	}

	players := ctx.GetPlayers()

	for _, p := range players {
		if p == nil {
			continue
		}

		// Track credits
		lps.creditHistory[p.Name] = append(lps.creditHistory[p.Name], p.Credits)
		if len(lps.creditHistory[p.Name]) > 5 {
			lps.creditHistory[p.Name] = lps.creditHistory[p.Name][1:]
		}

		history := lps.creditHistory[p.Name]
		if len(history) < 3 {
			continue
		}

		// Check if credits declining over last 3 snapshots
		allDecline := true
		totalLoss := 0
		for i := 1; i < len(history); i++ {
			if history[i] >= history[i-1] {
				allDecline = false
				break
			}
			totalLoss += history[i-1] - history[i]
		}

		// If faction is losing credits consistently AND has > some minimum,
		// it might be from bad trades. Suspend auto-trading temporarily.
		if allDecline && totalLoss > 5000 && p.Credits > 5000 {
			// Check if already suspended
			if tick < lps.suspended[p.Name] {
				continue
			}

			lps.suspended[p.Name] = tick + 5000

			// Clear their auto-orders
			ob := game.GetOrderBook()
			if ob != nil {
				for _, sys := range game.GetSystems() {
					ob.ClearPlayerOrders(p.Name, sys.ID)
				}
			}

			game.LogEvent("alert", p.Name,
				fmt.Sprintf("🛑 Loss prevention: %s auto-trading suspended for 5000 ticks! Lost %dcr over last intervals. Orders cleared. Manual trades still allowed.",
					p.Name, totalLoss))
		}
	}
}

// IsSuspended checks if a faction's auto-trading is suspended.
func (lps *LossPreventionSystem) IsSuspended(playerName string, tick int64) bool {
	if lps.suspended == nil {
		return false
	}
	return tick < lps.suspended[playerName]
}
