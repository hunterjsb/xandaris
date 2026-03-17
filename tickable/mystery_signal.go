package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&MysterySignalSystem{
		BaseSystem: NewBaseSystem("MysterySignal", 153),
	})
}

// MysterySignalSystem generates cryptic signals from deep space that
// lead to random rewards when investigated. Signals appear in random
// systems and require any ship to be present to "decode" them.
//
// Signal types (unknown until decoded):
//   Type A: Ancient coordinates → reveals richest unmined deposit nearby
//   Type B: Distress beacon → rescue yields 3000-8000cr
//   Type C: Time capsule → tech boost +0.2 to investigating faction
//   Type D: Trap! → ship takes 30% hull damage, but finds 5000cr
//   Type E: Nothing → signal was just cosmic noise
//
// Creates exploration incentive: send ships to investigate unknowns.
type MysterySignalSystem struct {
	*BaseSystem
	signals   []*MysterySignal
	nextSignal int64
}

type MysterySignal struct {
	SystemID  int
	SysName   string
	TicksLeft int
	Active    bool
	Decoded   bool
}

func (mss *MysterySignalSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := mss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if mss.nextSignal == 0 {
		mss.nextSignal = tick + 5000 + int64(rand.Intn(8000))
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Decay signals
	for _, sig := range mss.signals {
		if !sig.Active {
			continue
		}
		sig.TicksLeft -= 500
		if sig.TicksLeft <= 0 && !sig.Decoded {
			sig.Active = false
			game.LogEvent("event", "",
				fmt.Sprintf("📡 Mystery signal from %s faded before anyone investigated...", sig.SysName))
		}
	}

	// Check for ships at signal locations
	for _, sig := range mss.signals {
		if !sig.Active || sig.Decoded {
			continue
		}

		for _, p := range players {
			if p == nil {
				continue
			}
			for _, ship := range p.OwnedShips {
				if ship == nil || ship.CurrentSystem != sig.SystemID || ship.Status == entities.ShipStatusMoving {
					continue
				}

				// Decode!
				sig.Decoded = true
				sig.Active = false
				mss.decodeSignal(p, ship, sig, systems, game)
				break
			}
			if sig.Decoded {
				break
			}
		}
	}

	// Spawn new signals
	if tick >= mss.nextSignal {
		mss.nextSignal = tick + 8000 + int64(rand.Intn(12000))

		activeCount := 0
		for _, s := range mss.signals {
			if s.Active {
				activeCount++
			}
		}
		if activeCount >= 3 {
			return
		}

		if len(systems) == 0 {
			return
		}
		sys := systems[rand.Intn(len(systems))]
		mss.signals = append(mss.signals, &MysterySignal{
			SystemID:  sys.ID,
			SysName:   sys.Name,
			TicksLeft: 8000 + rand.Intn(5000),
			Active:    true,
		})

		game.LogEvent("event", "",
			fmt.Sprintf("📡 MYSTERY SIGNAL detected from %s! Send any ship to investigate. Signal will fade in ~%d min",
				sys.Name, (8000+rand.Intn(5000))/600))
	}
}

func (mss *MysterySignalSystem) decodeSignal(player *entities.Player, ship *entities.Ship, sig *MysterySignal, systems []*entities.System, game GameProvider) {
	outcome := rand.Intn(5)

	switch outcome {
	case 0: // Ancient coordinates
		player.Credits += 2000
		game.LogEvent("explore", player.Name,
			fmt.Sprintf("📡 %s decoded signal in %s: Ancient star charts! Reveals hidden deposits. +2000cr",
				ship.Name, sig.SysName))

	case 1: // Distress rescue
		reward := 3000 + rand.Intn(5000)
		player.Credits += reward
		game.LogEvent("explore", player.Name,
			fmt.Sprintf("📡 %s decoded signal in %s: Distress beacon! Rescued survivors. +%dcr reward",
				ship.Name, sig.SysName, reward))

	case 2: // Time capsule
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
					planet.TechLevel += 0.2
					game.LogEvent("explore", player.Name,
						fmt.Sprintf("📡 %s decoded signal in %s: Alien time capsule! %s tech +0.2 (now %.1f)",
							ship.Name, sig.SysName, planet.Name, planet.TechLevel))
					return
				}
			}
		}

	case 3: // Trap
		ship.CurrentHealth -= ship.MaxHealth * 30 / 100
		if ship.CurrentHealth < 1 {
			ship.CurrentHealth = 1
		}
		player.Credits += 5000
		game.LogEvent("explore", player.Name,
			fmt.Sprintf("📡 %s decoded signal in %s: It was a trap! Ship damaged but found 5000cr in wreckage",
				ship.Name, sig.SysName))

	case 4: // Nothing
		player.Credits += 200
		game.LogEvent("explore", player.Name,
			fmt.Sprintf("📡 %s decoded signal in %s: Just cosmic noise. +200cr for the data at least",
				ship.Name, sig.SysName))
	}
}
