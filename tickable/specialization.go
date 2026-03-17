package tickable

import (
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SpecializationSystem{
		BaseSystem: NewBaseSystem("Specialization", 9),
	})
}

// SpecializationSystem evolves planet workforce specialties over time.
// A planet with many mines develops Mining expertise (+extraction).
// A planet with factories develops Manufacturing (+electronics output).
// A planet as a trade hub develops Commerce (+credit income).
//
// Specialization grows slowly (0.1% per interval) based on building mix
// and is stored on the planet as specialty bonuses. It takes real time
// to develop a specialized workforce — you can't just build a factory
// and instantly get expert manufacturers.
//
// Specialties:
//   Mining:        +1% extraction per point (from mines)
//   Refining:      +1% refinery output per point (from refineries)
//   Manufacturing: +1% factory output per point (from factories)
//   Commerce:      +1% credit income per point (from Trading Posts)
//   Science:       +0.5% tech growth per point (from high-tech buildings)
type SpecializationSystem struct {
	*BaseSystem
}

func (ss *SpecializationSystem) OnTick(tick int64) {
	// Evolve slowly — every 200 ticks (~20 seconds)
	if tick%200 != 0 {
		return
	}

	ctx := ss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	for _, sys := range game.GetSystems() {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.Population <= 0 {
				continue
			}
			ss.evolveSpecialties(planet)
		}
	}
}

func (ss *SpecializationSystem) evolveSpecialties(planet *entities.Planet) {
	// Initialize specialties map if needed
	if planet.Specialties == nil {
		planet.Specialties = make(map[string]float64)
	}

	// Count building types (only staffed ones contribute)
	mines := 0
	refineries := 0
	factories := 0
	tradingPosts := 0
	highTech := 0 // Fusion Reactor, Factory, Shipyard

	for _, be := range planet.Buildings {
		b, ok := be.(*entities.Building)
		if !ok || !b.IsOperational || b.GetStaffingRatio() <= 0 {
			continue
		}
		switch b.BuildingType {
		case entities.BuildingMine:
			mines++
		case entities.BuildingRefinery:
			refineries++
		case entities.BuildingFactory:
			factories++
		case entities.BuildingTradingPost:
			tradingPosts++
		case entities.BuildingFusionReactor, entities.BuildingShipyard:
			highTech++
		}
	}

	// Each staffed building of a type contributes 0.1 specialty points per interval
	// Specialty decays 0.02 per interval toward 0 (use it or lose it)
	growthRate := 0.1
	decayRate := 0.02
	maxSpecialty := 20.0 // cap at 20% bonus

	evolve := func(key string, count int) {
		current := planet.Specialties[key]
		if count > 0 {
			gain := growthRate * float64(count)
			current += gain
			if current > maxSpecialty {
				current = maxSpecialty
			}
		} else if current > 0 {
			current -= decayRate
			if current < 0 {
				current = 0
			}
		}
		if current > 0 {
			planet.Specialties[key] = current
		} else {
			delete(planet.Specialties, key)
		}
	}

	evolve("mining", mines)
	evolve("refining", refineries)
	evolve("manufacturing", factories)
	evolve("commerce", tradingPosts)
	evolve("science", highTech)
}
