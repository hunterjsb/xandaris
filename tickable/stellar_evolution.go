package tickable

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&StellarEvolutionSystem{
		BaseSystem: NewBaseSystem("StellarEvolution", 3),
	})
}

// StellarEvolutionSystem makes stars age and change over game time.
// Stars are not static — they evolve through stages:
//
//   Main sequence → gradual luminosity increase
//   Red dwarf → extremely slow (stable for 100 Gyr)
//   Red giant → luminosity fluctuations, mass loss
//   White dwarf → gradual cooling
//   Blue giant → rapid evolution, flare activity
//
// Effects on planets:
//   - Luminosity changes shift the habitable zone
//   - Flare events damage unshielded planet atmospheres
//   - Mass loss reduces gravitational hold on outer planets
//
// All changes are very slow (star aging maps game ticks to Myr).
// 1 Myr ≈ 100,000 ticks. Star lifetime is millions to billions of years.
type StellarEvolutionSystem struct {
	*BaseSystem
	lastFlare map[int]int64 // starID → last flare tick
}

func (ses *StellarEvolutionSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := ses.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ses.lastFlare == nil {
		ses.lastFlare = make(map[int]int64)
	}

	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			star, ok := e.(*entities.Star)
			if !ok {
				continue
			}

			ses.evolveStar(tick, star, sys, game)
		}
	}
}

func (ses *StellarEvolutionSystem) evolveStar(tick int64, star *entities.Star, sys *entities.System, game GameProvider) {
	// Age the star (1 Myr per 100,000 ticks = 0.00001 Gyr per tick at tick%1000)
	ageRate := 0.00001
	star.Age += ageRate

	switch star.StarType {
	case "Main Sequence":
		// Luminosity slowly increases (~10% per Gyr for Sun-like)
		star.Luminosity *= 1.0 + 0.0000001*star.Mass

	case "Red Dwarf":
		// Extremely stable — almost no change
		// But occasional flares
		if star.Flares && rand.Intn(5000) == 0 {
			ses.stellarFlare(tick, star, sys, game)
		}

	case "Red Giant":
		// Luminosity fluctuations + mass loss
		star.Luminosity *= 1.0 + (rand.Float64()-0.5)*0.00001
		star.Mass *= 1.0 - 0.0000001 // slow mass loss

	case "Blue Giant":
		// Fast evolution, frequent flares
		star.Luminosity *= 1.0 + 0.000001
		if star.Flares && rand.Intn(1000) == 0 {
			ses.stellarFlare(tick, star, sys, game)
		}

	case "White Dwarf":
		// Gradual cooling
		if star.Temperature > 4000 {
			star.Temperature -= 1 // very slow cooling
		}
		star.Luminosity *= 0.9999999
	}
}

func (ses *StellarEvolutionSystem) stellarFlare(tick int64, star *entities.Star, sys *entities.System, game GameProvider) {
	// Rate limit flares
	if tick-ses.lastFlare[star.GetID()] < 10000 {
		return
	}
	ses.lastFlare[star.GetID()] = tick

	// Flare intensity scales with star type
	intensity := 0.3
	switch star.StarType {
	case "Red Dwarf":
		intensity = 0.5 // red dwarf flares are proportionally devastating
	case "Blue Giant":
		intensity = 0.8 // massive flares
	}

	// Damage unshielded planets
	damaged := 0
	for _, e := range sys.Entities {
		planet, ok := e.(*entities.Planet)
		if !ok || planet.Mass == 0 || planet.Owner == "" {
			continue
		}

		// Magnetic field + Planetary Shield protect
		protection := planet.MagneticField
		for _, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingPlanetShield && b.IsOperational {
				protection += 0.5
			}
		}

		if protection < intensity {
			// Flare strips some atmosphere
			loss := (intensity - protection) * 0.01
			planet.AtmoPressure -= planet.AtmoPressure * loss
			if planet.AtmoPressure < 0 {
				planet.AtmoPressure = 0
			}
			// Radiation damages population
			popLoss := int64(float64(planet.Population) * loss * 0.1)
			if popLoss > 0 {
				planet.Population -= popLoss
				if planet.Population < 0 {
					planet.Population = 0
				}
			}
			damaged++
		}
	}

	flareType := "moderate"
	if intensity > 0.6 {
		flareType = "massive"
	}

	game.LogEvent("event", "",
		fmt.Sprintf("☀️ STELLAR FLARE from %s in %s! %s flare (intensity %.0f%%). %d planets affected. Magnetic fields and Planetary Shields provide protection!",
			star.Name, sys.Name, flareType, intensity*100, damaged))

	_ = math.Abs // suppress
}
