package planet

import (
	"image/color"
	"testing"

	"github.com/hunterjsb/xandaris/entities"
)

func TestRetrofitLegacyPlanet(t *testing.T) {
	star := &entities.Star{
		BaseEntity:  entities.BaseEntity{ID: 1, Type: entities.EntityTypeStar},
		Mass:        1.0,
		Luminosity:  1.0,
		Temperature: 5778,
		Metallicity: 1.0,
	}

	// Create a legacy planet (as the old generators would)
	planet := entities.NewPlanet(12345, "Legacy World", "Terrestrial",
		80.0, 1.5, color.RGBA{100, 150, 100, 255})
	planet.Temperature = 22
	planet.Atmosphere = entities.AtmosphereBreathable
	planet.Habitability = 75
	planet.Population = 5000
	planet.Owner = "TestPlayer"

	// Verify it's a legacy planet
	if planet.Mass != 0 {
		t.Fatal("expected zero mass for legacy planet")
	}

	// Retrofit
	RetrofitPhysics(planet, star, 0)

	// Verify physics were added
	if planet.Mass <= 0 {
		t.Error("mass should be non-zero after retrofit")
	}
	if planet.Gravity <= 0 {
		t.Error("gravity should be non-zero")
	}
	if planet.AtmoPressure <= 0 {
		t.Error("breathable atmosphere should have pressure > 0")
	}
	if planet.MagneticField <= 0 {
		t.Error("terrestrial planet should have magnetic field")
	}

	// Verify game state wasn't changed
	if planet.Population != 5000 {
		t.Errorf("population changed from 5000 to %d", planet.Population)
	}
	if planet.Owner != "TestPlayer" {
		t.Errorf("owner changed from TestPlayer to %s", planet.Owner)
	}
	if planet.Temperature != 22 {
		t.Errorf("temperature changed from 22 to %d", planet.Temperature)
	}
	if planet.Atmosphere != entities.AtmosphereBreathable {
		t.Errorf("atmosphere changed to %s", planet.Atmosphere)
	}
	if planet.Habitability != 75 {
		t.Errorf("habitability changed from 75 to %d", planet.Habitability)
	}

	t.Logf("Retrofitted: mass=%.2f gravity=%.2f pressure=%.2f magField=%.2f ocean=%.0f%% density=%.1f",
		planet.Mass, planet.Gravity, planet.AtmoPressure, planet.MagneticField,
		planet.OceanCoverage*100, planet.Density)

	// Idempotent: running again should be a no-op
	oldMass := planet.Mass
	RetrofitPhysics(planet, star, 0)
	if planet.Mass != oldMass {
		t.Error("retrofit should be idempotent (skip planets with Mass > 0)")
	}
}
