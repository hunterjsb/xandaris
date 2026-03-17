package tickable

import (
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SolarBonusSystem{
		BaseSystem: NewBaseSystem("SolarBonus", 7),
	})
}

// SolarBonusSystem provides free baseline solar power to all planets.
// This prevents the total power death spiral where generators need
// fuel but fuel production needs power.
//
// Every owned planet gets free solar power based on proximity to star:
//   Inner orbit (orbit < 100): 30 MW free solar
//   Mid orbit (100-200):       20 MW free solar
//   Outer orbit (200+):        10 MW free solar
//
// This doesn't replace generators — 30 MW is enough to keep a Mine
// and basic infrastructure running, but not enough for factories
// and advanced buildings. It's the safety net that prevents total
// economic collapse from fuel shortage.
//
// Priority 7: runs before the main power system (10) to seed
// minimum power before consumption is calculated.
type SolarBonusSystem struct {
	*BaseSystem
}

func (sbs *SolarBonusSystem) OnTick(tick int64) {
	if tick%10 != 0 {
		return
	}

	ctx := sbs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			// Free solar power based on orbit distance
			solarMW := 10.0
			if planet.OrbitDistance < 100 {
				solarMW = 30.0
			} else if planet.OrbitDistance < 200 {
				solarMW = 20.0
			}

			planet.PowerGenerated += solarMW
		}
	}
}
