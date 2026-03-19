package tickable

import (
	"math"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&PlanetaryEvolutionSystem{
		BaseSystem: NewBaseSystem("PlanetaryEvolution", 4),
	})
}

// PlanetaryEvolutionSystem makes planets evolve over game time.
// Nothing is frozen after generation — planetary properties slowly
// change based on ongoing physical processes.
//
// Processes modeled:
//   Atmospheric erosion: planets without magnetic fields slowly lose
//     atmosphere (pressure decreases over time). Solar wind strips it.
//
//   Geological cooling: internal heat decreases over time (radiogenic
//     decay). Volcanic activity diminishes. Eventually tectonics stop.
//
//   Hydrosphere changes: as temperature changes from greenhouse/cooling,
//     water state shifts (ice melts, oceans evaporate, etc.)
//
//   Resource regeneration: tectonic activity exposes new mineral deposits.
//     Volcanic eruptions create new resource nodes. Dead planets don't
//     regenerate resources.
//
// All changes are VERY slow (noticeable over thousands of ticks).
// Priority 4: runs before everything else to set physical state.
type PlanetaryEvolutionSystem struct {
	*BaseSystem
}

func (pes *PlanetaryEvolutionSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := pes.GetContext()
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
			if !ok || planet.Mass == 0 {
				continue // skip legacy planets
			}

			pes.evolve(planet)
		}
	}
}

func (pes *PlanetaryEvolutionSystem) evolve(planet *entities.Planet) {
	// Atmospheric erosion: no magnetic field = slow pressure loss
	if planet.MagneticField < 0.2 && planet.AtmoPressure > 0.001 {
		erosionRate := (0.2 - planet.MagneticField) * 0.0001
		planet.AtmoPressure -= planet.AtmoPressure * erosionRate
		if planet.AtmoPressure < 0 {
			planet.AtmoPressure = 0
		}
	}

	// Geological cooling: internal heat slowly decreases
	if planet.InternalHeat > 0.005 {
		coolingRate := 0.00001 // very slow
		// Larger planets cool slower (more thermal mass)
		coolingRate /= math.Max(1, planet.Mass)
		planet.InternalHeat -= coolingRate
		if planet.InternalHeat < 0 {
			planet.InternalHeat = 0
		}
	}

	// Volcanism follows internal heat
	if planet.InternalHeat < 0.02 && planet.VolcanicLevel > 0 {
		planet.VolcanicLevel *= 0.9999 // slow decline
	}

	// Tectonics shut down when internal heat drops too low
	if planet.TectonicActive && planet.InternalHeat < 0.02 {
		planet.TectonicActive = false
	}

	// Magnetic field weakens as core cools
	if planet.InternalHeat < 0.03 && planet.MagneticField > 0 {
		planet.MagneticField *= 0.99999
	}

	// Hydrosphere responds to temperature changes
	if planet.Comp.Water > 0.01 {
		ocean, ice := computeHydro(planet.Comp.Water, planet.Temperature, planet.AtmoPressure)
		// Smooth transition (don't jump)
		planet.OceanCoverage = planet.OceanCoverage*0.99 + ocean*0.01
		planet.IceCoverage = planet.IceCoverage*0.99 + ice*0.01
	}

	// Resource regeneration from tectonic activity
	if planet.TectonicActive && rand.Intn(10000) == 0 {
		// Tectonics expose new deposits — boost existing resource abundance
		for _, re := range planet.Resources {
			if r, ok := re.(*entities.Resource); ok {
				if r.Abundance < 100 {
					r.Abundance += 1 + rand.Intn(3)
					if r.Abundance > 100 {
						r.Abundance = 100
					}
				}
			}
		}
	}
}

func computeHydro(waterFrac float64, tempC int, pressure float64) (ocean, ice float64) {
	if waterFrac < 0.01 || pressure < 0.006 {
		return 0, 0
	}
	if tempC > 100 {
		return 0, 0
	}
	if tempC < 0 {
		ice = waterFrac * 2
		if ice > 1 {
			ice = 1
		}
		if pressure > 0.5 && tempC > -40 {
			ocean = waterFrac * 0.1
		}
		return
	}
	ocean = waterFrac * 3
	if ocean > 0.95 {
		ocean = 0.95
	}
	if tempC < 30 {
		ice = math.Max(0, (30-float64(tempC))/100) * waterFrac
	}
	return
}
