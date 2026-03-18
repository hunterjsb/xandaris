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
	orbits := make([]float64, planetCount)
	orbits[0] = 0.2 + rng.Float64()*0.3 // 0.2-0.5 AU
	for i := 1; i < planetCount; i++ {
		orbits[i] = orbits[i-1] * (1.4 + rng.Float64()*0.6)
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

		// Habitability
		hab := formationHabitability(tempC, atmo, gravity, comp)

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

	return result
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
