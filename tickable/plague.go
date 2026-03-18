package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&PlagueSystem{
		BaseSystem: NewBaseSystem("Plague", 43),
	})
}

// PlagueSystem introduces diseases that spread between planets via trade.
// Plagues create tension between trade and safety: more trade connections
// mean more exposure to disease.
//
// Plague lifecycle:
//   1. Outbreak: random planet gets infected (more likely on crowded planets)
//   2. Spread: cargo ships that dock at infected planets carry the plague
//   3. Effect: infected planets lose 1-3% population per tick, -30% happiness
//   4. Cure: planets with high tech (3.0+) or Research Labs cure faster
//   5. Quarantine: not docking at infected planets prevents spread
//
// Prevention:
//   - Research Lab on planet: cures in 2000 ticks instead of 5000
//   - Planetary Shield: 50% chance to block incoming plague
//   - High tech level: faster natural recovery
//
// This creates a quarantine dilemma: do you stop trading to prevent
// spread, or keep trading and risk pandemic?
type PlagueSystem struct {
	*BaseSystem
	infections map[int]*Infection // planetID → active infection
	nextCheck  int64
}

// Infection tracks an active plague on a planet.
type Infection struct {
	PlanetID   int
	Severity   float64 // 0.01-0.03 population loss per tick
	TicksLeft  int
	Source     string // how it arrived: "outbreak", "trade", "refugee"
}

func (ps *PlagueSystem) OnTick(tick int64) {
	if tick%300 != 0 {
		return
	}

	ctx := ps.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ps.infections == nil {
		ps.infections = make(map[int]*Infection)
	}

	if ps.nextCheck == 0 {
		ps.nextCheck = tick + 5000 + int64(rand.Intn(10000))
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Process existing infections
	for pid, inf := range ps.infections {
		inf.TicksLeft -= 300
		if inf.TicksLeft <= 0 {
			delete(ps.infections, pid)
			// Find planet to announce cure
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if planet, ok := e.(*entities.Planet); ok && planet.GetID() == pid {
						game.LogEvent("event", planet.Owner,
							fmt.Sprintf("💊 Plague on %s has been eradicated! Population recovering", planet.Name))
					}
				}
			}
			continue
		}

		ps.applyInfection(pid, inf, systems, game)
	}

	// Spread via cargo ships docked at infected planets
	ps.spreadViaTrade(players, systems, game)

	// Random new outbreak
	if tick >= ps.nextCheck {
		ps.nextCheck = tick + 15000 + int64(rand.Intn(20000))
		ps.randomOutbreak(systems, game)
	}
}

func (ps *PlagueSystem) applyInfection(pid int, inf *Infection, systems []*entities.System, game GameProvider) {
	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.GetID() != pid || planet.Owner == "" {
				continue
			}

			// Population loss
			loss := int64(float64(planet.Population) * inf.Severity)
			if loss > 0 && planet.Population > loss {
				planet.Population -= loss
				if planet.Population < 500 {
					planet.Population = 500 // Colony core survives
				}
			}

			// Happiness penalty
			planet.Happiness -= 0.05
			if planet.Happiness < 0.05 {
				planet.Happiness = 0.05
			}

			// Research Lab or high tech accelerates cure
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingResearchLab && b.IsOperational {
					inf.TicksLeft -= 200 // cures 2x faster
					break
				}
			}
			if planet.TechLevel >= 3.0 {
				inf.TicksLeft -= 100 // tech accelerates cure
			}

			return
		}
	}
}

func (ps *PlagueSystem) spreadViaTrade(players []*entities.Player, systems []*entities.System, game GameProvider) {
	// Check cargo ships that recently docked
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.ShipType != entities.ShipTypeCargo || ship.DockedAtPlanet == 0 {
				continue
			}

			// Is the ship docked at an infected planet?
			isInfectedPort := false
			for pid := range ps.infections {
				if ship.DockedAtPlanet == pid {
					isInfectedPort = true
					break
				}
			}
			if !isInfectedPort {
				continue
			}

			// Ship is at an infected port — when it docks elsewhere, it may spread
			// Check all planets in the same system for spread
			for _, sys := range systems {
				if sys.ID != ship.CurrentSystem {
					continue
				}
				for _, e := range sys.Entities {
					planet, ok := e.(*entities.Planet)
					if !ok || planet.Owner == "" || planet.GetID() == ship.DockedAtPlanet {
						continue
					}
					if _, already := ps.infections[planet.GetID()]; already {
						continue
					}

					// 10% chance to spread per tick
					if rand.Intn(10) != 0 {
						continue
					}

					// Planetary Shield blocks 50%
					hasShield := false
					for _, be := range planet.Buildings {
						if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingPlanetShield && b.IsOperational {
							hasShield = true
							break
						}
					}
					if hasShield && rand.Intn(2) == 0 {
						game.LogEvent("event", planet.Owner,
							fmt.Sprintf("🛡️ Planetary Shield on %s blocked an incoming plague!", planet.Name))
						continue
					}

					ps.infections[planet.GetID()] = &Infection{
						PlanetID:  planet.GetID(),
						Severity:  0.005 + rand.Float64()*0.015,
						TicksLeft: 3000 + rand.Intn(3000),
						Source:    "trade",
					}
					game.LogEvent("alert", planet.Owner,
						fmt.Sprintf("🦠 Plague has spread to %s via trade ship! Population at risk. Build a Research Lab to accelerate cure!",
							planet.Name))
				}
				break
			}
		}
	}
}

func (ps *PlagueSystem) randomOutbreak(systems []*entities.System, game GameProvider) {
	// More likely on crowded planets
	var candidates []*entities.Planet
	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.Population < 5000 {
				continue
			}
			if _, infected := ps.infections[planet.GetID()]; infected {
				continue
			}
			// Higher population = more likely to have outbreak
			weight := int(planet.Population / 10000)
			if weight < 1 {
				weight = 1
			}
			for i := 0; i < weight; i++ {
				candidates = append(candidates, planet)
			}
		}
	}

	if len(candidates) == 0 {
		return
	}

	// Max 2 concurrent plagues
	if len(ps.infections) >= 2 {
		return
	}

	planet := candidates[rand.Intn(len(candidates))]
	ps.infections[planet.GetID()] = &Infection{
		PlanetID:  planet.GetID(),
		Severity:  0.005 + rand.Float64()*0.01,
		TicksLeft: 4000 + rand.Intn(3000),
		Source:    "outbreak",
	}

	game.LogEvent("alert", planet.Owner,
		fmt.Sprintf("🦠 PLAGUE OUTBREAK on %s! Population declining. Quarantine trade ships and build Research Labs!",
			planet.Name))
}

// GetInfectedPlanets returns the count of currently infected planets.
func (ps *PlagueSystem) GetInfectedPlanets() int {
	return len(ps.infections)
}

// IsInfected checks if a specific planet has an active plague.
func (ps *PlagueSystem) IsInfected(planetID int) bool {
	_, ok := ps.infections[planetID]
	return ok
}
