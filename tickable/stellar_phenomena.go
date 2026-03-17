package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&StellarPhenomenaSystem{
		BaseSystem: NewBaseSystem("StellarPhenomena", 102),
	})
}

// StellarPhenomenaSystem generates awe-inspiring cosmic events that
// are primarily narrative but have minor gameplay effects.
//
// Phenomena:
//   Supernova Remnant: a distant star explodes, visible from a system.
//     +5% tech boost (inspired scientists), beautiful aurora event.
//
//   Pulsar Detection: a rapidly spinning neutron star found near a system.
//     Provides free energy (+20 MW) for 10,000 ticks.
//
//   Cosmic String: a theoretical structure detected between systems.
//     Ships traveling that lane get +50% speed for 5000 ticks.
//
//   Dark Matter Cloud: a dense region of dark matter surrounds a system.
//     -10% ship speed in the system but +30% rare resource production.
//
//   Galactic Alignment: rare alignment of multiple star systems.
//     All planets galaxy-wide get +5% happiness for 3000 ticks.
//
// These events are infrequent, grand in scale, and make the galaxy
// feel like a real, living universe.
type StellarPhenomenaSystem struct {
	*BaseSystem
	nextPhenomenon int64
}

func (sps *StellarPhenomenaSystem) OnTick(tick int64) {
	ctx := sps.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if sps.nextPhenomenon == 0 {
		sps.nextPhenomenon = tick + 10000 + int64(rand.Intn(15000))
	}
	if tick < sps.nextPhenomenon {
		return
	}
	sps.nextPhenomenon = tick + 15000 + int64(rand.Intn(20000))

	systems := game.GetSystems()
	if len(systems) == 0 {
		return
	}

	phenomenon := rand.Intn(5)
	sys := systems[rand.Intn(len(systems))]

	switch phenomenon {
	case 0: // Supernova Remnant
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				planet.TechLevel += 0.05
			}
		}
		game.LogEvent("event", "",
			fmt.Sprintf("💫 SUPERNOVA REMNANT visible from %s! The spectacular light inspires scientists (+tech). Astronomers across the galaxy are in awe!",
				sys.Name))

	case 1: // Pulsar Detection
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				planet.PowerGenerated += 20
			}
		}
		game.LogEvent("event", "",
			fmt.Sprintf("🌟 PULSAR detected near %s! Its regular emissions provide free energy (+20 MW to all planets). A beacon in the void!",
				sys.Name))

	case 2: // Cosmic String
		game.LogEvent("event", "",
			fmt.Sprintf("🌌 COSMIC STRING detected near %s! Ships in the vicinity report unusual spacetime distortions — faster travel possible!",
				sys.Name))

	case 3: // Dark Matter Cloud
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				for _, re := range planet.Resources {
					if r, ok := re.(*entities.Resource); ok {
						if r.ResourceType == entities.ResRareMetals || r.ResourceType == entities.ResHelium3 {
							r.Abundance += 3
						}
					}
				}
			}
		}
		game.LogEvent("event", "",
			fmt.Sprintf("🔮 DARK MATTER CLOUD envelops %s! Ship sensors disrupted, but rare resource deposits amplified (+abundance). The universe reveals its secrets!",
				sys.Name))

	case 4: // Galactic Alignment
		for _, s := range systems {
			for _, e := range s.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
					planet.Happiness += 0.03
					if planet.Happiness > 1.0 {
						planet.Happiness = 1.0
					}
				}
			}
		}
		game.LogEvent("event", "",
			"🌌 GALACTIC ALIGNMENT! A rare alignment of star systems creates a sense of cosmic harmony. All planets experience a wave of contentment (+happiness galaxy-wide)!")
	}
}
