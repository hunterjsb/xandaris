package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&SectorControlSystem{
		BaseSystem: NewBaseSystem("SectorControl", 36),
	})
}

// SectorControlSystem grants bonuses to factions that control entire systems.
// "Control" means owning ALL habitable planets in a system.
//
// Bonuses for sector control:
//   - All planets in the system get +10% production bonus
//   - +500 credits per controlled system per interval
//   - System is marked with the controller's faction on the galaxy map
//
// This creates incentive for:
//   - Colonizing ALL planets in a system (not just cherry-picking the best)
//   - Defending your systems from other factions colonizing there
//   - Strategic system choices (small systems are easier to control)
type SectorControlSystem struct {
	*BaseSystem
	controllers map[int]string // systemID → controlling faction name
}

func (scs *SectorControlSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := scs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if scs.controllers == nil {
		scs.controllers = make(map[int]string)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	playerByName := make(map[string]*entities.Player)
	for _, p := range players {
		if p != nil {
			playerByName[p.Name] = p
		}
	}

	for _, sys := range systems {
		controller := scs.checkControl(sys)
		oldController := scs.controllers[sys.ID]

		if controller != oldController {
			if controller != "" {
				game.LogEvent("event", controller,
					fmt.Sprintf("👑 %s now controls %s! All planets get +10%% production bonus",
						controller, sys.Name))
			} else if oldController != "" {
				game.LogEvent("event", oldController,
					fmt.Sprintf("⚠️ %s lost control of %s — another faction colonized there",
						oldController, sys.Name))
			}
			scs.controllers[sys.ID] = controller
		}

		// Apply bonuses
		if controller != "" {
			player := playerByName[controller]
			if player != nil {
				player.Credits += 500 // sector control income
			}

			// Production bonus applied via planet marker
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == controller {
					// Mark planet as in a controlled sector (used by production systems)
					if planet.Specialties == nil {
						planet.Specialties = make(map[string]float64)
					}
					planet.Specialties["sector_control"] = 10.0 // +10% bonus
				}
			}
		} else {
			// Remove sector control bonus
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok {
					if planet.Specialties != nil {
						delete(planet.Specialties, "sector_control")
					}
				}
			}
		}
	}
}

// checkControl returns the faction name that controls the system, or "" if contested.
func (scs *SectorControlSystem) checkControl(sys *entities.System) string {
	var habitablePlanets []*entities.Planet
	for _, e := range sys.Entities {
		if planet, ok := e.(*entities.Planet); ok && planet.IsHabitable() {
			habitablePlanets = append(habitablePlanets, planet)
		}
	}

	if len(habitablePlanets) == 0 {
		return ""
	}

	// Check if all habitable planets are owned by the same faction
	owner := ""
	for _, planet := range habitablePlanets {
		if planet.Owner == "" {
			return "" // unclaimed planet = no control
		}
		if owner == "" {
			owner = planet.Owner
		} else if planet.Owner != owner {
			return "" // different owners = contested
		}
	}

	return owner
}
