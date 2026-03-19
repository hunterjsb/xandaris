package planet

import (
	"math"
	"math/rand"
	"testing"

	"github.com/hunterjsb/xandaris/entities"
)

// TestPhysicsChainConsistency verifies the full derivation chain
// produces physically consistent results across many planets.
func TestPhysicsChainConsistency(t *testing.T) {
	star := &entities.Star{
		BaseEntity:  entities.BaseEntity{ID: 1, Type: entities.EntityTypeStar},
		Mass:        1.0,
		Luminosity:  1.0,
		Temperature: 5778,
		Metallicity: 1.0,
	}

	violations := 0
	checked := 0

	for seed := int64(0); seed < 30; seed++ {
		rng := rand.New(rand.NewSource(seed * 41))
		result := FormSystem(star, rng, int(seed), seed*41)

		for _, e := range result {
			p := e.(*entities.Planet)
			if p.PlanetType == "Asteroid Belt" {
				continue
			}
			checked++

			// 1. Mass > 0
			if p.Mass <= 0 {
				t.Errorf("planet %s has non-positive mass %.4f", p.Name, p.Mass)
				violations++
			}

			// 2. Radius > 0 and consistent with mass
			if p.RadiusAU <= 0 {
				t.Errorf("planet %s has non-positive radius %.4f", p.Name, p.RadiusAU)
				violations++
			}

			// 3. Gravity = mass / radius^2 (should be exact)
			expectedGravity := p.Mass / (p.RadiusAU * p.RadiusAU)
			if math.Abs(p.Gravity-expectedGravity) > 0.01 {
				t.Errorf("planet %s gravity %.4f != expected %.4f (mass/r²)", p.Name, p.Gravity, expectedGravity)
				violations++
			}

			// 4. Density = mass / radius^3 * 5.51
			expectedDensity := p.Mass / (p.RadiusAU * p.RadiusAU * p.RadiusAU) * 5.51
			if math.Abs(p.Density-expectedDensity) > 0.1 {
				t.Errorf("planet %s density %.2f != expected %.2f", p.Name, p.Density, expectedDensity)
				violations++
			}

			// 5. Composition sums to ~1.0
			comp := p.Comp
			total := comp.Iron + comp.Silicate + comp.Water + comp.Gas + comp.Organics + comp.RareEarth
			if total < 0.95 || total > 1.05 {
				t.Errorf("planet %s composition sums to %.3f", p.Name, total)
				violations++
			}

			// 6. Gas giants should have high gas fraction
			if p.PlanetType == "Gas Giant" && comp.Gas < 0.3 {
				t.Errorf("gas giant %s has only %.0f%% gas", p.Name, comp.Gas*100)
				violations++
			}

			// 7. Ocean worlds should have water or subsurface ocean
			if p.PlanetType == "Ocean" && p.OceanCoverage < 0.1 && comp.Water < 0.1 {
				t.Errorf("ocean world %s has only %.0f%% ocean, %.0f%% water", p.Name, p.OceanCoverage*100, comp.Water*100)
				violations++
			}

			// 8. Magnetic field requires iron core
			if p.MagneticField > 1.0 && p.CoreIronFrac < 0.05 {
				t.Errorf("planet %s has magnetic field %.2f with tiny core iron %.2f", p.Name, p.MagneticField, p.CoreIronFrac)
				violations++
			}

			// 9. Tidally locked planets should have weak or no magnetic field
			if p.TidallyLocked && p.MagneticField > 1.0 {
				t.Errorf("tidally locked %s has strong magnetic field %.2f", p.Name, p.MagneticField)
				violations++
			}

			// 10. Tectonics require internal heat
			if p.TectonicActive && p.InternalHeat < 0.02 {
				t.Errorf("tectonic %s has low internal heat %.4f", p.Name, p.InternalHeat)
				violations++
			}

			// 11. OrbitAU > 0
			if p.OrbitAU <= 0 && p.ParentPlanetID == 0 {
				t.Errorf("planet %s has non-positive orbit %.4f AU", p.Name, p.OrbitAU)
				violations++
			}

			// 12. Habitability 0-100
			if p.Habitability < 0 || p.Habitability > 100 {
				t.Errorf("planet %s habitability %d out of range", p.Name, p.Habitability)
				violations++
			}

			// 13. Pressure >= 0
			if p.AtmoPressure < 0 {
				t.Errorf("planet %s has negative pressure %.4f", p.Name, p.AtmoPressure)
				violations++
			}

			// 14. Moons should reference valid parent
			if p.ParentPlanetID > 0 {
				found := false
				for _, e2 := range result {
					if e2.GetID() == p.ParentPlanetID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("moon %s references parent %d which doesn't exist", p.Name, p.ParentPlanetID)
					violations++
				}
			}
		}
	}

	t.Logf("Checked %d planets/moons across 30 systems: %d violations", checked, violations)
	if violations > 0 {
		t.Errorf("%d physics consistency violations found", violations)
	}
}

// TestFrostLineEffect verifies that planets inside the frost line
// are rocky and planets outside are icy/gaseous.
func TestFrostLineEffect(t *testing.T) {
	star := &entities.Star{
		BaseEntity:  entities.BaseEntity{ID: 1, Type: entities.EntityTypeStar},
		Mass:        1.0,
		Luminosity:  1.0,
		Temperature: 5778,
		Metallicity: 1.0,
	}

	frostLine := 2.7 // AU for Sun-like

	innerRocky := 0
	innerGas := 0
	outerRocky := 0
	outerGas := 0

	for seed := int64(0); seed < 30; seed++ {
		rng := rand.New(rand.NewSource(seed * 53))
		result := FormSystem(star, rng, int(seed), seed*53)

		for _, e := range result {
			p := e.(*entities.Planet)
			if p.ParentPlanetID > 0 || p.PlanetType == "Asteroid Belt" {
				continue
			}

			isRocky := p.Comp.Silicate+p.Comp.Iron > 0.5
			isGasIce := p.Comp.Gas+p.Comp.Water > 0.5

			if p.OrbitAU < frostLine {
				if isRocky {
					innerRocky++
				}
				if isGasIce {
					innerGas++
				}
			} else {
				if isRocky {
					outerRocky++
				}
				if isGasIce {
					outerGas++
				}
			}
		}
	}

	t.Logf("Frost line effect (30 systems):")
	t.Logf("  Inside frost line: %d rocky, %d gas/ice", innerRocky, innerGas)
	t.Logf("  Outside frost line: %d rocky, %d gas/ice", outerRocky, outerGas)

	// Inner system should be predominantly rocky
	if innerGas > innerRocky {
		t.Errorf("more gas/ice inside frost line (%d) than rocky (%d) — frost line broken", innerGas, innerRocky)
	}
	// Outer system should have gas/ice
	if outerGas == 0 {
		t.Error("no gas/ice worlds outside frost line")
	}
}
