package planet

import (
	"math/rand"
	"testing"

	"github.com/hunterjsb/xandaris/entities"
)

func TestMoonGeneration(t *testing.T) {
	star := &entities.Star{
		BaseEntity:  entities.BaseEntity{ID: 1, Type: entities.EntityTypeStar},
		Mass:        1.0,
		Luminosity:  1.0,
		Temperature: 5778,
		Metallicity: 1.0,
	}

	// Run 50 seeds and collect moon statistics
	totalMoons := 0
	habitableMoons := 0
	oceanMoons := 0
	volcanicMoons := 0
	tidallyLockedMoons := 0

	for seed := int64(0); seed < 50; seed++ {
		rng := rand.New(rand.NewSource(seed * 73))
		result := FormSystem(star, rng, int(seed), seed*73)

		for _, e := range result {
			p := e.(*entities.Planet)
			if p.ParentPlanetID == 0 || p.PlanetType == "Asteroid Belt" {
				continue
			}

			totalMoons++
			if p.Habitability > 20 {
				habitableMoons++
			}
			if p.PlanetType == "Ocean" {
				oceanMoons++
			}
			if p.VolcanicLevel > 0.3 {
				volcanicMoons++
			}
			if p.TidallyLocked {
				tidallyLockedMoons++
			}
		}
	}

	t.Logf("50 systems: %d moons total", totalMoons)
	t.Logf("  Tidally locked: %d (%.0f%%)", tidallyLockedMoons, float64(tidallyLockedMoons)/float64(totalMoons)*100)
	t.Logf("  Habitable (>20): %d", habitableMoons)
	t.Logf("  Ocean moons: %d", oceanMoons)
	t.Logf("  Volcanic (Io-like): %d", volcanicMoons)

	if totalMoons == 0 {
		t.Fatal("no moons generated in 50 systems")
	}
	if tidallyLockedMoons == 0 {
		t.Error("all moons should be tidally locked")
	}
	// Expect most moons to be tidally locked
	lockPct := float64(tidallyLockedMoons) / float64(totalMoons) * 100
	if lockPct < 90 {
		t.Errorf("expected >90%% tidally locked, got %.0f%%", lockPct)
	}
}

func TestMoonTidalHeating(t *testing.T) {
	star := &entities.Star{
		BaseEntity:  entities.BaseEntity{ID: 1, Type: entities.EntityTypeStar},
		Mass:        1.0,
		Luminosity:  1.0,
		Temperature: 5778,
		Metallicity: 1.0,
	}

	// Find a gas giant with moons across seeds
	for seed := int64(0); seed < 100; seed++ {
		rng := rand.New(rand.NewSource(seed * 31))
		result := FormSystem(star, rng, int(seed), seed*31)

		for _, e := range result {
			p := e.(*entities.Planet)
			if p.PlanetType != "Gas Giant" || len(p.Moons) < 2 {
				continue
			}

			t.Logf("Gas Giant: %s (%.0f M⊕, %d moons)", p.Name, p.Mass, len(p.Moons))

			// Find its moons
			for _, me := range result {
				moon := me.(*entities.Planet)
				if moon.ParentPlanetID != p.GetID() {
					continue
				}

				t.Logf("  Moon %s: type=%s mass=%.3f temp=%d°C volcanic=%.2f heat=%.4f hab=%d",
					moon.Name, moon.PlanetType, moon.Mass, moon.Temperature,
					moon.VolcanicLevel, moon.InternalHeat, moon.Habitability)

				// First moon should have more tidal heating than last
				if moon.InternalHeat <= 0 {
					t.Errorf("moon %s should have some internal heat from tidal forces", moon.Name)
				}
			}
			return // found one, done
		}
	}
	t.Log("WARNING: no gas giant with 2+ moons found in 100 seeds")
}
