package planet

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/entities/building"
)

func init() {
	// Register the formation function so the registry can call it
	entities.FormationFunc = FormSystem
}

// FormSystem generates all planets for a star system using a
// physics-based formation simulation. The dependency chain:
//
//   star → disk mass + frost line
//   distance → composition (rock inside frost line, gas/ice outside)
//   mass + composition → planet class
//   mass + class → radius
//   mass + radius → gravity, density
//   star + distance + atmosphere → temperature
//   temperature + atmosphere + gravity + composition → habitability
//   composition → resource deposits
//
// All randomness uses the provided rng for deterministic generation.
func FormSystem(star *entities.Star, rng *rand.Rand, systemID int, seed int64) []entities.Entity {
	// Disk properties from star
	diskMass := star.Mass * star.Metallicity * 0.01 * 3300 // Earth masses
	frostLine := 2.7 * math.Sqrt(star.Luminosity)          // AU

	// Planet count: 2-7, biased by disk mass
	planetCount := 2 + rng.Intn(4)
	if diskMass > 50 {
		planetCount += 1 + rng.Intn(2)
	}
	if planetCount > 7 {
		planetCount = 7
	}

	// Place orbits using Titius-Bode-like geometric progression
	// Ensure at least 1-2 planets beyond the frost line for gas giant formation
	orbits := make([]float64, planetCount)
	orbits[0] = 0.3 + rng.Float64()*0.4 // 0.3-0.7 AU
	for i := 1; i < planetCount; i++ {
		orbits[i] = orbits[i-1] * (1.5 + rng.Float64()*0.8) // wider spacing
	}
	// If all orbits are inside frost line and we have 4+ planets, push last 1-2 out
	if planetCount >= 4 && orbits[planetCount-1] < frostLine*1.2 {
		orbits[planetCount-1] = frostLine * (1.2 + rng.Float64()*1.5)
		if planetCount >= 5 {
			orbits[planetCount-2] = frostLine * (0.8 + rng.Float64()*0.5)
		}
	}

	// Total feeding zone for mass distribution
	totalFeeding := 0.0
	for _, au := range orbits {
		totalFeeding += au * 0.3
	}

	result := make([]entities.Entity, 0, planetCount)

	for i := 0; i < planetCount; i++ {
		au := orbits[i]
		insideFrost := au < frostLine

		// Composition from zone
		comp := computeComposition(insideFrost, star.Metallicity, rng)

		// Mass accretion
		feedingZone := au * 0.3
		massShare := (feedingZone / totalFeeding) * diskMass
		massShare *= 0.7 + rng.Float64()*0.6 // ±30% perturbation
		if !insideFrost {
			massShare *= 3.0 + rng.Float64()*2.0 // ice/gas amplification
		}
		if massShare < 0.01 {
			massShare = 0.01
		}
		if massShare > 3000 {
			massShare = 3000
		}

		// Temperature (equilibrium + greenhouse)
		tempK := 278.0 * math.Pow(star.Luminosity, 0.25) / math.Sqrt(au)

		// Atmosphere (needed for greenhouse adjustment)
		atmo := determineAtmosphere(massShare, comp, tempK)

		// Greenhouse adjustment
		switch atmo {
		case entities.AtmosphereBreathable:
			tempK += 33
		case entities.AtmosphereDense:
			tempK += 200
		case entities.AtmosphereCorrosive:
			tempK += 300
		case entities.AtmosphereNone:
			tempK -= 20
		}
		tempC := int(tempK - 273)

		// Planet class from composition + mass + temperature
		planetType := classifyPlanet(massShare, comp, tempC)

		// Radius from mass + class
		radiusE := computeRadius(massShare, planetType)

		// Gravity = mass / radius^2 (in Earth units)
		gravity := massShare / (radiusE * radiusE)

		// Density = mass / radius^3 * 5.51
		density := massShare / (radiusE * radiusE * radiusE) * 5.51

		// === Deep simulation ===

		// Core differentiation: iron sinks to core based on mass + heat
		coreIronFrac := computeCoreFraction(massShare, comp, rng)

		// Internal heat: radiogenic + gravitational (scales with mass)
		internalHeat := computeInternalHeat(massShare, density, rng)

		// Magnetic field: requires molten iron core + rotation
		// Tidal locking kills the dynamo (no differential rotation)
		tidallyLocked := au < 0.3*math.Pow(star.Mass, 0.5) // close orbits lock
		dayLength := computeDayLength(massShare, au, tidallyLocked, rng)
		magneticField := computeMagneticField(coreIronFrac, internalHeat, massShare, tidallyLocked)

		// Axial tilt (random, but gas giants tend toward low tilt)
		axialTilt := rng.Float64() * 45
		if comp.Gas > 0.4 {
			axialTilt = rng.Float64() * 10
		}

		// Tectonics: requires internal heat + silicate mantle + mass
		tectonicActive := internalHeat > 0.04 && comp.Silicate > 0.2 && massShare > 0.3 && massShare < 10
		volcanicLevel := computeVolcanism(internalHeat, tectonicActive, massShare, rng)

		// Atmospheric pressure: from mass, gravity, composition, magnetic field
		atmoPressure := computeAtmoPressure(massShare, gravity, comp, magneticField, star.Luminosity, au, rng)

		// Recalculate temperature with proper greenhouse (using pressure)
		tempK = computeTemperature(star.Luminosity, au, atmoPressure, comp, volcanicLevel)
		tempC = int(tempK - 273)

		// Redetermine atmosphere type from pressure
		atmo = atmosphereFromPressure(atmoPressure, comp, tempK)

		// Albedo from surface composition + ice/ocean
		albedo := computeAlbedo(comp, tempC)

		// Hydrosphere: water state depends on temp + pressure
		oceanCoverage, iceCoverage := computeHydrosphere(comp.Water, tempC, atmoPressure)

		// Re-classify planet with full physics
		planetType = classifyPlanetDeep(massShare, comp, tempC, oceanCoverage, atmoPressure)

		// Full habitability with all factors
		hab := deepHabitability(tempC, atmo, gravity, comp, magneticField,
			atmoPressure, oceanCoverage, tidallyLocked, tectonicActive)

		// Visual properties
		pixelSize := radiusToPixelSize(radiusE, planetType)
		orbitPx := auToPixels(au)
		orbitAngle := rng.Float64() * 2 * math.Pi
		hasRings := shouldHaveRings(planetType, massShare, rng)
		pColor := planetColor(planetType, comp, rng)
		name := planetName(planetType, rng)

		// Create planet entity
		id := systemID*1000 + 100 + i*100 + rng.Intn(99)
		planet := entities.NewPlanet(id, name, planetType, orbitPx, orbitAngle, pColor)

		// Set physics properties
		planet.Size = pixelSize
		planet.Temperature = tempC
		planet.Atmosphere = atmo
		planet.Habitability = hab
		planet.HasRings = hasRings
		planet.Mass = massShare
		planet.RadiusAU = radiusE
		planet.Gravity = gravity
		planet.Density = density
		planet.Comp = comp
		planet.OrbitAU = au

		// Deep sim properties
		planet.MagneticField = magneticField
		planet.CoreIronFrac = coreIronFrac
		planet.AtmoPressure = atmoPressure
		planet.OceanCoverage = oceanCoverage
		planet.IceCoverage = iceCoverage
		planet.AxialTilt = axialTilt
		planet.DayLength = dayLength
		planet.TidallyLocked = tidallyLocked
		planet.TectonicActive = tectonicActive
		planet.VolcanicLevel = volcanicLevel
		planet.InternalHeat = internalHeat
		planet.Albedo = albedo

		// Generate resources from composition
		generateResourcesFromComposition(planet, comp, rng)

		// Ensure base building
		building.EnsurePlanetHasBase(planet, entities.GenerationParams{
			SystemID:     systemID,
			OrbitDistance: orbitPx,
			OrbitAngle:   orbitAngle,
			SystemSeed:   seed,
		})

		result = append(result, planet)
	}

	// === Moon generation ===
	// Gas giants and large planets spawn moons
	var moons []entities.Entity
	for _, e := range result {
		parent, ok := e.(*entities.Planet)
		if !ok || parent.Mass < 5.0 {
			continue // only massive planets get moons
		}

		moonCount := 1 + rng.Intn(3)
		if parent.PlanetType == "Gas Giant" {
			moonCount = 2 + rng.Intn(4) // 2-5 moons for gas giants
		}

		for m := 0; m < moonCount; m++ {
			moon := generateMoon(parent, m, systemID, star, rng, seed)
			parent.Moons = append(parent.Moons, moon.GetID())
			moons = append(moons, moon)
		}
	}
	result = append(result, moons...)

	return result
}

func generateMoon(parent *entities.Planet, index, systemID int, star *entities.Star, rng *rand.Rand, seed int64) *entities.Planet {
	// Moon mass: 0.001-0.05 of parent mass
	moonMass := parent.Mass * (0.001 + rng.Float64()*0.05)
	if moonMass < 0.001 {
		moonMass = 0.001
	}

	// Composition: inherit from parent's zone but more rocky
	comp := entities.Composition{
		Iron:      parent.Comp.Iron * (1.0 + rng.Float64()*0.5),
		Silicate:  parent.Comp.Silicate * (1.0 + rng.Float64()*0.5),
		Water:     parent.Comp.Water * (0.5 + rng.Float64()),
		Gas:       0.01 * rng.Float64(), // moons lose gas
		Organics:  parent.Comp.Organics * (0.5 + rng.Float64()),
		RareEarth: parent.Comp.RareEarth * (1.0 + rng.Float64()),
	}
	// Normalize
	total := comp.Iron + comp.Silicate + comp.Water + comp.Gas + comp.Organics + comp.RareEarth
	if total > 0 {
		comp.Iron /= total
		comp.Silicate /= total
		comp.Water /= total
		comp.Gas /= total
		comp.Organics /= total
		comp.RareEarth /= total
	}

	radiusE := computeRadius(moonMass, "")
	gravity := moonMass / (radiusE * radiusE)
	density := moonMass / (radiusE * radiusE * radiusE) * 5.51

	// Tidal heating from parent (closer = more heating)
	tidalHeat := parent.Mass * 0.001 / float64(index+1)
	internalHeat := 0.01 + tidalHeat
	volcanicLevel := 0.0
	if tidalHeat > 0.05 {
		volcanicLevel = tidalHeat * 2
		if volcanicLevel > 1.0 {
			volcanicLevel = 1.0
		}
	}

	// Moons are tidally locked to parent
	tidallyLocked := true
	coreIronFrac := comp.Iron * (0.5 + rng.Float64()*0.3)
	magneticField := 0.0
	if coreIronFrac > 0.15 && internalHeat > 0.03 {
		magneticField = coreIronFrac * internalHeat * 5
		if magneticField > 0.5 {
			magneticField = 0.5 // moons have weak fields
		}
	}

	// Temperature: parent's orbit AU from star + tidal heating
	tempK := 278.0 * math.Pow(star.Luminosity, 0.25) / math.Sqrt(parent.OrbitAU)
	tempK += tidalHeat * 200 // tidal heating warms the moon
	atmoPressure := moonMass * gravity * 0.1 * magneticField
	if atmoPressure > 2.0 {
		atmoPressure = 2.0
	}

	atmo := atmosphereFromPressure(atmoPressure, comp, tempK)
	tempC := int(tempK - 273)

	oceanCov, iceCov := computeHydrosphere(comp.Water, tempC, atmoPressure)

	planetType := "Barren"
	if comp.Water > 0.3 && tempC > -40 && tempC < 50 {
		planetType = "Ocean"
	} else if comp.Water > 0.2 && tempC < -20 {
		planetType = "Ice"
	} else if volcanicLevel > 0.5 {
		planetType = "Lava"
	}

	hab := deepHabitability(tempC, atmo, gravity, comp, magneticField,
		atmoPressure, oceanCov, tidallyLocked, comp.Silicate > 0.2)

	// Visual: moons are small
	pixelSize := 3
	if moonMass > 0.01 {
		pixelSize = 4
	}

	// Orbit the parent planet (visual offset)
	moonOrbitPx := parent.OrbitDistance + 12.0 + float64(index)*8.0
	moonAngle := parent.OrbitAngle + float64(index)*1.2

	moonNames := []string{"Luna", "Europa", "Titan", "Io", "Ganymede",
		"Callisto", "Enceladus", "Triton", "Oberon", "Charon"}
	name := fmt.Sprintf("%s %d", moonNames[rng.Intn(len(moonNames))], rng.Intn(99)+1)

	pColor := planetColor(planetType, comp, rng)
	id := systemID*1000 + parent.GetID()*10 + index + 50

	moon := entities.NewPlanet(id, name, planetType, moonOrbitPx, moonAngle, pColor)
	moon.Size = pixelSize
	moon.Temperature = tempC
	moon.Atmosphere = atmo
	moon.Habitability = hab
	moon.Mass = moonMass
	moon.RadiusAU = radiusE
	moon.Gravity = gravity
	moon.Density = density
	moon.Comp = comp
	moon.OrbitAU = parent.OrbitAU
	moon.MagneticField = magneticField
	moon.CoreIronFrac = coreIronFrac
	moon.AtmoPressure = atmoPressure
	moon.OceanCoverage = oceanCov
	moon.IceCoverage = iceCov
	moon.TidallyLocked = tidallyLocked
	moon.VolcanicLevel = volcanicLevel
	moon.InternalHeat = internalHeat
	moon.DayLength = 0 // tidally locked
	moon.ParentPlanetID = parent.GetID()
	moon.Albedo = computeAlbedo(comp, tempC)

	generateResourcesFromComposition(moon, comp, rng)
	building.EnsurePlanetHasBase(moon, entities.GenerationParams{
		SystemID: systemID, OrbitDistance: moonOrbitPx,
		OrbitAngle: moonAngle, SystemSeed: seed,
	})

	return moon
}

func computeComposition(insideFrost bool, metallicity float64, rng *rand.Rand) entities.Composition {
	var c entities.Composition
	if insideFrost {
		c.Iron = 0.25 + rng.Float64()*0.15
		c.Silicate = 0.45 + rng.Float64()*0.15
		c.Water = 0.03 + rng.Float64()*0.07
		c.Gas = 0.01 + rng.Float64()*0.04
		c.Organics = 0.03 + rng.Float64()*0.07
		c.RareEarth = metallicity * (0.01 + rng.Float64()*0.02)
	} else {
		c.Iron = 0.03 + rng.Float64()*0.07
		c.Silicate = 0.08 + rng.Float64()*0.07
		c.Water = 0.25 + rng.Float64()*0.15
		c.Gas = 0.35 + rng.Float64()*0.20
		c.Organics = 0.03 + rng.Float64()*0.05
		c.RareEarth = metallicity * (0.005 + rng.Float64()*0.01)
	}
	// Normalize to 1.0
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

func classifyPlanet(mass float64, comp entities.Composition, tempC int) string {
	if comp.Gas > 0.45 && mass > 10.0 {
		return "Gas Giant"
	}
	if comp.Water > 0.35 && mass < 15.0 && tempC > -80 && tempC < 100 {
		return "Ocean"
	}
	if comp.Water > 0.25 && tempC < -30 {
		return "Ice"
	}
	if tempC > 500 {
		return "Lava"
	}
	if comp.Silicate > 0.30 {
		if tempC > 60 {
			return "Desert"
		}
		if comp.Water > 0.08 {
			return "Terrestrial"
		}
		return "Barren"
	}
	return "Barren"
}

func computeRadius(mass float64, planetType string) float64 {
	switch planetType {
	case "Gas Giant":
		if mass > 100 {
			return 11.0 * math.Pow(mass/318.0, 0.06) // Jupiter-like degeneracy
		}
		return math.Pow(mass, 0.55) * 0.5
	default:
		r := math.Pow(mass, 0.27)
		if r < 0.3 {
			r = 0.3
		}
		return r
	}
}

func determineAtmosphere(mass float64, comp entities.Composition, tempK float64) string {
	if mass < 0.01 {
		return entities.AtmosphereNone
	}
	gravity := mass / math.Pow(computeRadius(mass, ""), 2)
	if comp.Gas > 0.45 {
		return entities.AtmosphereDense
	}
	if tempK > 1200 {
		return entities.AtmosphereCorrosive
	}
	if comp.Water > 0.15 && gravity > 0.4 && tempK > 200 && tempK < 400 {
		return entities.AtmosphereBreathable
	}
	if gravity > 0.2 && tempK > 150 && tempK < 600 {
		if comp.Organics > 0.08 {
			return entities.AtmosphereToxic
		}
		return entities.AtmosphereThin
	}
	if gravity < 0.1 {
		return entities.AtmosphereNone
	}
	return entities.AtmosphereThin
}

func formationHabitability(tempC int, atmo string, gravity float64, comp entities.Composition) int {
	score := 0

	// Temperature (ideal: -10 to 35°C)
	if tempC >= -10 && tempC <= 35 {
		score += 40
	} else if tempC >= -50 && tempC <= 60 {
		score += 15
	} else {
		score -= 20
	}

	// Atmosphere
	switch atmo {
	case entities.AtmosphereBreathable:
		score += 25
	case entities.AtmosphereThin:
		score += 5
	case entities.AtmosphereDense:
		score -= 10
	case entities.AtmosphereToxic:
		score -= 25
	case entities.AtmosphereCorrosive:
		score -= 40
	default: // None
		score -= 30
	}

	// Gravity (ideal: 0.5-1.5g)
	if gravity >= 0.5 && gravity <= 1.5 {
		score += 15
	} else if gravity >= 0.2 && gravity <= 2.5 {
		score += 5
	} else {
		score -= 10
	}

	// Water presence
	waterBonus := int(comp.Water * 20)
	if waterBonus > 20 {
		waterBonus = 20
	}
	score += waterBonus

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

func radiusToPixelSize(radiusE float64, planetType string) int {
	switch planetType {
	case "Gas Giant":
		s := 8 + int(radiusE*0.3)
		if s < 8 {
			s = 8
		}
		if s > 12 {
			s = 12
		}
		return s
	default:
		s := 3 + int(radiusE*3)
		if s < 4 {
			s = 4
		}
		if s > 8 {
			s = 8
		}
		return s
	}
}

func auToPixels(au float64) float64 {
	return 30.0 + au*60.0 // 60px per AU, 30px minimum
}

func shouldHaveRings(planetType string, mass float64, rng *rand.Rand) bool {
	switch planetType {
	case "Gas Giant":
		return rng.Float64() < 0.5
	case "Ice":
		return rng.Float64() < 0.2
	default:
		return rng.Float64() < 0.08
	}
}

func planetColor(planetType string, comp entities.Composition, rng *rand.Rand) color.RGBA {
	switch planetType {
	case "Terrestrial":
		return color.RGBA{
			R: uint8(50 + rng.Intn(80)),
			G: uint8(100 + int(comp.Water*150)),
			B: uint8(50 + int(comp.Water*100)),
			A: 255,
		}
	case "Ocean":
		return color.RGBA{
			R: uint8(20 + rng.Intn(40)),
			G: uint8(100 + rng.Intn(80)),
			B: uint8(180 + rng.Intn(75)),
			A: 255,
		}
	case "Desert":
		return color.RGBA{
			R: uint8(180 + rng.Intn(60)),
			G: uint8(140 + rng.Intn(60)),
			B: uint8(80 + rng.Intn(40)),
			A: 255,
		}
	case "Gas Giant":
		return color.RGBA{
			R: uint8(150 + rng.Intn(80)),
			G: uint8(120 + rng.Intn(80)),
			B: uint8(80 + rng.Intn(80)),
			A: 255,
		}
	case "Ice":
		return color.RGBA{
			R: uint8(180 + rng.Intn(60)),
			G: uint8(200 + rng.Intn(55)),
			B: uint8(220 + rng.Intn(35)),
			A: 255,
		}
	case "Lava":
		return color.RGBA{
			R: uint8(200 + rng.Intn(55)),
			G: uint8(50 + rng.Intn(80)),
			B: uint8(20 + rng.Intn(40)),
			A: 255,
		}
	default: // Barren
		return color.RGBA{
			R: uint8(120 + rng.Intn(60)),
			G: uint8(110 + rng.Intn(50)),
			B: uint8(100 + rng.Intn(50)),
			A: 255,
		}
	}
}

func planetName(planetType string, rng *rand.Rand) string {
	var names []string
	switch planetType {
	case "Terrestrial":
		names = []string{"Terra", "Gaia", "Eden", "Verde", "Solara", "Arcadia", "Haven"}
	case "Ocean":
		names = []string{"Aqua", "Marina", "Oceanus", "Poseidon", "Nautilus", "Coral", "Atlantis"}
	case "Desert":
		names = []string{"Dune", "Arid", "Sahara", "Gobi", "Mojave", "Sirocco", "Mirage"}
	case "Gas Giant":
		names = []string{"Titan", "Colossus", "Nimbus", "Tempest", "Vortex", "Leviathan", "Magnus"}
	case "Ice":
		names = []string{"Frost", "Glacier", "Tundra", "Boreal", "Hail", "Crystal", "Nil"}
	case "Lava":
		names = []string{"Inferno", "Magma", "Vulcan", "Crucible", "Scoria", "Ember", "Cinder"}
	default:
		names = []string{"Rock", "Null", "Void", "Dust", "Crag", "Shard", "Desolate"}
	}
	return fmt.Sprintf("%s %d", names[rng.Intn(len(names))], rng.Intn(100)+1)
}

func generateResourcesFromComposition(planet *entities.Planet, comp entities.Composition, rng *rand.Rand) {
	type rule struct {
		fraction  float64
		resType   string
		rarity    string
		value     int
		threshold float64
	}
	rules := []rule{
		{comp.Iron, entities.ResIron, "Common", 75, 0.05},
		{comp.Water, entities.ResWater, "Common", 115, 0.05},
		{comp.Organics, entities.ResOil, "Uncommon", 150, 0.04},
		{comp.RareEarth, entities.ResRareMetals, "Rare", 550, 0.01},
		{comp.Gas * 0.3, entities.ResHelium3, "Very Rare", 700, 0.05}, // He-3 from gas envelope
	}

	maxDeposits := 6
	deposits := 0

	for _, r := range rules {
		if r.fraction < r.threshold || deposits >= maxDeposits {
			continue
		}

		count := int(r.fraction * 5)
		if count < 1 {
			count = 1
		}

		for j := 0; j < count && deposits < maxDeposits; j++ {
			abundance := int(r.fraction * 100 * (0.8 + rng.Float64()*0.4))
			if abundance < 5 {
				abundance = 5
			}
			if abundance > 100 {
				abundance = 100
			}

			resColor := color.RGBA{128, 128, 128, 255}
			switch r.resType {
			case entities.ResIron:
				resColor = color.RGBA{150, 150, 150, 255}
			case entities.ResWater:
				resColor = color.RGBA{50, 100, 200, 255}
			case entities.ResOil:
				resColor = color.RGBA{40, 40, 40, 255}
			case entities.ResRareMetals:
				resColor = color.RGBA{200, 150, 50, 255}
			case entities.ResHelium3:
				resColor = color.RGBA{150, 220, 255, 255}
			}

			angle := float64(deposits) / float64(maxDeposits) * 2 * math.Pi
			angle += rng.Float64() * 0.5

			res := &entities.Resource{
				BaseEntity: entities.BaseEntity{
					ID:            planet.GetID()*100 + deposits + 1,
					Name:          fmt.Sprintf("%s Deposit", r.resType),
					Type:          entities.EntityTypeResource,
					SubType:       r.resType,
					Color:         resColor,
					OrbitDistance: 10.0 + float64(deposits)*5.0 + rng.Float64()*5.0,
					OrbitAngle:    angle,
				},
				ResourceType:   r.resType,
				Abundance:      abundance,
				ExtractionRate: 0.4 + rng.Float64()*0.5,
				Value:          r.value + rng.Intn(r.value/2),
				Rarity:         r.rarity,
				Size:           4 + rng.Intn(4),
				Quality:        30 + rng.Intn(70),
			}

			planet.Resources = append(planet.Resources, res)
			deposits++
		}
	}
}

// === Deep simulation functions ===

func computeCoreFraction(mass float64, comp entities.Composition, rng *rand.Rand) float64 {
	// Iron sinks to core if planet is massive enough (gravitational differentiation)
	// Small bodies: iron distributed throughout (low core fraction)
	// Large bodies: iron sinks efficiently (high core fraction)
	if mass < 0.01 {
		return 0.1 + rng.Float64()*0.2
	}
	efficiency := math.Min(1.0, math.Log10(mass*100)/3) // 0.3 at 0.1Me, 1.0 at 10Me
	return comp.Iron * efficiency * (0.7 + rng.Float64()*0.3)
}

func computeInternalHeat(mass, density float64, rng *rand.Rand) float64 {
	// Internal heat from radiogenic decay + gravitational compression
	// Earth ≈ 0.087 W/m², scales with mass and density
	base := mass * 0.02 * density / 5.5
	base *= 0.7 + rng.Float64()*0.6
	if base > 1.0 {
		base = 1.0
	}
	return base
}

func computeMagneticField(coreIronFrac, internalHeat, mass float64, tidallyLocked bool) float64 {
	// Dynamo requires: molten iron core + rotation (not tidally locked)
	// Earth = 1.0 (core iron 0.32, internal heat 0.087)
	if tidallyLocked {
		return coreIronFrac * internalHeat * 0.5 // weak field if locked
	}
	field := coreIronFrac * internalHeat * mass * 8
	if field > 2.0 {
		field = 2.0
	}
	return field
}

func computeDayLength(mass, orbitAU float64, tidallyLocked bool, rng *rand.Rand) float64 {
	if tidallyLocked {
		return 0 // represents infinite day (one side always faces star)
	}
	// Day length: smaller planets spin faster (angular momentum conservation)
	// Earth (1.0Me) = 24h, Jupiter (318Me) = 10h, Mars (0.1Me) = 24.6h
	base := 10.0 + rng.Float64()*30.0
	if mass > 10 {
		base = 8 + rng.Float64()*6 // gas giants spin fast
	}
	return base
}

func computeVolcanism(internalHeat float64, tectonic bool, mass float64, rng *rand.Rand) float64 {
	if internalHeat < 0.02 {
		return 0 // dead world
	}
	v := internalHeat * 3 * (0.7 + rng.Float64()*0.6)
	if tectonic {
		v *= 1.5 // tectonics amplify surface expression
	}
	if v > 1.0 {
		v = 1.0
	}
	return v
}

func computeAtmoPressure(mass, gravity float64, comp entities.Composition, magField, starLum, orbitAU float64, rng *rand.Rand) float64 {
	if mass < 0.01 || gravity < 0.05 {
		return 0 // too small to hold atmosphere
	}

	// Base pressure from outgassing (proportional to mass × volatile content)
	volatiles := comp.Gas + comp.Water*0.3 + comp.Organics*0.2
	pressure := mass * gravity * volatiles * 5

	// Magnetic field protects atmosphere from solar wind stripping
	// No field + close to star = atmosphere stripped
	if magField < 0.2 {
		solarWindStrip := starLum / (orbitAU * orbitAU)
		pressure *= math.Max(0.01, 1.0-solarWindStrip*0.3)
	}

	// Gas giants have massive atmospheres
	if comp.Gas > 0.4 {
		pressure = mass * comp.Gas * 10
		if pressure > 1000 {
			pressure = 1000
		}
	}

	pressure *= 0.7 + rng.Float64()*0.6

	return pressure
}

func computeTemperature(starLum, orbitAU, atmoPressure float64, comp entities.Composition, volcanism float64) float64 {
	// Stefan-Boltzmann equilibrium temperature
	tempK := 278.0 * math.Pow(starLum, 0.25) / math.Sqrt(orbitAU)

	// Greenhouse effect: depends on pressure AND composition
	// Earth: ~33K greenhouse from 1 atm with 0.04% CO2
	// Venus: ~500K greenhouse from 92 atm of 96% CO2
	greenhouseGas := comp.Gas*0.2 + comp.Organics*0.4 + comp.Water*0.05
	greenhouse := math.Log1p(atmoPressure) * greenhouseGas * 50
	if greenhouse > 500 {
		greenhouse = 500
	}
	tempK += greenhouse

	// Volcanic outgassing adds modest heat
	tempK += volcanism * 20

	return tempK
}

func atmosphereFromPressure(pressure float64, comp entities.Composition, tempK float64) string {
	if pressure < 0.001 {
		return entities.AtmosphereNone
	}
	if pressure > 50 {
		return entities.AtmosphereDense
	}
	if tempK > 1200 || (comp.Organics > 0.15 && pressure > 5) {
		return entities.AtmosphereCorrosive
	}
	if comp.Organics > 0.1 && pressure > 0.5 {
		return entities.AtmosphereToxic
	}
	if pressure > 0.3 && pressure < 3.0 && comp.Water > 0.05 {
		return entities.AtmosphereBreathable
	}
	if pressure > 0.003 {
		return entities.AtmosphereThin
	}
	return entities.AtmosphereNone
}

func computeAlbedo(comp entities.Composition, tempC int) float64 {
	// Ice is very reflective, oceans absorb, rock is moderate
	albedo := 0.3 // rocky baseline
	if tempC < -30 && comp.Water > 0.1 {
		albedo = 0.5 + comp.Water*0.3 // ice world
	} else if tempC > 0 && comp.Water > 0.2 {
		albedo = 0.12 // ocean absorbs light
	}
	if comp.Gas > 0.4 {
		albedo = 0.3 + comp.Gas*0.2 // cloud cover
	}
	return math.Min(0.9, albedo)
}

func computeHydrosphere(waterFrac float64, tempC int, pressure float64) (ocean, ice float64) {
	if waterFrac < 0.01 || pressure < 0.006 { // 0.006 atm = Mars, below triple point
		return 0, 0 // no stable liquid water
	}

	if tempC > 100 { // above boiling at 1atm (simplified)
		return 0, 0 // water is vapor
	}

	if tempC < 0 {
		// Below freezing: ice coverage proportional to water fraction
		ice = waterFrac * 2
		if ice > 1.0 {
			ice = 1.0
		}
		// Some liquid possible under ice (subsurface ocean)
		if pressure > 0.5 && tempC > -40 {
			ocean = waterFrac * 0.1
		}
		return
	}

	// Liquid water range (0-100°C at ~1 atm)
	ocean = waterFrac * 3
	if ocean > 0.95 {
		ocean = 0.95
	}

	// Polar ice caps if tilt exists and cold enough
	if tempC < 30 {
		ice = math.Max(0, (30-float64(tempC))/100) * waterFrac
	}

	return
}

func classifyPlanetDeep(mass float64, comp entities.Composition, tempC int, oceanCov, atmoPressure float64) string {
	if comp.Gas > 0.45 && mass > 10.0 {
		return "Gas Giant"
	}
	if oceanCov > 0.5 {
		return "Ocean"
	}
	if comp.Water > 0.25 && tempC < -30 {
		return "Ice"
	}
	if tempC > 500 || (tempC > 300 && atmoPressure > 50) {
		return "Lava"
	}
	if comp.Silicate > 0.30 {
		if tempC > 60 || (atmoPressure < 0.01 && tempC > 20) {
			return "Desert"
		}
		if oceanCov > 0.1 || comp.Water > 0.08 {
			return "Terrestrial"
		}
		return "Barren"
	}
	return "Barren"
}

func deepHabitability(tempC int, atmo string, gravity float64, comp entities.Composition,
	magField, atmoPressure, oceanCov float64, tidallyLocked, tectonic bool) int {

	score := 0

	// Temperature (ideal: -10 to 35°C)
	if tempC >= -10 && tempC <= 35 {
		score += 30
	} else if tempC >= -50 && tempC <= 60 {
		score += 10
	} else {
		score -= 25
	}

	// Atmosphere (type + pressure)
	switch atmo {
	case entities.AtmosphereBreathable:
		score += 20
	case entities.AtmosphereThin:
		score += 3
	default:
		score -= 20
	}
	// Pressure sweet spot: 0.5-2.0 atm
	if atmoPressure >= 0.5 && atmoPressure <= 2.0 {
		score += 10
	} else if atmoPressure > 0 && atmoPressure < 5 {
		score += 3
	}

	// Gravity (ideal: 0.5-1.5g)
	if gravity >= 0.5 && gravity <= 1.5 {
		score += 10
	} else if gravity >= 0.2 && gravity <= 2.5 {
		score += 3
	} else {
		score -= 10
	}

	// Magnetic field (protects from radiation)
	if magField > 0.5 {
		score += 10
	} else if magField > 0.1 {
		score += 5
	} else {
		score -= 5 // radiation exposure
	}

	// Water / ocean
	if oceanCov > 0.3 {
		score += 10
	} else if oceanCov > 0.05 {
		score += 5
	}
	score += int(comp.Water * 10)

	// Tidal locking penalty (habitable twilight band only)
	if tidallyLocked {
		score -= 10
	}

	// Tectonics bonus (carbon cycle, resource renewal)
	if tectonic {
		score += 5
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}
