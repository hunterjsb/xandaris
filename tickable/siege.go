package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SiegeSystem{
		BaseSystem: NewBaseSystem("Siege", 38),
	})
}

// SiegeSystem allows military ships to lay siege to enemy planets.
// A siege is a sustained bombardment that degrades the planet's
// infrastructure over time without requiring ground troops.
//
// Siege mechanics:
//   - Need 3+ military ships in system to siege an enemy planet
//   - Each tick: random building takes damage (may go offline)
//   - Planetary Shield absorbs bombardment (shield must fall first)
//   - Population flees at 2% per tick during siege
//   - Defending military ships in the system break the siege
//
// If all buildings are destroyed and population drops below 1000,
// the planet is "conquered" — ownership transfers to the attacker.
//
// This creates the full military arc: build ships → blockade → siege → conquer.
// Defense requires Planetary Shields and a standing military fleet.
type SiegeSystem struct {
	*BaseSystem
	sieges map[int]*Siege // planetID → active siege
}

// Siege represents an active bombardment of a planet.
type Siege struct {
	PlanetID    int
	Attacker    string
	Defender    string
	ShipCount   int
	TotalPower  int
	TickStarted int64
	Active      bool
}

func (ss *SiegeSystem) OnTick(tick int64) {
	if tick%400 != 0 {
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

	dm := game.GetDiplomacyManager()
	if dm == nil {
		return
	}

	if ss.sieges == nil {
		ss.sieges = make(map[int]*Siege)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, sys := range systems {
		ss.evaluateSieges(tick, sys, players, dm, game)
	}
}

func (ss *SiegeSystem) evaluateSieges(tick int64, sys *entities.System, players []*entities.Player, dm interface{ GetRelation(a, b string) int }, game GameProvider) {
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

	// Check each planet for siege conditions
	for _, e := range sys.Entities {
		planet, ok := e.(*entities.Planet)
		if !ok || planet.Owner == "" {
			continue
		}
		pid := planet.GetID()

		// Check if defender has military here (breaks siege)
		defenderFleet := factions[planet.Owner]

		// Check all attacker factions
		for attackerName, fleet := range factions {
			if attackerName == planet.Owner {
				continue
			}
			relation := dm.GetRelation(attackerName, planet.Owner)
			if relation > -2 {
				continue // only Hostile factions siege
			}

			// Need 3+ military ships and more power than defender
			if fleet.ships < 3 {
				continue
			}
			defPower := 0
			if defenderFleet != nil {
				defPower = defenderFleet.power
			}
			if fleet.power <= defPower {
				// Defender matches attacker — siege broken
				if siege, exists := ss.sieges[pid]; exists && siege.Active && siege.Attacker == attackerName {
					siege.Active = false
					game.LogEvent("military", planet.Owner,
						fmt.Sprintf("✅ Siege of %s broken! %s's fleet defended the planet",
							planet.Name, planet.Owner))
				}
				continue
			}

			// Start or continue siege
			siege, exists := ss.sieges[pid]
			if !exists || !siege.Active {
				ss.sieges[pid] = &Siege{
					PlanetID:    pid,
					Attacker:    attackerName,
					Defender:    planet.Owner,
					ShipCount:   fleet.ships,
					TotalPower:  fleet.power,
					TickStarted: tick,
					Active:      true,
				}
				game.LogEvent("military", planet.Owner,
					fmt.Sprintf("🔥 %s is laying SIEGE to %s in %s! %d warships bombarding! Send reinforcements!",
						attackerName, planet.Name, sys.Name, fleet.ships))
				game.LogEvent("military", attackerName,
					fmt.Sprintf("🔥 Siege of %s in %s begun! %d ships bombarding %s's planet",
						planet.Name, sys.Name, fleet.ships, planet.Owner))
				siege = ss.sieges[pid]
			} else {
				siege.ShipCount = fleet.ships
				siege.TotalPower = fleet.power
			}

			// Apply siege damage
			ss.applySiegeDamage(planet, siege, players, game)
			return // one siege per system per tick
		}
	}
}

func (ss *SiegeSystem) applySiegeDamage(planet *entities.Planet, siege *Siege, players []*entities.Player, game GameProvider) {
	// Check for Planetary Shield — must be knocked out first
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingPlanetShield && b.IsOperational {
			// Shield absorbs bombardment
			if rand.Intn(3) == 0 {
				b.IsOperational = false
				game.LogEvent("military", planet.Owner,
					fmt.Sprintf("💥 Planetary Shield on %s destroyed by bombardment!", planet.Name))
			} else {
				game.LogEvent("military", planet.Owner,
					fmt.Sprintf("🛡️ %s's Planetary Shield absorbing bombardment...", planet.Name))
			}
			return // shield takes the hit this tick
		}
	}

	// Damage a random operational building (never the Base)
	if len(planet.Buildings) > 0 && rand.Intn(2) == 0 {
		// Find operational buildings (excluding Base)
		var operational []int
		for i, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok && b.IsOperational && b.BuildingType != entities.BuildingBase {
				operational = append(operational, i)
			}
		}
		if len(operational) > 0 {
			idx := operational[rand.Intn(len(operational))]
			if b, ok := planet.Buildings[idx].(*entities.Building); ok {
				b.IsOperational = false
				game.LogEvent("military", planet.Owner,
					fmt.Sprintf("💥 Bombardment destroyed %s on %s!", b.Name, planet.Name))
			}
		}
	}

	// Population flees
	fled := planet.Population / 50 // 2% per tick
	if fled > 0 {
		planet.Population -= fled
		if planet.Population < 500 {
			planet.Population = 500 // Colony core survives even under siege
		}
	}

	// Check for conquest: all buildings destroyed + population low
	operationalCount := 0
	for _, be := range planet.Buildings {
		if b, ok := be.(*entities.Building); ok && b.IsOperational {
			operationalCount++
		}
	}

	if operationalCount == 0 && planet.Population < 1000 {
		// Planet conquered!
		oldOwner := planet.Owner
		planet.Owner = siege.Attacker

		// Update resource ownership
		for _, re := range planet.Resources {
			if r, ok := re.(*entities.Resource); ok {
				r.Owner = siege.Attacker
			}
		}

		// Add to attacker's planets, remove from defender's
		for _, p := range players {
			if p == nil {
				continue
			}
			if p.Name == siege.Attacker {
				p.OwnedPlanets = append(p.OwnedPlanets, planet)
			}
			if p.Name == oldOwner {
				for i, op := range p.OwnedPlanets {
					if op != nil && op.GetID() == planet.GetID() {
						p.OwnedPlanets = append(p.OwnedPlanets[:i], p.OwnedPlanets[i+1:]...)
						break
					}
				}
			}
		}

		siege.Active = false
		delete(ss.sieges, planet.GetID())

		game.LogEvent("event", "",
			fmt.Sprintf("🏴 CONQUEST! %s has captured %s from %s! The planet's defenses crumbled under siege.",
				siege.Attacker, planet.Name, oldOwner))
	}
}

// GetActiveSieges returns all active sieges (for API/dashboard).
func (ss *SiegeSystem) GetActiveSieges() []*Siege {
	var result []*Siege
	for _, s := range ss.sieges {
		if s.Active {
			result = append(result, s)
		}
	}
	return result
}
