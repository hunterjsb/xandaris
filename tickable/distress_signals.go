package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&DistressSignalSystem{
		BaseSystem: NewBaseSystem("DistressSignals", 64),
	})
}

// DistressSignalSystem generates random distress signals from NPC
// vessels and colonies, creating rescue missions that reward
// the faction that responds first.
//
// Distress types:
//   Stranded Freighter: cargo ship stranded in a system. Send a ship
//     to claim free cargo (200-500 units of random resource)
//
//   Colony in Peril: NPC settlement running out of Water/Food.
//     Deliver 50+ units of needed resource → +2000cr + reputation
//
//   Science Vessel: research ship discovered something but is damaged.
//     Send military escort → tech boost (+0.3) to your nearest planet
//
//   Diplomatic Envoy: neutral faction ambassador needs safe passage.
//     Send a Frigate to the system → +500 reputation + diplomacy bonus
//
// Distress signals expire after 5000 ticks if no one responds.
// This creates emergent quests that give players something to do
// beyond just trading and building.
type DistressSignalSystem struct {
	*BaseSystem
	signals   []*DistressSignal
	nextSignal int64
}

// DistressSignal represents an active distress call.
type DistressSignal struct {
	ID        int
	SystemID  int
	SysName   string
	Type      string // "freighter", "colony", "science", "envoy"
	Resource  string // for colony type: what they need
	Reward    int
	TicksLeft int
	Active    bool
	Claimed   bool
	ClaimedBy string
}

func (dss *DistressSignalSystem) OnTick(tick int64) {
	if tick%200 != 0 {
		return
	}

	ctx := dss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if dss.nextSignal == 0 {
		dss.nextSignal = tick + 2000 + int64(rand.Intn(3000))
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Decay existing signals
	for _, sig := range dss.signals {
		if !sig.Active {
			continue
		}
		sig.TicksLeft -= 200
		if sig.TicksLeft <= 0 {
			sig.Active = false
			if !sig.Claimed {
				game.LogEvent("event", "",
					fmt.Sprintf("📡 Distress signal from %s has gone silent. No one responded in time...",
						sig.SysName))
			}
		}
	}

	// Check for responses
	for _, sig := range dss.signals {
		if !sig.Active || sig.Claimed {
			continue
		}
		dss.checkResponse(sig, players, systems, game)
	}

	// Generate new signals
	if tick >= dss.nextSignal {
		dss.nextSignal = tick + 5000 + int64(rand.Intn(8000))

		// Max 3 active signals
		activeCount := 0
		for _, s := range dss.signals {
			if s.Active {
				activeCount++
			}
		}
		if activeCount < 3 {
			dss.generateSignal(game, systems)
		}
	}
}

func (dss *DistressSignalSystem) checkResponse(sig *DistressSignal, players []*entities.Player, systems []*entities.System, game GameProvider) {
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.CurrentSystem != sig.SystemID || ship.Status == entities.ShipStatusMoving {
				continue
			}

			switch sig.Type {
			case "freighter":
				// Any ship can claim stranded cargo
				resources := []string{entities.ResIron, entities.ResOil, entities.ResRareMetals, entities.ResWater}
				res := resources[rand.Intn(len(resources))]
				qty := 200 + rand.Intn(300)
				loaded := ship.AddCargo(res, qty)
				if loaded > 0 {
					sig.Claimed = true
					sig.ClaimedBy = player.Name
					player.Credits += sig.Reward
					game.LogEvent("event", player.Name,
						fmt.Sprintf("🆘 %s responded to distress signal in %s! Rescued %d %s from stranded freighter +%dcr",
							player.Name, sig.SysName, loaded, res, sig.Reward))
					return
				}

			case "science":
				// Military ship required
				if ship.ShipType == entities.ShipTypeFrigate ||
					ship.ShipType == entities.ShipTypeDestroyer ||
					ship.ShipType == entities.ShipTypeCruiser {
					sig.Claimed = true
					sig.ClaimedBy = player.Name
					player.Credits += sig.Reward
					// Tech boost to nearest planet
					for _, sys := range systems {
						for _, e := range sys.Entities {
							if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
								planet.TechLevel += 0.3
								game.LogEvent("event", player.Name,
									fmt.Sprintf("🆘 %s escorted a science vessel in %s! Tech data shared — %s tech +0.3 (now %.1f) +%dcr",
										player.Name, sig.SysName, planet.Name, planet.TechLevel, sig.Reward))
								return
							}
						}
					}
				}

			case "envoy":
				// Frigate+ required
				if ship.ShipType == entities.ShipTypeFrigate ||
					ship.ShipType == entities.ShipTypeDestroyer ||
					ship.ShipType == entities.ShipTypeCruiser {
					sig.Claimed = true
					sig.ClaimedBy = player.Name
					player.Credits += sig.Reward
					game.LogEvent("event", player.Name,
						fmt.Sprintf("🆘 %s provided safe passage for diplomatic envoy in %s! +%dcr + galactic reputation boost",
							player.Name, sig.SysName, sig.Reward))
					return
				}

			case "colony":
				// Need cargo ship with the right resource
				if ship.ShipType == entities.ShipTypeCargo && ship.CargoHold[sig.Resource] >= 50 {
					sig.Claimed = true
					sig.ClaimedBy = player.Name
					ship.RemoveCargo(sig.Resource, 50)
					player.Credits += sig.Reward
					game.LogEvent("event", player.Name,
						fmt.Sprintf("🆘 %s delivered %s to colony in peril at %s! +%dcr for humanitarian aid",
							player.Name, sig.Resource, sig.SysName, sig.Reward))
					return
				}
			}
		}
	}
}

func (dss *DistressSignalSystem) generateSignal(game GameProvider, systems []*entities.System) {
	if len(systems) == 0 {
		return
	}

	sys := systems[rand.Intn(len(systems))]
	signalType := rand.Intn(4)

	var sig *DistressSignal
	switch signalType {
	case 0:
		sig = &DistressSignal{
			ID: len(dss.signals) + 1, SystemID: sys.ID, SysName: sys.Name,
			Type: "freighter", Reward: 1000 + rand.Intn(2000),
			TicksLeft: 5000 + rand.Intn(3000), Active: true,
		}
		game.LogEvent("event", "",
			fmt.Sprintf("📡 DISTRESS: Stranded freighter in %s! Send any ship to salvage cargo and earn %dcr",
				sys.Name, sig.Reward))

	case 1:
		resources := []string{entities.ResWater, entities.ResIron}
		res := resources[rand.Intn(len(resources))]
		sig = &DistressSignal{
			ID: len(dss.signals) + 1, SystemID: sys.ID, SysName: sys.Name,
			Type: "colony", Resource: res, Reward: 2000 + rand.Intn(3000),
			TicksLeft: 6000 + rand.Intn(4000), Active: true,
		}
		game.LogEvent("event", "",
			fmt.Sprintf("📡 DISTRESS: Colony in %s running out of %s! Deliver 50+ units for %dcr reward",
				sys.Name, res, sig.Reward))

	case 2:
		sig = &DistressSignal{
			ID: len(dss.signals) + 1, SystemID: sys.ID, SysName: sys.Name,
			Type: "science", Reward: 1500 + rand.Intn(2500),
			TicksLeft: 4000 + rand.Intn(3000), Active: true,
		}
		game.LogEvent("event", "",
			fmt.Sprintf("📡 DISTRESS: Science vessel in %s needs military escort! Send a warship for tech data + %dcr",
				sys.Name, sig.Reward))

	case 3:
		sig = &DistressSignal{
			ID: len(dss.signals) + 1, SystemID: sys.ID, SysName: sys.Name,
			Type: "envoy", Reward: 2000 + rand.Intn(3000),
			TicksLeft: 5000 + rand.Intn(3000), Active: true,
		}
		game.LogEvent("event", "",
			fmt.Sprintf("📡 DISTRESS: Diplomatic envoy in %s requests escort! Send a warship for %dcr + reputation",
				sys.Name, sig.Reward))
	}

	if sig != nil {
		dss.signals = append(dss.signals, sig)
	}
}

// GetActiveSignals returns active distress signals.
func (dss *DistressSignalSystem) GetActiveSignals() []*DistressSignal {
	var result []*DistressSignal
	for _, s := range dss.signals {
		if s.Active && !s.Claimed {
			result = append(result, s)
		}
	}
	return result
}
