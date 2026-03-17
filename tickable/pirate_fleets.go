package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&PirateFleetSystem{
		BaseSystem: NewBaseSystem("PirateFleets", 34),
	})
}

// PirateFleetSystem spawns pirate fleets that threaten trade routes.
// Pirate fleets appear in systems with high trade activity and raid
// cargo ships. Players can defeat pirates with Frigates/Destroyers
// to earn bounties and protect their trade routes.
//
// Pirates create demand for military ships and make trade risky.
type PirateFleetSystem struct {
	*BaseSystem
	pirates map[int]*PirateFleet // systemID → fleet
}

// PirateFleet represents a pirate presence in a system.
type PirateFleet struct {
	SystemID int
	Strength int // 1-5, determines how dangerous they are
	Bounty   int // credits for defeating them
	Active   bool
}

func (pfs *PirateFleetSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := pfs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if pfs.pirates == nil {
		pfs.pirates = make(map[int]*PirateFleet)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Spawn new pirates in random systems (max 3 active fleets)
	activeCount := 0
	for _, p := range pfs.pirates {
		if p.Active {
			activeCount++
		}
	}
	if activeCount < 3 && rand.Intn(3) == 0 {
		pfs.spawnPirates(game, systems)
	}

	// Pirates raid cargo ships in their system
	for _, pirate := range pfs.pirates {
		if !pirate.Active {
			continue
		}
		pfs.pirateRaid(game, pirate, players, systems)
		pfs.checkDefeat(game, pirate, players)
	}
}

func (pfs *PirateFleetSystem) spawnPirates(game GameProvider, systems []*entities.System) {
	if len(systems) == 0 {
		return
	}
	sys := systems[rand.Intn(len(systems))]

	// Don't spawn in systems that already have pirates
	if _, exists := pfs.pirates[sys.ID]; exists && pfs.pirates[sys.ID].Active {
		return
	}

	strength := 1 + rand.Intn(3)
	bounty := strength * (2000 + rand.Intn(3000))

	pfs.pirates[sys.ID] = &PirateFleet{
		SystemID: sys.ID,
		Strength: strength,
		Bounty:   bounty,
		Active:   true,
	}

	game.LogEvent("event", "",
		fmt.Sprintf("🏴‍☠️ Pirate fleet (strength %d) appeared in %s! Bounty: %d credits. Send military ships to clear them!",
			strength, sys.Name, bounty))
}

func (pfs *PirateFleetSystem) pirateRaid(game GameProvider, pirate *PirateFleet, players []*entities.Player, systems []*entities.System) {
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.CurrentSystem != pirate.SystemID {
				continue
			}
			if ship.ShipType != entities.ShipTypeCargo || ship.GetTotalCargo() == 0 {
				continue
			}
			// Convoy protection: escorted cargo ships are immune to pirate raids
			if HasEscort(ship, players) {
				continue
			}
			// Pirates steal cargo based on strength
			// Strength 1: 5%, Strength 3: 15%
			stealRate := float64(pirate.Strength) * 0.05
			if rand.Intn(3) != 0 {
				continue // 33% chance per tick
			}
			totalStolen := 0
			for res, amt := range ship.CargoHold {
				stolen := int(float64(amt) * stealRate)
				if stolen > 0 {
					ship.CargoHold[res] -= stolen
					if ship.CargoHold[res] <= 0 {
						delete(ship.CargoHold, res)
					}
					totalStolen += stolen
				}
			}
			if totalStolen > 0 {
				game.LogEvent("event", player.Name,
					fmt.Sprintf("🏴‍☠️ Pirates in %s raided %s! Lost %d units of cargo. (Send Frigates to clear them!)",
						systems[0].Name, ship.Name, totalStolen))
			}
		}
	}
}

func (pfs *PirateFleetSystem) checkDefeat(game GameProvider, pirate *PirateFleet, players []*entities.Player) {
	// Count military ships in the pirate's system
	militaryPower := 0
	var defeater *entities.Player
	for _, player := range players {
		if player == nil {
			continue
		}
		playerPower := 0
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.CurrentSystem != pirate.SystemID {
				continue
			}
			switch ship.ShipType {
			case entities.ShipTypeFrigate:
				playerPower += 2
			case entities.ShipTypeDestroyer:
				playerPower += 4
			case entities.ShipTypeCruiser:
				playerPower += 6
			}
		}
		if playerPower > 0 {
			militaryPower += playerPower
			if defeater == nil || playerPower > 0 {
				defeater = player
			}
		}
	}

	// Military power must exceed pirate strength to defeat
	if militaryPower >= pirate.Strength*2 && defeater != nil {
		pirate.Active = false
		defeater.Credits += pirate.Bounty
		game.LogEvent("event", defeater.Name,
			fmt.Sprintf("⚔️ %s defeated pirates in SYS-%d! Earned %d credits bounty!",
				defeater.Name, pirate.SystemID+1, pirate.Bounty))
	}
}
