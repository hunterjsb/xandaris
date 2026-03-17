package tickable

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&MarketVolatilitySystem{
		BaseSystem: NewBaseSystem("MarketVolatility", 123),
	})
}

// MarketVolatilitySystem adds price fluctuation to the market,
// making trade timing matter. Prices oscillate around base values
// with random perturbations.
//
// Each resource has a "volatility factor" that changes every 2000 ticks:
//   Iron:        low volatility (±10%)  — stable industrial commodity
//   Water:       low volatility (±10%)  — essential, stable
//   Oil:         medium volatility (±20%) — energy market swings
//   Fuel:        medium volatility (±15%) — follows oil
//   Rare Metals: high volatility (±30%)  — speculation target
//   Helium-3:    high volatility (±35%)  — scarcity-driven
//   Electronics: medium volatility (±25%) — tech demand cycles
//
// Price multipliers are applied via market trade volume adjustments.
// Factions watching price trends (via PriceHistory system) can
// buy low and sell high.
type MarketVolatilitySystem struct {
	*BaseSystem
	factors    map[string]float64 // resource → current price factor
	nextShift  int64
}

var resourceVolatility = map[string]float64{
	entities.ResIron:        0.10,
	entities.ResWater:       0.10,
	entities.ResOil:         0.20,
	entities.ResFuel:        0.15,
	entities.ResRareMetals:  0.30,
	entities.ResHelium3:     0.35,
	entities.ResElectronics: 0.25,
}

func (mvs *MarketVolatilitySystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := mvs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	if mvs.factors == nil {
		mvs.factors = make(map[string]float64)
		for res := range resourceVolatility {
			mvs.factors[res] = 1.0
		}
	}

	if mvs.nextShift == 0 {
		mvs.nextShift = tick + 2000
	}

	// Shift factors periodically
	if tick >= mvs.nextShift {
		mvs.nextShift = tick + 2000 + int64(rand.Intn(1000))

		// Announce significant shifts
		var bigMoves []string

		for res, vol := range resourceVolatility {
			// Random walk: current factor drifts toward 1.0 with noise
			current := mvs.factors[res]
			drift := (1.0 - current) * 0.3 // mean reversion
			noise := (rand.Float64()*2 - 1) * vol
			newFactor := current + drift + noise
			newFactor = math.Max(1.0-vol*1.5, math.Min(1.0+vol*1.5, newFactor))
			mvs.factors[res] = newFactor

			// Apply to market via trade volume pressure
			if newFactor > 1.1 {
				market.AddTradeVolume(res, int(newFactor*20), true) // demand pressure
			} else if newFactor < 0.9 {
				market.AddTradeVolume(res, int((2-newFactor)*20), false) // supply pressure
			}

			// Track big moves
			changePct := (newFactor - current) / current * 100
			if math.Abs(changePct) > 10 {
				dir := "📈"
				if changePct < 0 {
					dir = "📉"
				}
				bigMoves = append(bigMoves, fmt.Sprintf("%s %s %+.0f%%", dir, res, changePct))
			}
		}

		// Announce big market moves
		if len(bigMoves) > 0 && rand.Intn(3) == 0 {
			msg := "📊 Market moves: "
			for _, m := range bigMoves {
				msg += m + " "
			}
			game.LogEvent("intel", "", msg)
		}
	}
}
