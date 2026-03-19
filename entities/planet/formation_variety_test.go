package planet

import (
	"math/rand"
	"testing"

	"github.com/hunterjsb/xandaris/entities"
)

// TestFormationVariety runs 20 different seeds and checks that we get
// diverse planet types, gas giants, habitable worlds, and moons.
func TestFormationVariety(t *testing.T) {
	star := &entities.Star{
		BaseEntity:  entities.BaseEntity{ID: 1, Name: "Sol", Type: entities.EntityTypeStar},
		StarType:    "Main Sequence",
		Mass:        1.0,
		Luminosity:  1.0,
		Temperature: 5778,
		Metallicity: 1.0,
	}

	totalPlanets := 0
	totalMoons := 0
	totalBelts := 0
	gasGiants := 0
	terrestrials := 0
	oceans := 0
	deserts := 0
	ices := 0
	lavas := 0
	barrens := 0
	habitable := 0 // habitability > 30

	for seed := int64(0); seed < 20; seed++ {
		rng := rand.New(rand.NewSource(seed * 137))
		result := FormSystem(star, rng, int(seed), seed*137)

		for _, e := range result {
			p := e.(*entities.Planet)
			if p.PlanetType == "Asteroid Belt" {
				totalBelts++
				continue
			}
			if p.ParentPlanetID > 0 {
				totalMoons++
			} else {
				totalPlanets++
			}

			switch p.PlanetType {
			case "Gas Giant":
				gasGiants++
			case "Terrestrial":
				terrestrials++
			case "Ocean":
				oceans++
			case "Desert":
				deserts++
			case "Ice":
				ices++
			case "Lava":
				lavas++
			case "Barren":
				barrens++
			}

			if p.Habitability > 30 {
				habitable++
			}
		}
	}

	t.Logf("20 Sun-like systems:")
	t.Logf("  Planets: %d | Moons: %d | Belts: %d", totalPlanets, totalMoons, totalBelts)
	t.Logf("  Types: %d Gas Giant, %d Terrestrial, %d Ocean, %d Desert, %d Ice, %d Lava, %d Barren",
		gasGiants, terrestrials, oceans, deserts, ices, lavas, barrens)
	t.Logf("  Habitable (>30): %d", habitable)

	// We should see variety across 20 systems
	if gasGiants == 0 {
		t.Error("PROBLEM: zero gas giants in 20 Sun-like systems — formation is broken")
	}
	if habitable == 0 {
		t.Error("PROBLEM: zero habitable worlds in 20 systems — habitability calc too harsh")
	}
	if terrestrials+oceans == 0 {
		t.Error("PROBLEM: no terrestrial or ocean worlds — inner system too hot?")
	}
	if totalMoons == 0 {
		t.Error("PROBLEM: no moons generated")
	}
}

func TestFormationBlueGiant(t *testing.T) {
	star := &entities.Star{
		BaseEntity:  entities.BaseEntity{ID: 1, Name: "Rigel", Type: entities.EntityTypeStar},
		StarType:    "Blue Giant",
		Mass:        20.0,
		Luminosity:  50000.0,
		Temperature: 30000,
		Metallicity: 0.8,
		Age:         0.01,
	}
	rng := rand.New(rand.NewSource(55))
	result := FormSystem(star, rng, 0, 55)

	t.Logf("Blue Giant system (%d entities):", len(result))
	for _, e := range result {
		p := e.(*entities.Planet)
		if p.PlanetType == "Asteroid Belt" {
			t.Logf("  %s at %.1f AU", p.PlanetType, p.OrbitAU)
			continue
		}
		t.Logf("  %s: type=%s mass=%.1f orbit=%.1f AU temp=%d°C hab=%d moons=%d",
			p.Name, p.PlanetType, p.Mass, p.OrbitAU, p.Temperature, p.Habitability, len(p.Moons))
	}

	// Blue giant: massive disk, frost line very far out, should have huge planets
	// Luminosity 50000 → frost line at 2.7 * sqrt(50000) ≈ 600 AU!
	// All planets should be inside frost line (rocky/hot)
}

func TestFormationWhiteDwarf(t *testing.T) {
	star := &entities.Star{
		BaseEntity:  entities.BaseEntity{ID: 1, Name: "Sirius B", Type: entities.EntityTypeStar},
		StarType:    "White Dwarf",
		Mass:        0.8,
		Luminosity:  0.005,
		Temperature: 25000,
		Metallicity: 0.5,
		Age:         8.0,
	}
	rng := rand.New(rand.NewSource(77))
	result := FormSystem(star, rng, 0, 77)

	t.Logf("White Dwarf system (%d entities):", len(result))
	for _, e := range result {
		p := e.(*entities.Planet)
		if p.PlanetType == "Asteroid Belt" {
			t.Logf("  %s at %.2f AU", p.PlanetType, p.OrbitAU)
			continue
		}
		t.Logf("  %s: type=%s mass=%.2f orbit=%.2f AU temp=%d°C hab=%d",
			p.Name, p.PlanetType, p.Mass, p.OrbitAU, p.Temperature, p.Habitability)
	}

	// White dwarf: very low luminosity, planets should be cold
	// Frost line at 2.7 * sqrt(0.005) ≈ 0.19 AU — almost everything is ice/gas
}

func TestFormationBinary(t *testing.T) {
	star := &entities.Star{
		BaseEntity:  entities.BaseEntity{ID: 1, Name: "Alpha Cen A", Type: entities.EntityTypeStar},
		StarType:    "Main Sequence",
		Mass:        1.1,
		Luminosity:  1.5,
		Temperature: 5790,
		Metallicity: 1.2,
		IsBinary:    true,
	}
	rng := rand.New(rand.NewSource(99))
	result := FormSystem(star, rng, 0, 99)

	t.Logf("Binary system (%d entities):", len(result))
	planets := 0
	for _, e := range result {
		p := e.(*entities.Planet)
		if p.PlanetType == "Asteroid Belt" || p.ParentPlanetID > 0 {
			continue
		}
		planets++
		t.Logf("  %s: type=%s mass=%.2f orbit=%.2f AU", p.Name, p.PlanetType, p.Mass, p.OrbitAU)
	}

	// Binary systems should have fewer planets (disk disruption)
	if planets > 6 {
		t.Errorf("binary system has %d planets — expected fewer due to disk disruption", planets)
	}
	t.Logf("Binary produced %d planets (expected fewer than single star)", planets)
}
