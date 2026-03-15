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
	"Base":           10,
	"Mine":           15,
	"Habitat":        10,
	"Trading Post":   10,
	"Refinery":       25,
	"Factory":        30,
	"Shipyard":       35,
	"Generator":      5,
	"Fusion Reactor": 10,
}

func (ps *PowerSystem) OnTick(tick int64) {
	if tick%10 != 0 {
		return
	}

	ctx := ps.GetContext()
	if ctx == nil {
		return
	}
	playersIface := ctx.GetPlayers()
	if playersIface == nil {
		return
	}
	players, ok := playersIface.([]*entities.Player)
	if !ok {
		return
	}

	for _, player := range players {
		for _, planet := range player.OwnedPlanets {
			if planet != nil {
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
		case "Generator":
			// Burns 2 Fuel per interval → 50 MW
			fuelNeeded := int(2.0 * levelMult)
			if planet.GetStoredAmount("Fuel") >= fuelNeeded {
				planet.RemoveStoredResource("Fuel", fuelNeeded)
				generated += 50.0 * levelMult * staffing
			}
		case "Fusion Reactor":
			// Burns 1 Helium-3 per interval → 200 MW
			he3Needed := int(1.0 * levelMult)
			if he3Needed < 1 {
				he3Needed = 1
			}
			if planet.GetStoredAmount("Helium-3") >= he3Needed {
				planet.RemoveStoredResource("Helium-3", he3Needed)
				generated += 200.0 * levelMult * staffing
			}
		}
	}

	planet.PowerGenerated = generated
	planet.PowerConsumed = consumed
	planet.PowerRatio = planet.GetPowerRatio()
}
