package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ExtinctionEventSystem{
		BaseSystem: NewBaseSystem("ExtinctionEvent", 166),
	})
}

// ExtinctionEventSystem generates galaxy-threatening events that
// require cooperation to survive. These are the biggest, rarest
// events in the game — once per 200,000+ ticks.
//
// Events:
//   Gamma Ray Burst: approaching radiation wave will hit the galaxy
//     in 10,000 ticks. Factions with Planetary Shields survive intact.
//     Unshielded planets lose 50% population. Gives time to build shields.
//
//   Galactic Plague: super-plague spreads to ALL planets. Only factions
//     with Research Labs can develop a cure. Cured factions share the
//     cure with allies. Unfriended factions suffer longer.
//
// These events are rare enough to be legendary but devastating enough
// to reshape the galaxy when they happen.
type ExtinctionEventSystem struct {
	*BaseSystem
	firedGamma  bool
	firedPlague bool
	warningTick int64
	eventType   string
}

func (ees *ExtinctionEventSystem) OnTick(tick int64) {
	if tick < 200000 {
		return // too early
	}

	ctx := ees.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	// Only one extinction event per game
	if ees.firedGamma && ees.firedPlague {
		return
	}

	// Warning phase
	if ees.warningTick > 0 {
		if tick >= ees.warningTick+10000 {
			// Event strikes!
			ees.executeEvent(game, ctx)
			return
		}

		// Countdown reminders
		remaining := ees.warningTick + 10000 - tick
		if remaining%2000 == 0 {
			game.LogEvent("alert", "",
				fmt.Sprintf("☠️ EXTINCTION EVENT in ~%d minutes! Build Planetary Shields and Research Labs NOW!",
					remaining/600))
		}
		return
	}

	// 0.05% chance per 10,000 ticks after 200K
	if tick%10000 != 0 || rand.Intn(2000) != 0 {
		return
	}

	// Start warning
	if !ees.firedGamma {
		ees.eventType = "gamma"
		ees.firedGamma = true
	} else {
		ees.eventType = "plague"
		ees.firedPlague = true
	}
	ees.warningTick = tick

	if ees.eventType == "gamma" {
		game.LogEvent("alert", "",
			"☠️ GAMMA RAY BURST DETECTED! Massive radiation wave approaching the galaxy! Impact in ~17 minutes! BUILD PLANETARY SHIELDS ON ALL PLANETS!")
	} else {
		game.LogEvent("alert", "",
			"☠️ GALACTIC SUPER-PLAGUE spreading from the void! All planets will be infected in ~17 minutes! BUILD RESEARCH LABS to develop a cure!")
	}
}

func (ees *ExtinctionEventSystem) executeEvent(game GameProvider, ctx SystemContext) {
	systems := game.GetSystems()
	players := ctx.GetPlayers()

	if ees.eventType == "gamma" {
		shielded := 0
		hit := 0
		totalLost := int64(0)

		for _, sys := range systems {
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner == "" || planet.Population == 0 {
					continue
				}

				hasShield := false
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingPlanetShield && b.IsOperational {
						hasShield = true
						break
					}
				}

				if hasShield {
					shielded++
				} else {
					hit++
					loss := planet.Population / 2
					planet.Population -= loss
					totalLost += loss
				}
			}
		}

		game.LogEvent("event", "",
			fmt.Sprintf("☠️ GAMMA RAY BURST HIT! %d planets shielded (safe), %d unshielded (lost 50%% pop). Total casualties: %d. The galaxy endures.",
				shielded, hit, totalLost))
	} else {
		cured := 0
		infected := 0

		for _, sys := range systems {
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner == "" {
					continue
				}

				hasLab := false
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingResearchLab && b.IsOperational {
						hasLab = true
						break
					}
				}

				if hasLab {
					cured++
				} else {
					infected++
					planet.Population -= planet.Population / 4
					planet.Happiness -= 0.2
					if planet.Happiness < 0.05 {
						planet.Happiness = 0.05
					}
				}
			}
		}

		game.LogEvent("event", "",
			fmt.Sprintf("☠️ GALACTIC PLAGUE struck! %d planets cured (Research Labs), %d infected (lost 25%% pop). Science saves civilization!",
				cured, infected))
	}

	// Reward survivors
	for _, p := range players {
		if p != nil {
			p.Credits += 2000
		}
	}

	ees.warningTick = 0
	game.LogEvent("event", "",
		"🌟 The galaxy survived an extinction event! +2000cr to all factions. Build defenses for the next one!")
}
