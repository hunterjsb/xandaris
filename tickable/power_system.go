package tickable

import (
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&PowerSystem{
		BaseSystem: NewBaseSystem("Power", 7),
	})
}

// PowerSystem calculates power generation and consumption per planet each tick.
// Power is instantaneous — generated and consumed, not stored.
//
// Generators: burn fuel/He-3 to produce MW.
// Consumers: buildings + population require MW.
// Deficit: reduces PowerRatio which feeds into happiness → productivity.
type PowerSystem struct {
	*BaseSystem
}

// Power draw per building type (MW)
var buildingPowerDraw = map[string]float64{
	entities.BuildingBase:          10,
	entities.BuildingMine:          15,
	entities.BuildingHabitat:       10,
	entities.BuildingTradingPost:   10,
	entities.BuildingRefinery:      25,
	entities.BuildingFactory:       30,
	entities.BuildingShipyard:      35,
	entities.BuildingGenerator:     5,
	entities.BuildingFusionReactor: 10,
}

func (ps *PowerSystem) OnTick(tick int64) {
	if tick%10 != 0 {
		return
	}

	ctx := ps.GetContext()
	if ctx == nil {
		return
	}

	// Use system entity planets (authoritative) instead of player.OwnedPlanets (stale)
	game := ctx.GetGame()
	if game == nil {
		return
	}
	for _, sys := range game.GetSystems() {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				computePower(planet)
			}
		}
	}
}

func computePower(planet *entities.Planet) {
	generated := 0.0
	consumed := 0.0

	// Population life support: 1 MW per 500 pop
	consumed += float64(planet.Population) / 500.0

	for _, be := range planet.Buildings {
		b, ok := be.(*entities.Building)
		if !ok || !b.IsOperational {
			continue
		}

		// Power draw
		if draw, ok := buildingPowerDraw[b.BuildingType]; ok {
			consumed += draw
		}

		staffing := b.GetStaffingRatio()
		levelMult := 1.0 + float64(b.Level-1)*0.3

		// Power generation
		switch b.BuildingType {
		case entities.BuildingGenerator:
			// Burns 2 Fuel per interval → 50 MW
			fuelNeeded := int(2.0 * levelMult)
			if planet.GetStoredAmount(entities.ResFuel) >= fuelNeeded {
				planet.RemoveStoredResource(entities.ResFuel, fuelNeeded)
				generated += 50.0 * levelMult * staffing
			}
		case entities.BuildingFusionReactor:
			// Burns 1 Helium-3 per interval → 200 MW
			he3Needed := int(1.0 * levelMult)
			if he3Needed < 1 {
				he3Needed = 1
			}
			if planet.GetStoredAmount(entities.ResHelium3) >= he3Needed {
				planet.RemoveStoredResource(entities.ResHelium3, he3Needed)
				generated += 200.0 * levelMult * staffing
			}
		}
	}

	planet.PowerGenerated = generated
	planet.PowerConsumed = consumed
	planet.PowerRatio = planet.GetPowerRatio()

	// Track power history for sparklines (keep last 50)
	planet.PowerHistory = append(planet.PowerHistory, planet.PowerRatio)
	if len(planet.PowerHistory) > 50 {
		planet.PowerHistory = planet.PowerHistory[len(planet.PowerHistory)-50:]
	}
}
