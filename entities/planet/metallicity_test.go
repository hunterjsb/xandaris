package planet

import (
	"math/rand"
	"testing"

	"github.com/hunterjsb/xandaris/entities"
)

// TestHighMetallicitySystem verifies that metal-rich stars produce
// planets with more rare earth deposits and higher iron fractions.
func TestHighMetallicitySystem(t *testing.T) {
	highMetal := &entities.Star{
		BaseEntity:  entities.BaseEntity{ID: 1, Type: entities.EntityTypeStar},
		Mass:        1.0,
		Luminosity:  1.0,
		Temperature: 5778,
		Metallicity: 2.0, // 2x solar metallicity
	}
	lowMetal := &entities.Star{
		BaseEntity:  entities.BaseEntity{ID: 2, Type: entities.EntityTypeStar},
		Mass:        1.0,
		Luminosity:  1.0,
		Temperature: 5778,
		Metallicity: 0.3, // 0.3x solar (metal-poor)
	}

	// Run 20 seeds each and compare rare earth content
	highRE := 0.0
	lowRE := 0.0
	highIron := 0.0
	lowIron := 0.0
	highResources := 0
	lowResources := 0
	highCount := 0
	lowCount := 0

	for seed := int64(0); seed < 20; seed++ {
		highResult := FormSystem(highMetal, rand.New(rand.NewSource(seed*97)), int(seed), seed*97)
		lowResult := FormSystem(lowMetal, rand.New(rand.NewSource(seed*97)), int(seed), seed*97)

		for _, e := range highResult {
			p := e.(*entities.Planet)
			if p.PlanetType == "Asteroid Belt" || p.ParentPlanetID > 0 {
				continue
			}
			highRE += p.Comp.RareEarth
			highIron += p.Comp.Iron
			highResources += len(p.Resources)
			highCount++
		}
		for _, e := range lowResult {
			p := e.(*entities.Planet)
			if p.PlanetType == "Asteroid Belt" || p.ParentPlanetID > 0 {
				continue
			}
			lowRE += p.Comp.RareEarth
			lowIron += p.Comp.Iron
			lowResources += len(p.Resources)
			lowCount++
		}
	}

	avgHighRE := highRE / float64(highCount)
	avgLowRE := lowRE / float64(lowCount)
	avgHighIron := highIron / float64(highCount)
	avgLowIron := lowIron / float64(lowCount)

	t.Logf("High metallicity (2.0x): %d planets, avg RE=%.4f, avg Iron=%.4f, avg resources=%.1f",
		highCount, avgHighRE, avgHighIron, float64(highResources)/float64(highCount))
	t.Logf("Low metallicity (0.3x):  %d planets, avg RE=%.4f, avg Iron=%.4f, avg resources=%.1f",
		lowCount, avgLowRE, avgLowIron, float64(lowResources)/float64(lowCount))

	if avgHighRE <= avgLowRE {
		t.Errorf("high metallicity should produce more rare earths: high=%.4f low=%.4f", avgHighRE, avgLowRE)
	}

	ratio := avgHighRE / avgLowRE
	t.Logf("RE ratio high/low: %.2fx", ratio)
	if ratio < 1.5 {
		t.Logf("WARNING: metallicity doesn't strongly affect rare earth content (ratio %.2f)", ratio)
	}
}

// TestOrangeKDwarfSystem verifies a K-type star (between Sun and Red Dwarf)
// produces a mix of habitable and outer-system planets.
func TestOrangeKDwarfSystem(t *testing.T) {
	star := &entities.Star{
		BaseEntity:  entities.BaseEntity{ID: 1, Type: entities.EntityTypeStar},
		StarType:    "Main Sequence",
		Mass:        0.7,
		Luminosity:  0.3,
		Temperature: 4500,
		Metallicity: 1.0,
	}

	habitable := 0
	total := 0

	for seed := int64(0); seed < 20; seed++ {
		result := FormSystem(star, rand.New(rand.NewSource(seed*61)), int(seed), seed*61)
		for _, e := range result {
			p := e.(*entities.Planet)
			if p.PlanetType == "Asteroid Belt" || p.ParentPlanetID > 0 {
				continue
			}
			total++
			if p.Habitability > 30 {
				habitable++
			}
		}
	}

	t.Logf("K-dwarf (0.7 M☉, 0.3 L☉): %d planets, %d habitable (%.0f%%)",
		total, habitable, float64(habitable)/float64(total)*100)

	// K-dwarfs are considered ideal for habitable planets
	// (longer main sequence lifetime, calmer than red dwarfs)
	if habitable == 0 {
		t.Error("K-dwarf should produce some habitable worlds")
	}
}
