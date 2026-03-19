package planet

import (
	"math/rand"
	"testing"

	"github.com/hunterjsb/xandaris/entities"
)

// TestAtmosphericPressurePhysics verifies that atmospheric pressure
// follows expected patterns based on mass, gravity, and composition.
func TestAtmosphericPressurePhysics(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	tests := []struct {
		name       string
		mass       float64
		gravity    float64
		comp       entities.Composition
		magField   float64
		starLum    float64
		orbitAU    float64
		expectLow  bool // expect < 0.1 atm
		expectHigh bool // expect > 10 atm
	}{
		{
			name: "Tiny airless body (Mercury-like)",
			mass: 0.005, gravity: 0.05,
			comp:     entities.Composition{Iron: 0.7, Silicate: 0.3},
			magField: 0, starLum: 1.0, orbitAU: 0.4,
			expectLow: true,
		},
		{
			name: "Earth-like",
			mass: 1.0, gravity: 1.0,
			comp:     entities.Composition{Iron: 0.32, Silicate: 0.40, Water: 0.12, Gas: 0.02, Organics: 0.08, RareEarth: 0.06},
			magField: 1.0, starLum: 1.0, orbitAU: 1.0,
		},
		{
			name: "Gas giant",
			mass: 300, gravity: 2.0,
			comp:     entities.Composition{Iron: 0.02, Silicate: 0.05, Water: 0.15, Gas: 0.70, Organics: 0.03, RareEarth: 0.05},
			magField: 2.0, starLum: 1.0, orbitAU: 5.0,
			expectHigh: true,
		},
		{
			name: "No magnetic field, close to star (atmosphere stripped)",
			mass: 0.5, gravity: 0.7,
			comp:     entities.Composition{Iron: 0.30, Silicate: 0.50, Water: 0.05, Gas: 0.02, Organics: 0.08, RareEarth: 0.05},
			magField: 0, starLum: 5.0, orbitAU: 0.3,
			expectLow: true,
		},
	}

	for _, tt := range tests {
		pressure := computeAtmoPressure(tt.mass, tt.gravity, tt.comp, tt.magField, tt.starLum, tt.orbitAU, rng)

		result := "OK"
		if tt.expectLow && pressure > 0.1 {
			result = "FAIL (expected low)"
			t.Errorf("%s: pressure %.4f atm, expected < 0.1", tt.name, pressure)
		}
		if tt.expectHigh && pressure < 10 {
			result = "FAIL (expected high)"
			t.Errorf("%s: pressure %.4f atm, expected > 10", tt.name, pressure)
		}

		t.Logf("  %s: %.4f atm [%s]", tt.name, pressure, result)
	}
}

// TestAlbedoConsistency verifies albedo values are physically reasonable.
func TestAlbedoConsistency(t *testing.T) {
	tests := []struct {
		name    string
		comp    entities.Composition
		tempC   int
		minAlb  float64
		maxAlb  float64
	}{
		{
			name:   "Ice world (high albedo)",
			comp:   entities.Composition{Water: 0.5, Silicate: 0.3, Iron: 0.2},
			tempC:  -100,
			minAlb: 0.4,
			maxAlb: 0.9,
		},
		{
			name:   "Ocean world (low albedo — absorbs light)",
			comp:   entities.Composition{Water: 0.4, Silicate: 0.3, Iron: 0.2, Organics: 0.1},
			tempC:  20,
			minAlb: 0.05,
			maxAlb: 0.3,
		},
		{
			name:   "Gas giant (cloud cover)",
			comp:   entities.Composition{Gas: 0.7, Water: 0.15, Iron: 0.05, Silicate: 0.1},
			tempC:  -150,
			minAlb: 0.3,
			maxAlb: 0.7,
		},
		{
			name:   "Rocky desert",
			comp:   entities.Composition{Silicate: 0.6, Iron: 0.3, Water: 0.02, Organics: 0.08},
			tempC:  80,
			minAlb: 0.15,
			maxAlb: 0.5,
		},
	}

	for _, tt := range tests {
		albedo := computeAlbedo(tt.comp, tt.tempC)
		if albedo < tt.minAlb || albedo > tt.maxAlb {
			t.Errorf("%s: albedo %.3f outside expected range [%.2f, %.2f]", tt.name, albedo, tt.minAlb, tt.maxAlb)
		}
		t.Logf("  %s: albedo=%.3f", tt.name, albedo)
	}
}

// TestTemperatureGreenhouse verifies the greenhouse effect works correctly.
func TestTemperatureGreenhouse(t *testing.T) {
	// Venus-like: high pressure + high CO2 → extreme greenhouse
	venusTemp := computeTemperature(1.0, 0.72, 90.0,
		entities.Composition{Gas: 0.3, Organics: 0.3, Silicate: 0.3, Iron: 0.1}, 0.2)
	venusC := int(venusTemp - 273)
	t.Logf("Venus analog: %.0fK (%d°C) [real Venus: 735K/462°C]", venusTemp, venusC)
	if venusC < 200 {
		t.Error("Venus analog too cold — greenhouse effect too weak")
	}

	// Mars-like: very low pressure → minimal greenhouse
	marsTemp := computeTemperature(1.0, 1.52, 0.006,
		entities.Composition{Silicate: 0.5, Iron: 0.3, Water: 0.05, Organics: 0.05, Gas: 0.1}, 0)
	marsC := int(marsTemp - 273)
	t.Logf("Mars analog: %.0fK (%d°C) [real Mars: 210K/-63°C]", marsTemp, marsC)
	if marsC > 0 {
		t.Error("Mars analog too warm — should be below freezing")
	}

	// Earth-like: moderate pressure + modest greenhouse
	earthTemp := computeTemperature(1.0, 1.0, 1.0,
		entities.Composition{Silicate: 0.4, Iron: 0.3, Water: 0.12, Gas: 0.02, Organics: 0.08, RareEarth: 0.08}, 0.1)
	earthC := int(earthTemp - 273)
	t.Logf("Earth analog: %.0fK (%d°C) [real Earth: 288K/15°C]", earthTemp, earthC)
	if earthC < -20 || earthC > 50 {
		t.Errorf("Earth analog temp %d°C outside reasonable range [-20, 50]", earthC)
	}
}
