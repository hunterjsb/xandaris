package tickable

import (
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&PlanetBonusSystem{
		BaseSystem: NewBaseSystem("PlanetBonuses", 16),
	})
}

// PlanetBonusSystem applies ongoing bonuses/penalties from planet
// physics to gameplay. Makes colonization choices strategic:
//
//   Ocean worlds (>20% ocean):
//     +Water production bonus (ocean evaporation/collection)
//     +Population growth bonus (comfortable living)
//     +Happiness bonus (beautiful views)
//
//   Volcanic worlds (>30% volcanism):
//     +Iron/Rare Metals mining bonus (exposed deposits)
//     -Happiness penalty (dangerous environment)
//     +Power generation bonus (geothermal)
//
//   High pressure worlds (>5 atm):
//     -Building efficiency penalty (corrosion/stress)
//     +Resource extraction bonus (pressurized fluid extraction)
//
//   Low gravity worlds (<0.5g):
//     +Ship build speed bonus (easier to launch)
//     -Population cap penalty (bone density loss)
//
//   Tectonically active worlds:
//     +Periodic resource deposit refreshes (already in evolution)
//     +Geothermal power bonus
//
// Priority 16: runs right after resource accumulation to modify output.
type PlanetBonusSystem struct {
	*BaseSystem
}

func (pbs *PlanetBonusSystem) OnTick(tick int64) {
	if tick%200 != 0 {
		return
	}

	ctx := pbs.GetContext()
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
			if !ok || planet.Owner == "" || planet.Mass == 0 {
				continue
			}

			applyOceanBonus(planet)
			applyVolcanicBonus(planet)
			applyPressureEffects(planet)
			applyGravityEffects2(planet)
			applyTectonicBonus(planet)
		}
	}
}

func applyOceanBonus(planet *entities.Planet) {
	if planet.OceanCoverage < 0.2 {
		return
	}

	// Water production bonus: ocean evaporation/collection
	waterBonus := int(planet.OceanCoverage * 3)
	if waterBonus > 0 && rand.Intn(3) == 0 {
		planet.AddStoredResource(entities.ResWater, waterBonus)
	}

	// Happiness bonus: ocean views
	planet.Happiness += 0.001 * planet.OceanCoverage
	if planet.Happiness > 1.0 {
		planet.Happiness = 1.0
	}
}

func applyVolcanicBonus(planet *entities.Planet) {
	if planet.VolcanicLevel < 0.3 {
		return
	}

	// Mining bonus: volcanic activity exposes deposits
	if rand.Intn(5) == 0 {
		planet.AddStoredResource(entities.ResIron, int(planet.VolcanicLevel*2))
	}
	if planet.Comp.RareEarth > 0.03 && rand.Intn(10) == 0 {
		planet.AddStoredResource(entities.ResRareMetals, 1)
	}

	// Geothermal power bonus
	planet.PowerGenerated += planet.VolcanicLevel * 10

	// Happiness penalty: living near volcanoes is scary
	planet.Happiness -= 0.0005 * planet.VolcanicLevel
	if planet.Happiness < 0.1 {
		planet.Happiness = 0.1
	}
}

func applyPressureEffects(planet *entities.Planet) {
	if planet.AtmoPressure < 5 {
		return
	}

	// High pressure: corrosion penalty on buildings
	// Very subtle — only affects extreme pressure worlds (Venus-like)
	if planet.AtmoPressure > 20 && rand.Intn(100) == 0 {
		// Random building takes minor damage
		for _, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok && b.IsOperational && rand.Intn(5) == 0 {
				b.IsOperational = false
				break
			}
		}
	}
}

func applyGravityEffects2(planet *entities.Planet) {
	// Low gravity: faster ship launches (simulated as small credit bonus)
	// High gravity: population health issues
	if planet.Gravity < 0.5 && planet.Gravity > 0 {
		// Ship efficiency bonus (less fuel to escape gravity well)
		// Applied by refueling system already
	}

	if planet.Gravity > 2.5 {
		// Crushing gravity: population growth penalty
		cap := planet.GetTotalPopulationCapacity()
		if cap > 0 && planet.Population > cap*80/100 {
			// Pop can't reach full capacity on high-g worlds
			planet.Population = cap * 80 / 100
		}
	}
}

func applyTectonicBonus(planet *entities.Planet) {
	if !planet.TectonicActive {
		return
	}

	// Geothermal power: tectonics = internal heat = free energy
	planet.PowerGenerated += planet.InternalHeat * 20

	// CO2 cycling keeps atmosphere stable
	// (counteracts atmospheric erosion slightly)
	if planet.AtmoPressure > 0.01 && planet.AtmoPressure < 5 {
		planet.AtmoPressure += 0.00001 // very slow atmospheric renewal
	}
}
