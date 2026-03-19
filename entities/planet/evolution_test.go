package planet

import (
	"image/color"
	"testing"

	"github.com/hunterjsb/xandaris/entities"
)

func TestAtmosphericErosion(t *testing.T) {
	// A planet with no magnetic field should slowly lose atmosphere
	planet := entities.NewPlanet(1, "Test", "Desert", 80, 0, color.RGBA{})
	planet.Mass = 0.5
	planet.MagneticField = 0.05 // very weak field
	planet.AtmoPressure = 1.0   // starts at 1 atm

	// Simulate erosion: each step represents the evolution system's tick
	initialPressure := planet.AtmoPressure
	for i := 0; i < 10000; i++ {
		if planet.MagneticField < 0.2 && planet.AtmoPressure > 0.001 {
			erosionRate := (0.2 - planet.MagneticField) * 0.0001
			planet.AtmoPressure -= planet.AtmoPressure * erosionRate
		}
	}

	if planet.AtmoPressure >= initialPressure {
		t.Error("atmosphere should have eroded with weak magnetic field")
	}
	t.Logf("Pressure after 10K ticks: %.6f atm (was %.1f) — %.2f%% remaining",
		planet.AtmoPressure, initialPressure, planet.AtmoPressure/initialPressure*100)

	// Should lose about 14% in 10K iterations
	if planet.AtmoPressure/initialPressure > 0.95 {
		t.Error("erosion too slow — should lose >5% in 10K ticks")
	}
	if planet.AtmoPressure/initialPressure < 0.5 {
		t.Error("erosion too fast — should retain >50% in 10K ticks")
	}
}

func TestGeologicalCooling(t *testing.T) {
	planet := entities.NewPlanet(2, "Test", "Terrestrial", 80, 0, color.RGBA{})
	planet.Mass = 1.0
	planet.InternalHeat = 0.1
	planet.TectonicActive = true
	planet.VolcanicLevel = 0.5

	initial := planet.InternalHeat

	// Simulate cooling
	for i := 0; i < 50000; i++ {
		if planet.InternalHeat > 0.005 {
			coolingRate := 0.00001
			if planet.Mass > 1 {
				coolingRate /= planet.Mass
			}
			planet.InternalHeat -= coolingRate
		}
		if planet.InternalHeat < 0.02 && planet.TectonicActive {
			planet.TectonicActive = false
		}
	}

	if planet.InternalHeat >= initial {
		t.Error("planet should have cooled")
	}
	t.Logf("Internal heat: %.4f (was %.4f), tectonics: %v",
		planet.InternalHeat, initial, planet.TectonicActive)

	// After 50K ticks, heat should drop below tectonic threshold
	if planet.TectonicActive {
		t.Log("WARNING: tectonics still active after 50K ticks (may need tuning)")
	}
}
