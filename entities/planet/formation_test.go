package planet

import (
	"math/rand"
	"testing"

	"github.com/hunterjsb/xandaris/entities"
)

func TestFormationSunlike(t *testing.T) {
	star := &entities.Star{
		BaseEntity: entities.BaseEntity{ID: 1, Name: "Sol", Type: entities.EntityTypeStar},
		StarType:   "Main Sequence",
		Mass:       1.0,
		Luminosity: 1.0,
		Temperature: 5778,
		Metallicity: 1.0,
	}
	rng := rand.New(rand.NewSource(42))
	result := FormSystem(star, rng, 0, 42)

	if len(result) < 3 {
		t.Fatalf("expected at least 3 entities (planets+moons), got %d", len(result))
	}

	planets := 0
	moons := 0
	var gasGiant *entities.Planet

	for _, e := range result {
		p, ok := e.(*entities.Planet)
		if !ok {
			t.Fatal("non-planet entity returned from FormSystem")
		}

		if p.Mass <= 0 {
			t.Errorf("planet %s has zero mass", p.Name)
		}
		if p.Gravity <= 0 {
			t.Errorf("planet %s has zero gravity", p.Name)
		}
		if p.RadiusAU <= 0 {
			t.Errorf("planet %s has zero radius", p.Name)
		}
		if p.Density <= 0 {
			t.Errorf("planet %s has zero density", p.Name)
		}
		if p.AtmoPressure < 0 {
			t.Errorf("planet %s has negative atmo pressure", p.Name)
		}

		// Composition should sum to ~1.0
		comp := p.Comp
		total := comp.Iron + comp.Silicate + comp.Water + comp.Gas + comp.Organics + comp.RareEarth
		if total < 0.95 || total > 1.05 {
			t.Errorf("planet %s composition sums to %.3f (expected ~1.0)", p.Name, total)
		}

		if p.ParentPlanetID > 0 {
			moons++
		} else {
			planets++
			if p.PlanetType == "Gas Giant" {
				gasGiant = p
			}
		}

		t.Logf("  %s: type=%s mass=%.2f gravity=%.2f temp=%d°C atmo=%s pressure=%.3fatm hab=%d mag=%.2f ocean=%.0f%% ice=%.0f%% locked=%v tectonic=%v moons=%d",
			p.Name, p.PlanetType, p.Mass, p.Gravity, p.Temperature,
			p.Atmosphere, p.AtmoPressure, p.Habitability,
			p.MagneticField, p.OceanCoverage*100, p.IceCoverage*100,
			p.TidallyLocked, p.TectonicActive, len(p.Moons))
	}

	if planets < 2 {
		t.Errorf("expected at least 2 planets, got %d", planets)
	}

	t.Logf("Total: %d planets, %d moons", planets, moons)

	// Sun-like star should produce at least one gas giant (beyond frost line)
	if gasGiant == nil {
		t.Log("WARNING: no gas giant generated (possible with this seed)")
	} else {
		if gasGiant.Mass < 10 {
			t.Errorf("gas giant %s has suspiciously low mass: %.2f", gasGiant.Name, gasGiant.Mass)
		}
		if len(gasGiant.Moons) == 0 {
			t.Errorf("gas giant %s should have moons", gasGiant.Name)
		}
	}
}

func TestFormationRedDwarf(t *testing.T) {
	star := &entities.Star{
		BaseEntity: entities.BaseEntity{ID: 2, Name: "Proxima", Type: entities.EntityTypeStar},
		StarType:   "Red Dwarf",
		Mass:       0.12,
		Luminosity: 0.0017,
		Temperature: 3042,
		Metallicity: 0.8,
	}
	rng := rand.New(rand.NewSource(123))
	result := FormSystem(star, rng, 1, 123)

	for _, e := range result {
		p := e.(*entities.Planet)

		// Red dwarf planets should mostly be close-in
		if p.ParentPlanetID == 0 && p.OrbitAU > 5.0 {
			t.Errorf("planet %s at %.2f AU — too far for red dwarf", p.Name, p.OrbitAU)
		}

		// Close-in planets should be tidally locked
		if p.OrbitAU < 0.1 && !p.TidallyLocked && p.ParentPlanetID == 0 {
			t.Errorf("planet %s at %.2f AU should be tidally locked", p.Name, p.OrbitAU)
		}

		t.Logf("  %s: type=%s mass=%.3f orbit=%.3fAU locked=%v mag=%.2f pressure=%.4f hab=%d",
			p.Name, p.PlanetType, p.Mass, p.OrbitAU,
			p.TidallyLocked, p.MagneticField, p.AtmoPressure, p.Habitability)
	}
}

func TestFormationDeterministic(t *testing.T) {
	star := &entities.Star{
		BaseEntity: entities.BaseEntity{ID: 1},
		Mass: 1.0, Luminosity: 1.0, Metallicity: 1.0, Temperature: 5778,
	}

	result1 := FormSystem(star, rand.New(rand.NewSource(999)), 0, 999)
	result2 := FormSystem(star, rand.New(rand.NewSource(999)), 0, 999)

	if len(result1) != len(result2) {
		t.Fatalf("determinism broken: %d vs %d entities", len(result1), len(result2))
	}

	for i := range result1 {
		p1 := result1[i].(*entities.Planet)
		p2 := result2[i].(*entities.Planet)
		if p1.Name != p2.Name || p1.Mass != p2.Mass || p1.Temperature != p2.Temperature {
			t.Errorf("determinism broken at index %d: %s(%.2f,%d) vs %s(%.2f,%d)",
				i, p1.Name, p1.Mass, p1.Temperature, p2.Name, p2.Mass, p2.Temperature)
		}
	}
	t.Log("Determinism verified: same seed → identical planets")
}

func TestHydrosphere(t *testing.T) {
	// Liquid water requires temp 0-100°C and pressure > 0.006 atm
	ocean, ice := computeHydrosphere(0.3, 20, 1.0) // Earth-like
	if ocean < 0.5 {
		t.Errorf("Earth-like conditions should have >50%% ocean, got %.0f%%", ocean*100)
	}

	ocean2, ice2 := computeHydrosphere(0.3, -50, 1.0) // Frozen
	if ocean2 > ice2 {
		t.Errorf("frozen world should have more ice than ocean: ocean=%.2f ice=%.2f", ocean2, ice2)
	}

	ocean3, _ := computeHydrosphere(0.3, 20, 0.001) // Mars-like pressure
	if ocean3 > 0 {
		t.Errorf("below triple point should have no liquid: ocean=%.2f", ocean3)
	}

	t.Logf("Earth-like: ocean=%.0f%% ice=%.0f%% | Frozen: ocean=%.0f%% ice=%.0f%% | Low pressure: ocean=%.0f%%",
		ocean*100, ice*100, ocean2*100, ice2*100, ocean3*100)
}
