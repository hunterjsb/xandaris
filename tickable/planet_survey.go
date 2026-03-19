package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&PlanetSurveySystem{
		BaseSystem: NewBaseSystem("PlanetSurvey", 170),
	})
}

// PlanetSurveySystem generates visible survey reports about planet
// physics that help players understand and value their worlds.
// Makes the deep simulation DATA visible in the event feed.
//
// Periodically highlights interesting physical properties:
//   "🔬 Survey of Coral 95: Ocean world, 1.77g, 37% ocean, magnetic
//    field active — excellent candidate for colonization!"
//   "⚠️ Inferno 29: Lava world, no magnetic field — atmosphere
//    being stripped by solar wind"
//
// One survey per ~5000 ticks. Rotates through notable planets.
type PlanetSurveySystem struct {
	*BaseSystem
	surveyed  map[int]bool
	nextSurvey int64
}

func (pss *PlanetSurveySystem) OnTick(tick int64) {
	ctx := pss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if pss.surveyed == nil {
		pss.surveyed = make(map[int]bool)
	}

	if pss.nextSurvey == 0 {
		pss.nextSurvey = tick + 3000 + int64(rand.Intn(3000))
	}
	if tick < pss.nextSurvey {
		return
	}
	pss.nextSurvey = tick + 5000 + int64(rand.Intn(3000))

	systems := game.GetSystems()

	// Find an interesting unsurveyed planet
	var candidates []*entities.Planet
	var candidateSystems []string

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Mass == 0 || pss.surveyed[planet.GetID()] {
				continue
			}
			if planet.PlanetType == "Asteroid Belt" {
				continue
			}
			// Interesting = has notable physics
			interesting := planet.OceanCoverage > 0.2 || planet.MagneticField > 0.5 ||
				planet.Gravity > 2.0 || planet.VolcanicLevel > 0.3 ||
				planet.TidallyLocked || planet.TectonicActive ||
				planet.Habitability > 40 || len(planet.Moons) > 0 ||
				planet.AtmoPressure > 10 || planet.Comp.RareEarth > 0.08

			if interesting {
				candidates = append(candidates, planet)
				candidateSystems = append(candidateSystems, sys.Name)
			}
		}
	}

	if len(candidates) == 0 {
		// Reset surveyed list to cycle again
		pss.surveyed = make(map[int]bool)
		return
	}

	idx := rand.Intn(len(candidates))
	planet := candidates[idx]
	sysName := candidateSystems[idx]
	pss.surveyed[planet.GetID()] = true

	// Build survey report
	msg := fmt.Sprintf("🔬 PLANETARY SURVEY — %s (%s, %s):\n", planet.Name, planet.PlanetType, sysName)

	// Physical properties
	props := []string{
		fmt.Sprintf("%.1fM⊕, %.1fg, %.1f g/cm³", planet.Mass, planet.Gravity, planet.Density),
	}

	if planet.OrbitAU > 0 {
		props = append(props, fmt.Sprintf("%.2f AU", planet.OrbitAU))
	}

	// Notable features
	if planet.OceanCoverage > 0.1 {
		props = append(props, fmt.Sprintf("%.0f%% ocean", planet.OceanCoverage*100))
	}
	if planet.IceCoverage > 0.1 {
		props = append(props, fmt.Sprintf("%.0f%% ice caps", planet.IceCoverage*100))
	}
	if planet.MagneticField > 0.3 {
		props = append(props, fmt.Sprintf("magnetic field %.0f%%", planet.MagneticField*100))
	} else if planet.MagneticField < 0.1 && planet.AtmoPressure > 0.1 {
		props = append(props, "⚠ weak magnetic field — atmosphere eroding")
	}
	if planet.AtmoPressure > 0.5 {
		props = append(props, fmt.Sprintf("%.1f atm", planet.AtmoPressure))
	}
	if planet.TidallyLocked {
		props = append(props, "tidally locked")
	}
	if planet.TectonicActive {
		props = append(props, "active tectonics")
	}
	if planet.VolcanicLevel > 0.3 {
		props = append(props, fmt.Sprintf("%.0f%% volcanic", planet.VolcanicLevel*100))
	}
	if len(planet.Moons) > 0 {
		props = append(props, fmt.Sprintf("%d moon(s)", len(planet.Moons)))
	}
	if planet.ParentPlanetID > 0 {
		props = append(props, "moon")
	}

	// Composition highlights
	maxComp := ""
	maxVal := 0.0
	compMap := map[string]float64{
		"iron": planet.Comp.Iron, "silicate": planet.Comp.Silicate,
		"water": planet.Comp.Water, "gas": planet.Comp.Gas,
		"organics": planet.Comp.Organics, "rare earth": planet.Comp.RareEarth,
	}
	for name, val := range compMap {
		if val > maxVal {
			maxVal = val
			maxComp = name
		}
	}
	if maxComp != "" {
		props = append(props, fmt.Sprintf("%.0f%% %s", maxVal*100, maxComp))
	}

	// Build final message
	for i, p := range props {
		if i > 0 {
			msg += " | "
		}
		msg += p
	}

	// Assessment
	if planet.Habitability > 50 {
		msg += " — PRIME COLONIZATION TARGET"
	} else if planet.Habitability > 30 {
		msg += " — viable for colonization"
	} else if planet.Comp.RareEarth > 0.08 {
		msg += " — valuable mining prospect"
	}

	game.LogEvent("explore", planet.Owner, msg)
}
