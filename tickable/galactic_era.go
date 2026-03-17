package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticEraSystem{
		BaseSystem: NewBaseSystem("GalacticEra", 84),
	})
}

// GalacticEraSystem tracks the overall technological and political
// state of the galaxy and announces era transitions. Eras affect
// the flavor of events and modify some game parameters.
//
// Eras (based on average galactic tech level):
//   Dawn Age (avg tech <0.5): primitive colonies, basic trade
//   Expansion Age (0.5-1.5): colonization boom, trade routes forming
//   Industrial Age (1.5-2.5): factories, refineries, military buildup
//   Golden Age (2.5-3.5): prosperity, mega-structures, diplomacy
//   Transcendence Age (3.5+): ancient relics, alien contact, wonders
//
// Each era transition is a major galaxy-wide event. The era affects:
//   - Which random events can fire
//   - Base production rates
//   - Diplomacy tendency (early = hostile, late = cooperative)
//
// The galaxy can only move forward in eras, never backward.
type GalacticEraSystem struct {
	*BaseSystem
	currentEra string
}

var galacticEras = []struct {
	name      string
	threshold float64
	desc      string
}{
	{"Transcendence Age", 3.5, "The galaxy reaches for the stars beyond stars"},
	{"Golden Age", 2.5, "Prosperity and cooperation define the era"},
	{"Industrial Age", 1.5, "Factories roar and fleets multiply"},
	{"Expansion Age", 0.5, "Colonies spread across the void"},
	{"Dawn Age", 0.0, "Civilization takes its first steps"},
}

func (ges *GalacticEraSystem) OnTick(tick int64) {
	if tick%5000 != 0 {
		return
	}

	ctx := ges.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	systems := game.GetSystems()

	// Calculate average tech level across all owned planets
	totalTech := 0.0
	planetCount := 0
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				totalTech += planet.TechLevel
				planetCount++
			}
		}
	}

	if planetCount == 0 {
		return
	}

	avgTech := totalTech / float64(planetCount)

	// Determine era
	newEra := "Dawn Age"
	for _, era := range galacticEras {
		if avgTech >= era.threshold {
			newEra = era.name
			break
		}
	}

	if ges.currentEra == "" {
		ges.currentEra = newEra
		return
	}

	if newEra != ges.currentEra {
		// Can only advance, not regress
		newIdx := eraIndex(newEra)
		currentIdx := eraIndex(ges.currentEra)
		if newIdx <= currentIdx {
			return // don't go backward
		}

		ges.currentEra = newEra
		desc := ""
		for _, era := range galacticEras {
			if era.name == newEra {
				desc = era.desc
				break
			}
		}

		game.LogEvent("event", "",
			fmt.Sprintf("🌟 A NEW ERA DAWNS: The galaxy enters the %s! %s (avg tech: %.1f)",
				newEra, desc, avgTech))
	}
}

func eraIndex(name string) int {
	for i, era := range galacticEras {
		if era.name == name {
			return len(galacticEras) - 1 - i // reverse so Dawn=0, Transcendence=4
		}
	}
	return 0
}

// GetCurrentEra returns the current galactic era.
func (ges *GalacticEraSystem) GetCurrentEra() string {
	if ges.currentEra == "" {
		return "Dawn Age"
	}
	return ges.currentEra
}
