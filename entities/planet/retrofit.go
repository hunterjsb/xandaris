package planet

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

// RetrofitPhysics adds formation-sim physics to a legacy planet
// (one with Mass=0) WITHOUT changing its game state (buildings,
// resources, population, owner). Only adds the physical properties
// that were missing.
//
// This lets existing live games benefit from the deep simulation
// without requiring a galaxy reset.
func RetrofitPhysics(planet *entities.Planet, star *entities.Star, systemID int) {
	if planet.Mass > 0 {
		return // already has physics
	}
	if star == nil {
		return
	}

	// Use planet ID as deterministic seed
	rng := rand.New(rand.NewSource(int64(planet.GetID())))

	// Estimate orbit AU from pixel orbit distance
	orbitAU := (planet.OrbitDistance - 30.0) / 60.0
	if orbitAU < 0.1 {
		orbitAU = 0.1 + rng.Float64()*0.3
	}

	frostLine := 2.7 * math.Sqrt(star.Luminosity)
	insideFrost := orbitAU < frostLine

	// Composition from planet type + zone
	comp := inferComposition(planet.PlanetType, insideFrost, rng)

	// Mass from planet type
	mass := inferMass(planet.PlanetType, rng)

	// Radius
	radiusE := computeRadius(mass, planet.PlanetType)

	// Gravity
	gravity := mass / (radiusE * radiusE)

	// Density
	density := mass / (radiusE * radiusE * radiusE) * 5.51

	// Core + magnetic field
	coreIronFrac := computeCoreFraction(mass, comp, rng)
	internalHeat := computeInternalHeat(mass, density, rng)
	tidallyLocked := orbitAU < 0.3*math.Pow(star.Mass, 0.5)
	magneticField := computeMagneticField(coreIronFrac, internalHeat, mass, tidallyLocked)
	dayLength := computeDayLength(mass, orbitAU, tidallyLocked, rng)

	// Atmospheric pressure from existing atmosphere type
	atmoPressure := inferAtmoPressure(planet.Atmosphere, mass, gravity)

	// Hydrosphere
	oceanCov, iceCov := computeHydrosphere(comp.Water, planet.Temperature, atmoPressure)

	// Tectonics + volcanism
	tectonicActive := internalHeat > 0.04 && comp.Silicate > 0.2 && mass > 0.3 && mass < 10
	volcanicLevel := computeVolcanism(internalHeat, tectonicActive, mass, rng)

	// Albedo
	albedo := computeAlbedo(comp, planet.Temperature)

	// Axial tilt
	axialTilt := rng.Float64() * 45
	if comp.Gas > 0.4 {
		axialTilt = rng.Float64() * 10
	}

	// Set all physics properties
	planet.Mass = mass
	planet.RadiusAU = radiusE
	planet.Gravity = gravity
	planet.Density = density
	planet.Comp = comp
	planet.OrbitAU = orbitAU
	planet.MagneticField = magneticField
	planet.CoreIronFrac = coreIronFrac
	planet.AtmoPressure = atmoPressure
	planet.OceanCoverage = oceanCov
	planet.IceCoverage = iceCov
	planet.AxialTilt = axialTilt
	planet.DayLength = dayLength
	planet.TidallyLocked = tidallyLocked
	planet.TectonicActive = tectonicActive
	planet.VolcanicLevel = volcanicLevel
	planet.InternalHeat = internalHeat
	planet.Albedo = albedo

	fmt.Printf("[Retrofit] %s: mass=%.2f gravity=%.2f pressure=%.2f ocean=%.0f%% mag=%.2f\n",
		planet.Name, mass, gravity, atmoPressure, oceanCov*100, magneticField)
}

func inferComposition(planetType string, insideFrost bool, rng *rand.Rand) entities.Composition {
	var c entities.Composition
	switch planetType {
	case "Terrestrial":
		c = entities.Composition{Iron: 0.32, Silicate: 0.40, Water: 0.12, Gas: 0.02, Organics: 0.08, RareEarth: 0.06}
	case "Ocean":
		c = entities.Composition{Iron: 0.15, Silicate: 0.25, Water: 0.40, Gas: 0.05, Organics: 0.10, RareEarth: 0.05}
	case "Desert":
		c = entities.Composition{Iron: 0.30, Silicate: 0.50, Water: 0.03, Gas: 0.02, Organics: 0.10, RareEarth: 0.05}
	case "Gas Giant":
		c = entities.Composition{Iron: 0.02, Silicate: 0.05, Water: 0.15, Gas: 0.70, Organics: 0.03, RareEarth: 0.05}
	case "Ice":
		c = entities.Composition{Iron: 0.10, Silicate: 0.15, Water: 0.50, Gas: 0.15, Organics: 0.05, RareEarth: 0.05}
	case "Lava":
		c = entities.Composition{Iron: 0.40, Silicate: 0.45, Water: 0.01, Gas: 0.01, Organics: 0.02, RareEarth: 0.11}
	default: // Barren
		c = entities.Composition{Iron: 0.25, Silicate: 0.55, Water: 0.02, Gas: 0.01, Organics: 0.02, RareEarth: 0.15}
	}
	// Add small random variation
	c.Iron *= 0.9 + rng.Float64()*0.2
	c.Silicate *= 0.9 + rng.Float64()*0.2
	c.Water *= 0.8 + rng.Float64()*0.4
	c.Gas *= 0.8 + rng.Float64()*0.4
	c.Organics *= 0.8 + rng.Float64()*0.4
	c.RareEarth *= 0.8 + rng.Float64()*0.4
	// Normalize
	total := c.Iron + c.Silicate + c.Water + c.Gas + c.Organics + c.RareEarth
	if total > 0 {
		c.Iron /= total
		c.Silicate /= total
		c.Water /= total
		c.Gas /= total
		c.Organics /= total
		c.RareEarth /= total
	}
	return c
}

func inferMass(planetType string, rng *rand.Rand) float64 {
	switch planetType {
	case "Terrestrial":
		return 0.5 + rng.Float64()*2.0 // 0.5-2.5 M⊕
	case "Ocean":
		return 0.8 + rng.Float64()*3.0 // 0.8-3.8 M⊕
	case "Desert":
		return 0.3 + rng.Float64()*2.0
	case "Gas Giant":
		return 20 + rng.Float64()*300 // 20-320 M⊕
	case "Ice":
		return 0.1 + rng.Float64()*2.0
	case "Lava":
		return 0.5 + rng.Float64()*3.0
	default: // Barren
		return 0.05 + rng.Float64()*0.5
	}
}

func inferAtmoPressure(atmo string, mass, gravity float64) float64 {
	switch atmo {
	case entities.AtmosphereBreathable:
		return 0.8 + gravity*0.3 // Earth-like
	case entities.AtmosphereThin:
		return 0.01 + gravity*0.05
	case entities.AtmosphereDense:
		return 10 + mass*2
	case entities.AtmosphereToxic:
		return 0.5 + gravity*0.5
	case entities.AtmosphereCorrosive:
		return 50 + mass*5
	default: // None
		return 0
	}
}
