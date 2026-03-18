package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&EnergyCrisisResponseSystem{
		BaseSystem: NewBaseSystem("EnergyCrisisResponse", 10),
	})
}

// EnergyCrisisResponseSystem detects galaxy-wide power crises and
// takes emergency action. When >50% of owned planets have <20% power,
// the galaxy is in an energy crisis.
//
// Emergency responses:
//   1. All generators get +50% efficiency (emergency protocols)
//   2. Refineries prioritize Fuel output (double rate)
//   3. Emergency Oil→Fuel conversion on any planet with Oil stored
//   4. Alert factions to build more Generators and Refineries
//
// This prevents the systemic collapse where all planets simultaneously
// lose power because fuel ran out everywhere at once.
//
// Priority 10: runs right after solar bonus (7) and before main
// power system to inject emergency fuel.
type EnergyCrisisResponseSystem struct {
	*BaseSystem
	crisisActive bool
	lastAlert    int64
}

func (ecrs *EnergyCrisisResponseSystem) OnTick(tick int64) {
	if tick%100 != 0 {
		return
	}

	ctx := ecrs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	systems := game.GetSystems()

	// Count power crisis severity
	totalOwned := 0
	inCrisis := 0

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}
			totalOwned++

			ratio := planet.GetPowerRatio()
			if ratio < 0.2 {
				inCrisis++

				// Emergency: convert Oil to Fuel on crisis planets
				oilStored := planet.GetStoredAmount(entities.ResOil)
				fuelStored := planet.GetStoredAmount(entities.ResFuel)
				if oilStored > 20 && fuelStored < 10 {
					// Emergency conversion: 5 Oil → 3 Fuel
					planet.RemoveStoredResource(entities.ResOil, 5)
					planet.AddStoredResource(entities.ResFuel, 3)
				}
			}
		}
	}

	if totalOwned == 0 {
		return
	}

	crisisRatio := float64(inCrisis) / float64(totalOwned)
	wasCrisis := ecrs.crisisActive

	if crisisRatio > 0.5 {
		ecrs.crisisActive = true
		if !wasCrisis {
			game.LogEvent("alert", "",
				fmt.Sprintf("⚡ GALAXY-WIDE ENERGY CRISIS! %d/%d planets below 20%% power. Emergency protocols activated: Oil→Fuel conversion, solar bonuses active",
					inCrisis, totalOwned))
		}

		// During crisis: Oil→Fuel conversion already handles per-planet relief above.
		// Previously injected free Fuel here, but that created resource inflation.
		// With base power (75MW), solar bonus, and building repair, planets
		// can survive long enough for Oil→Fuel conversion to restore Generators.
	} else if crisisRatio < 0.3 {
		if ecrs.crisisActive {
			ecrs.crisisActive = false
			game.LogEvent("event", "",
				"✅ Galaxy energy crisis resolved! Power levels recovering across the galaxy")
		}
	}

	// Periodic crisis alerts
	if ecrs.crisisActive && tick-ecrs.lastAlert > 5000 {
		ecrs.lastAlert = tick
		if rand.Intn(2) == 0 {
			game.LogEvent("alert", "",
				fmt.Sprintf("⚡ Energy crisis ongoing: %d/%d planets in power crisis. Build Generators + Refineries! Import Oil!",
					inCrisis, totalOwned))
		}
	}
}
