package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&NaturalDisasterSystem{
		BaseSystem: NewBaseSystem("NaturalDisasters", 65),
	})
}

// NaturalDisasterSystem generates catastrophic planetary events
// that create emergencies requiring immediate response.
//
// Disasters:
//   Volcanic Eruption: destroys a mine, damages planet. Rich mineral
//     deposits appear after (new resource with high abundance).
//     Prevention: Planetary Shield reduces damage by 50%.
//
//   Tectonic Quake: damages 1-2 random buildings, population panics (-10% happiness).
//     Recovery: buildings auto-repair after 2000 ticks.
//
//   Magnetic Storm: disrupts all Electronics on planet (stored zeroed out).
//     Prevention: Research Lab recovers 50% of lost Electronics.
//
//   Comet Impact: massive damage + resource bonus. Destroys a building,
//     kills 5% population, but deposits rare resources (Rare Metals, He-3).
//
// Frequency: ~1 disaster per 15,000 ticks per owned planet.
// Disasters create demand for Planetary Shields and diversified economies.
type NaturalDisasterSystem struct {
	*BaseSystem
	nextDisaster int64
}

func (nds *NaturalDisasterSystem) OnTick(tick int64) {
	ctx := nds.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if nds.nextDisaster == 0 {
		nds.nextDisaster = tick + 8000 + int64(rand.Intn(10000))
	}
	if tick < nds.nextDisaster {
		return
	}
	nds.nextDisaster = tick + 12000 + int64(rand.Intn(10000))

	systems := game.GetSystems()

	// Find all owned planets
	var candidates []*entities.Planet
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" && planet.Population > 500 {
				candidates = append(candidates, planet)
			}
		}
	}
	if len(candidates) == 0 {
		return
	}

	planet := candidates[rand.Intn(len(candidates))]

	// Check for Planetary Shield (reduces disaster severity)
	hasShield := false
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingPlanetShield && b.IsOperational {
			hasShield = true
			break
		}
	}

	disasterType := rand.Intn(4)
	switch disasterType {
	case 0: // Volcanic Eruption
		nds.volcanicEruption(planet, hasShield, game)
	case 1: // Tectonic Quake
		nds.tectonicQuake(planet, hasShield, game)
	case 2: // Magnetic Storm
		nds.magneticStorm(planet, game)
	case 3: // Comet Impact
		nds.cometImpact(planet, hasShield, game)
	}
}

func (nds *NaturalDisasterSystem) volcanicEruption(planet *entities.Planet, hasShield bool, game GameProvider) {
	// Damage a mine
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingMine && b.IsOperational {
			if hasShield {
				game.LogEvent("event", planet.Owner,
					fmt.Sprintf("🌋 Volcanic eruption on %s! Planetary Shield reduced damage — mine still operational",
						planet.Name))
				return
			}
			b.IsOperational = false
			// But leave mineral deposits
			planet.AddStoredResource(entities.ResIron, 100+rand.Intn(200))
			planet.AddStoredResource(entities.ResRareMetals, 20+rand.Intn(50))
			game.LogEvent("event", planet.Owner,
				fmt.Sprintf("🌋 Volcanic eruption on %s! Mine damaged, but rich mineral deposits uncovered (+Iron, +Rare Metals)",
					planet.Name))
			return
		}
	}
	// No mine to damage
	planet.AddStoredResource(entities.ResIron, 50+rand.Intn(100))
	game.LogEvent("event", planet.Owner,
		fmt.Sprintf("🌋 Minor volcanic activity on %s. Mineral deposits revealed (+Iron)",
			planet.Name))
}

func (nds *NaturalDisasterSystem) tectonicQuake(planet *entities.Planet, hasShield bool, game GameProvider) {
	damaged := 0
	severity := 2
	if hasShield {
		severity = 1
	}
	for i, be := range planet.Buildings {
		if damaged >= severity {
			break
		}
		if b, ok := be.(*entities.Building); ok && b.IsOperational {
			_ = i
			b.IsOperational = false
			damaged++
		}
	}

	planet.Happiness -= 0.1
	if planet.Happiness < 0.1 {
		planet.Happiness = 0.1
	}

	msg := fmt.Sprintf("🏚️ Tectonic quake on %s! %d buildings damaged, population panicking (-10%% happiness)",
		planet.Name, damaged)
	if hasShield {
		msg += " — shield reduced severity"
	}
	game.LogEvent("event", planet.Owner, msg)
}

func (nds *NaturalDisasterSystem) magneticStorm(planet *entities.Planet, game GameProvider) {
	stored := planet.GetStoredAmount(entities.ResElectronics)
	if stored <= 0 {
		return
	}

	// Research Lab salvages 50%
	hasLab := false
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingResearchLab && b.IsOperational {
			hasLab = true
			break
		}
	}

	loss := stored
	if hasLab {
		loss = stored / 2
	}
	planet.RemoveStoredResource(entities.ResElectronics, loss)

	msg := fmt.Sprintf("🧲 Magnetic storm on %s! Lost %d Electronics", planet.Name, loss)
	if hasLab {
		msg += fmt.Sprintf(" (Research Lab saved %d)", stored-loss)
	} else {
		msg += " — build a Research Lab to protect against future storms!"
	}
	game.LogEvent("event", planet.Owner, msg)
}

func (nds *NaturalDisasterSystem) cometImpact(planet *entities.Planet, hasShield bool, game GameProvider) {
	if hasShield {
		// Shield absorbs impact
		planet.AddStoredResource(entities.ResRareMetals, 30+rand.Intn(40))
		planet.AddStoredResource(entities.ResHelium3, 10+rand.Intn(20))
		game.LogEvent("event", planet.Owner,
			fmt.Sprintf("☄️ Comet intercepted by %s's Planetary Shield! Debris collected (+Rare Metals, +Helium-3)",
				planet.Name))
		return
	}

	// Destruction + resources
	popLoss := planet.Population / 20 // 5%
	planet.Population -= popLoss

	// Damage a building
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok && b.IsOperational {
			b.IsOperational = false
			break
		}
	}

	// But deposit rare materials
	planet.AddStoredResource(entities.ResRareMetals, 50+rand.Intn(80))
	planet.AddStoredResource(entities.ResHelium3, 20+rand.Intn(40))

	game.LogEvent("event", planet.Owner,
		fmt.Sprintf("☄️ COMET IMPACT on %s! %d casualties, building destroyed — but rare materials deposited (+RM, +He-3). Build a Planetary Shield!",
			planet.Name, popLoss))
}
