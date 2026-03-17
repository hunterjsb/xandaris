package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&WealthTaxSystem{
		BaseSystem: NewBaseSystem("WealthTax", 137),
	})
}

// WealthTaxSystem applies a small galactic wealth tax that funds
// public goods (galactic infrastructure, supply depots, etc).
// This creates a natural credit ceiling and funds underdog bonuses.
//
// Tax brackets:
//   0-100K credits:    0% (exempt)
//   100K-500K:         0.1% per interval
//   500K-1M:           0.2% per interval
//   1M+:               0.3% per interval (cap: 500cr per interval)
//
// Tax revenue goes into a "galactic fund" that:
//   - Funds supply depot restocking
//   - Contributes to megaproject pools
//   - Provides underdog grants
//
// This prevents infinite credit accumulation and creates a reason
// to SPEND credits (invest in planets/ships rather than hoard).
type WealthTaxSystem struct {
	*BaseSystem
	galacticFund int
	nextReport   int64
}

func (wts *WealthTaxSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := wts.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if wts.nextReport == 0 {
		wts.nextReport = tick + 10000
	}

	players := ctx.GetPlayers()
	totalTax := 0

	for _, p := range players {
		if p == nil || p.Credits <= 100000 {
			continue
		}

		var tax int
		switch {
		case p.Credits > 1000000:
			tax = p.Credits / 333 // 0.3%
		case p.Credits > 500000:
			tax = p.Credits / 500 // 0.2%
		default:
			tax = p.Credits / 1000 // 0.1%
		}

		if tax > 500 {
			tax = 500
		}
		if tax <= 0 {
			continue
		}

		p.Credits -= tax
		totalTax += tax
		wts.galacticFund += tax
	}

	// Distribute fund to underdogs
	if wts.galacticFund > 1000 {
		// Find poorest faction
		var poorest string
		poorestCredits := 999999999
		for _, p := range players {
			if p != nil && p.Credits < poorestCredits {
				poorestCredits = p.Credits
				poorest = p.Name
			}
		}

		if poorest != "" && poorestCredits < 50000 {
			grant := wts.galacticFund / 2
			if grant > 2000 {
				grant = 2000
			}
			wts.galacticFund -= grant
			for _, p := range players {
				if p != nil && p.Name == poorest {
					p.Credits += grant
					break
				}
			}
		}
	}

	// Periodic report
	if tick >= wts.nextReport {
		wts.nextReport = tick + 15000 + int64(rand.Intn(10000))
		if totalTax > 0 {
			game.LogEvent("intel", "",
				fmt.Sprintf("🏛️ Galactic wealth tax collected %dcr this interval. Fund balance: %dcr. Invest your credits!",
					totalTax, wts.galacticFund))
		}
	}
}
