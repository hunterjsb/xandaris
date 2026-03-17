package tickable

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&EconomicCycleSystem{
		BaseSystem: NewBaseSystem("EconomicCycles", 63),
	})
}

// EconomicCycleSystem simulates galactic-scale economic cycles that
// affect all factions. The economy oscillates between growth and
// recession on a long-term sine wave.
//
// Cycle phases (period ~40,000 ticks ≈ 66 minutes):
//   Expansion:   +20% credit generation, +10% population growth, prices fall
//   Peak:        Maximum prosperity, inflation starts
//   Contraction: -15% credit generation, prices rise, building costs +25%
//   Trough:      Minimum activity, but opportunities for bargain purchases
//
// The cycle affects:
//   - Domestic credit generation (economy/credit_production.go multiplier)
//   - Building construction costs
//   - Market prices (base price adjustment)
//   - Population growth rate
//
// Smart factions build during troughs (cheap), sell during peaks (expensive).
// This creates real economic strategy beyond just "make number go up".
type EconomicCycleSystem struct {
	*BaseSystem
	cyclePosition float64 // 0.0 to 2π (sine wave position)
	lastPhase     string
}

func (ecs *EconomicCycleSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := ecs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	// Advance cycle: full rotation every ~40,000 ticks
	ecs.cyclePosition += 0.08 // ~500 * 80 increments = 40,000 ticks per cycle
	if ecs.cyclePosition > 2*math.Pi {
		ecs.cyclePosition -= 2 * math.Pi
	}

	// Calculate current economic multiplier: 0.85 to 1.20
	multiplier := 1.0 + 0.175*math.Sin(ecs.cyclePosition)

	// Determine phase
	phase := ecs.getPhase()

	// Announce phase transitions
	if phase != ecs.lastPhase && ecs.lastPhase != "" {
		ecs.announcePhaseChange(phase, multiplier, game)
	}
	ecs.lastPhase = phase

	// Apply effects
	ecs.applyEffects(multiplier, phase, game)
}

func (ecs *EconomicCycleSystem) getPhase() string {
	sin := math.Sin(ecs.cyclePosition)
	switch {
	case sin > 0.5:
		return "Expansion"
	case sin > 0:
		return "Peak"
	case sin > -0.5:
		return "Contraction"
	default:
		return "Trough"
	}
}

func (ecs *EconomicCycleSystem) announcePhaseChange(phase string, multiplier float64, game GameProvider) {
	switch phase {
	case "Expansion":
		game.LogEvent("event", "",
			fmt.Sprintf("📈 ECONOMIC EXPANSION! Markets growing — credit generation +%.0f%%, prices falling. Build and invest!",
				(multiplier-1)*100))
	case "Peak":
		game.LogEvent("event", "",
			"📊 Economy at PEAK. Maximum prosperity but inflation looming. Sell high before the downturn!")
	case "Contraction":
		game.LogEvent("event", "",
			fmt.Sprintf("📉 ECONOMIC CONTRACTION! Markets cooling — credit generation %.0f%%, prices rising. Tighten budgets!",
				(multiplier-1)*100))
	case "Trough":
		game.LogEvent("event", "",
			"📉 Economy at TROUGH. Hard times — but building costs are cheapest now. Smart investors buy the dip!")
	}
}

func (ecs *EconomicCycleSystem) applyEffects(multiplier float64, phase string, game GameProvider) {
	players := game.GetPlayers()
	systems := game.GetSystems()

	// Credit generation modifier
	for _, player := range players {
		if player == nil {
			continue
		}

		// Small credit adjustment based on cycle
		if multiplier > 1.0 {
			// Expansion: bonus credits
			bonus := int(float64(player.Credits) * 0.0001 * (multiplier - 1.0))
			if bonus > 100 {
				bonus = 100 // cap per tick
			}
			player.Credits += bonus
		} else {
			// Contraction: small drain (tax/inflation)
			drain := int(float64(player.Credits) * 0.00005 * (1.0 - multiplier))
			if drain > 50 {
				drain = 50
			}
			player.Credits -= drain
			if player.Credits < 0 {
				player.Credits = 0
			}
		}
	}

	// Population growth modifier during expansion
	if phase == "Expansion" && rand.Intn(3) == 0 {
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" && planet.Population > 100 {
					cap := planet.GetTotalPopulationCapacity()
					if cap > 0 && planet.Population < cap {
						bonus := int64(50 + rand.Intn(100))
						if planet.Population+bonus > cap {
							bonus = cap - planet.Population
						}
						planet.Population += bonus
					}
				}
			}
		}
	}
}

// GetEconomicMultiplier returns the current economic cycle multiplier.
func (ecs *EconomicCycleSystem) GetEconomicMultiplier() float64 {
	return 1.0 + 0.175*math.Sin(ecs.cyclePosition)
}

// GetPhase returns the current economic phase name.
func (ecs *EconomicCycleSystem) GetPhase() string {
	return ecs.getPhase()
}
