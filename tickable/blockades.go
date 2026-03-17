package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&BlockadeSystem{
		BaseSystem: NewBaseSystem("Blockades", 36),
	})
}

// BlockadeSystem lets military ships enforce blockades on systems.
// When a faction has 2+ military ships in a system where an enemy faction
// has planets, cargo ships belonging to that enemy cannot depart.
//
// Blockades create strategic pressure:
//   - Cut off enemy supply lines without direct combat
//   - Force enemies to build military to break the blockade
//   - Cargo ships caught in a blockade are intercepted (cargo seized)
//   - Neutral factions can still trade freely
//
// Breaking a blockade: bring enough military power to outmatch the blockader.
// The FleetCombat system handles the actual fighting if relations are Hostile.
type BlockadeSystem struct {
	*BaseSystem
	blockades map[int]*Blockade // systemID → active blockade
}

// Blockade represents a military blockade of a star system.
type Blockade struct {
	SystemID    int
	Enforcer    string // faction enforcing the blockade
	TargetOwner string // faction being blockaded ("" = all non-allies)
	ShipCount   int    // military ships enforcing
	TotalPower  int    // combined attack power
	TickStarted int64
	Active      bool
}

func (bs *BlockadeSystem) OnTick(tick int64) {
	if tick%300 != 0 {
		return
	}

	ctx := bs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	dm := game.GetDiplomacyManager()
	if dm == nil {
		return
	}

	if bs.blockades == nil {
		bs.blockades = make(map[int]*Blockade)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Evaluate blockade status for each system
	for _, sys := range systems {
		bs.evaluateBlockade(tick, sys, players, dm, game)
	}

	// Intercept cargo ships trying to operate in blockaded systems
	bs.interceptCargo(game, players, systems)
}

func (bs *BlockadeSystem) evaluateBlockade(tick int64, sys *entities.System, players []*entities.Player, dm interface{ GetRelation(a, b string) int }, game GameProvider) {
	// Count military presence per faction
	type fleetPresence struct {
		ships int
		power int
	}
	factions := make(map[string]*fleetPresence)

	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.CurrentSystem != sys.ID || ship.Status == entities.ShipStatusMoving {
				continue
			}
			if ship.ShipType != entities.ShipTypeFrigate &&
				ship.ShipType != entities.ShipTypeDestroyer &&
				ship.ShipType != entities.ShipTypeCruiser {
				continue
			}
			if factions[player.Name] == nil {
				factions[player.Name] = &fleetPresence{}
			}
			factions[player.Name].ships++
			factions[player.Name].power += ship.AttackPower
		}
	}

	// Find system owner (faction with planets here)
	systemOwners := make(map[string]bool)
	for _, e := range sys.Entities {
		if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
			systemOwners[planet.Owner] = true
		}
	}

	// Check if any faction can establish a blockade
	// Need 2+ military ships in a system where an enemy has planets
	existing := bs.blockades[sys.ID]

	for factionName, fleet := range factions {
		if fleet.ships < 2 {
			continue
		}

		// Check if there's an enemy faction with planets here
		for owner := range systemOwners {
			if owner == factionName {
				continue
			}
			relation := dm.GetRelation(factionName, owner)
			if relation > -1 {
				continue // only blockade enemies (Hostile or Unfriendly)
			}

			// Establish or maintain blockade
			if existing == nil || !existing.Active || existing.Enforcer != factionName {
				bs.blockades[sys.ID] = &Blockade{
					SystemID:    sys.ID,
					Enforcer:    factionName,
					TargetOwner: owner,
					ShipCount:   fleet.ships,
					TotalPower:  fleet.power,
					TickStarted: tick,
					Active:      true,
				}
				game.LogEvent("military", factionName,
					fmt.Sprintf("🚫 %s has established a blockade of %s! %s's cargo ships are cut off (%d warships enforcing)",
						factionName, sys.Name, owner, fleet.ships))
			} else {
				existing.ShipCount = fleet.ships
				existing.TotalPower = fleet.power
			}
			return
		}
	}

	// If no one qualifies, lift any existing blockade
	if existing != nil && existing.Active {
		// Check if enforcer still has ships
		enforcer := factions[existing.Enforcer]
		if enforcer == nil || enforcer.ships < 2 {
			existing.Active = false
			game.LogEvent("military", existing.TargetOwner,
				fmt.Sprintf("✅ Blockade of %s has been broken! %s's trade routes are free",
					sys.Name, existing.TargetOwner))
		}
	}
}

func (bs *BlockadeSystem) interceptCargo(game GameProvider, players []*entities.Player, systems []*entities.System) {
	for _, blockade := range bs.blockades {
		if !blockade.Active {
			continue
		}

		// Find cargo ships belonging to the blockaded faction in this system
		for _, player := range players {
			if player == nil || player.Name != blockade.TargetOwner {
				continue
			}
			for _, ship := range player.OwnedShips {
				if ship == nil || ship.CurrentSystem != blockade.SystemID {
					continue
				}
				if ship.ShipType != entities.ShipTypeCargo || ship.GetTotalCargo() == 0 {
					continue
				}
				// 15% chance per tick to intercept
				if rand.Intn(7) != 0 {
					continue
				}

				// Seize 20-40% of cargo
				seizureRate := 0.2 + rand.Float64()*0.2
				totalSeized := 0
				for res, amt := range ship.CargoHold {
					seized := int(float64(amt) * seizureRate)
					if seized > 0 {
						ship.CargoHold[res] -= seized
						if ship.CargoHold[res] <= 0 {
							delete(ship.CargoHold, res)
						}
						totalSeized += seized
					}
				}
				if totalSeized > 0 {
					// Enforcer gets a share as credits
					reward := totalSeized * 10
					for _, p := range players {
						if p != nil && p.Name == blockade.Enforcer {
							p.Credits += reward
							break
						}
					}

					sysName := fmt.Sprintf("SYS-%d", blockade.SystemID+1)
					for _, sys := range systems {
						if sys.ID == blockade.SystemID {
							sysName = sys.Name
							break
						}
					}
					game.LogEvent("military", player.Name,
						fmt.Sprintf("🚫 %s's cargo ship %s intercepted by %s's blockade in %s! %d units seized",
							player.Name, ship.Name, blockade.Enforcer, sysName, totalSeized))
				}
			}
		}
	}
}

// GetActiveBlockades returns all active blockades (for API/dashboard).
func (bs *BlockadeSystem) GetActiveBlockades() []*Blockade {
	var result []*Blockade
	for _, b := range bs.blockades {
		if b.Active {
			result = append(result, b)
		}
	}
	return result
}

// IsBlockaded checks if a faction's cargo is blocked in a specific system.
func (bs *BlockadeSystem) IsBlockaded(systemID int, faction string) bool {
	if bs.blockades == nil {
		return false
	}
	b, exists := bs.blockades[systemID]
	return exists && b.Active && b.TargetOwner == faction
}
