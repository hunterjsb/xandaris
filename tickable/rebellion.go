package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&RebellionSystem{
		BaseSystem: NewBaseSystem("Rebellion", 48),
	})
}

// RebellionSystem causes planets with extremely low happiness to rebel.
// Rebellions are the ultimate consequence of neglecting your population.
//
// Stages:
//   1. Unrest (happiness < 0.20 for 3000+ ticks): warning, -25% productivity
//   2. Riots (happiness < 0.15 for 5000+ ticks): buildings damaged, population flees
//   3. Rebellion (happiness < 0.10 for 8000+ ticks): planet goes independent (owner = "")
//
// Rebellions can be prevented by:
//   - Supplying resources to raise happiness
//   - Stationing military ships in the system (intimidation)
//   - Building a Planetary Shield (counts as garrison)
//
// An independent rebellious planet keeps its buildings and population.
// Any faction can re-colonize it with a Colony Ship.
type RebellionSystem struct {
	*BaseSystem
	unrestTimers map[int]int64 // planetID → tick when unrest began
	warned       map[int]bool  // planetID → already warned
}

func (rs *RebellionSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := rs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if rs.unrestTimers == nil {
		rs.unrestTimers = make(map[int]int64)
		rs.warned = make(map[int]bool)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.Population < 1000 {
				continue
			}

			rs.evaluatePlanet(tick, planet, sys, players, game)
		}
	}
}

func (rs *RebellionSystem) evaluatePlanet(tick int64, planet *entities.Planet, sys *entities.System, players []*entities.Player, game GameProvider) {
	pid := planet.GetID()

	// Check for military presence (suppresses rebellion)
	hasMilitary := false
	for _, p := range players {
		if p == nil || p.Name != planet.Owner {
			continue
		}
		for _, ship := range p.OwnedShips {
			if ship == nil || ship.CurrentSystem != sys.ID {
				continue
			}
			if ship.ShipType == entities.ShipTypeFrigate ||
				ship.ShipType == entities.ShipTypeDestroyer ||
				ship.ShipType == entities.ShipTypeCruiser {
				hasMilitary = true
				break
			}
		}
		break
	}

	// Check for Planetary Shield (acts as garrison)
	if !hasMilitary {
		for _, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingPlanetShield && b.IsOperational {
				hasMilitary = true
				break
			}
		}
	}

	// Happy planets or garrisoned planets: clear unrest
	if planet.Happiness >= 0.20 || hasMilitary {
		if rs.unrestTimers[pid] > 0 {
			delete(rs.unrestTimers, pid)
			delete(rs.warned, pid)
		}
		return
	}

	// Start tracking unrest
	if rs.unrestTimers[pid] == 0 {
		rs.unrestTimers[pid] = tick
	}

	unrestDuration := tick - rs.unrestTimers[pid]

	// Stage 1: Unrest warning (3000+ ticks)
	if unrestDuration >= 3000 && !rs.warned[pid] {
		rs.warned[pid] = true
		game.LogEvent("alert", planet.Owner,
			fmt.Sprintf("⚠️ UNREST on %s! Happiness at %.0f%%. Citizens are restless — supply resources or station troops!",
				planet.Name, planet.Happiness*100))
	}

	// Stage 2: Riots (5000+ ticks) — damage buildings, population flees
	if unrestDuration >= 5000 && unrestDuration < 8000 {
		if rand.Intn(3) == 0 {
			// Damage a random building (never the Base)
			if len(planet.Buildings) > 0 {
				idx := rand.Intn(len(planet.Buildings))
				if b, ok := planet.Buildings[idx].(*entities.Building); ok && b.IsOperational && b.BuildingType != entities.BuildingBase {
					b.IsOperational = false
					game.LogEvent("alert", planet.Owner,
						fmt.Sprintf("🔥 Riots on %s! %s has been damaged and is offline",
							planet.Name, b.Name))
				}
			}

			// Population flees
			fled := planet.Population / 10
			if fled > 0 {
				planet.Population -= fled
			}
		}
	}

	// Stage 3: Full rebellion (8000+ ticks) — planet goes independent
	if unrestDuration >= 8000 {
		oldOwner := planet.Owner
		planet.Owner = ""

		// Remove from player's owned planets
		for _, p := range players {
			if p == nil || p.Name != oldOwner {
				continue
			}
			for i, op := range p.OwnedPlanets {
				if op != nil && op.GetID() == planet.GetID() {
					p.OwnedPlanets = append(p.OwnedPlanets[:i], p.OwnedPlanets[i+1:]...)
					break
				}
			}
			break
		}

		// Clear resource ownership
		for _, re := range planet.Resources {
			if r, ok := re.(*entities.Resource); ok {
				r.Owner = ""
			}
		}

		delete(rs.unrestTimers, planet.GetID())
		delete(rs.warned, planet.GetID())

		game.LogEvent("event", oldOwner,
			fmt.Sprintf("🏴 REBELLION! %s has declared independence from %s! The population has overthrown their rulers. Any faction can re-colonize.",
				planet.Name, oldOwner))
	}
}
