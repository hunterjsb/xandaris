package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AlertSystem{
		BaseSystem:  NewBaseSystem("Alerts", 55),
		lastAlerted: make(map[string]int64),
	})
}

// AlertSystem generates warnings when player planets have critical issues.
// Fires into the event log so the command bar, spectator, and API all see them.
type AlertSystem struct {
	*BaseSystem
	lastAlerted map[string]int64 // "player:alert_type" -> last tick alerted (prevents spam)
}

const alertCooldown = 500 // ticks between repeated alerts (~50 seconds)

func (as *AlertSystem) OnTick(tick int64) {
	if tick%50 != 0 {
		return
	}

	ctx := as.GetContext()
	if ctx == nil {
		return
	}

	playersIface := ctx.GetPlayers()
	if playersIface == nil {
		return
	}
	players, ok := playersIface.([]*entities.Player)
	if !ok {
		return
	}

	logger, _ := ctx.GetGame().(EventLogger)
	if logger == nil {
		return
	}

	for _, player := range players {
		if player == nil {
			continue
		}
		as.checkAlerts(player, tick, logger)
	}
}

func (as *AlertSystem) checkAlerts(player *entities.Player, tick int64, logger EventLogger) {
	for _, planet := range player.OwnedPlanets {
		if planet == nil {
			continue
		}

		// Alert: Water critically low (< 20 units with population > 500)
		water := planet.GetStoredAmount("Water")
		if water < 20 && planet.Population > 500 {
			as.alert(player.Name, "water_critical", tick, logger,
				fmt.Sprintf("Water critical on %s! (%d units, %d pop)", planet.Name, water, planet.Population))
		}

		// Alert: Fuel depleted (0 units, has buildings that need it)
		fuel := planet.GetStoredAmount("Fuel")
		if fuel == 0 {
			hasFuelBuilding := false
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok {
					if b.BuildingType == "Base" || b.BuildingType == "Habitat" || b.BuildingType == "Shipyard" {
						hasFuelBuilding = true
						break
					}
				}
			}
			if hasFuelBuilding {
				as.alert(player.Name, "fuel_depleted", tick, logger,
					fmt.Sprintf("Fuel depleted on %s! Buildings will shut down", planet.Name))
			}
		}

		// Alert: Happiness very low (< 25%)
		if planet.Happiness < 0.25 && planet.Population > 1000 {
			as.alert(player.Name, "unhappy", tick, logger,
				fmt.Sprintf("%s is miserable (%.0f%% happy) — productivity at %.1fx",
					planet.Name, planet.Happiness*100, planet.ProductivityBonus))
		}

		// Alert: Storage nearly full (> 90% on any resource)
		for resType, s := range planet.StoredResources {
			if s != nil && s.Capacity > 0 && s.Amount > 0 {
				pct := float64(s.Amount) / float64(s.Capacity)
				if pct > 0.90 {
					as.alert(player.Name, "storage_full_"+resType, tick, logger,
						fmt.Sprintf("%s storage full on %s (%d/%d) — sell or expand!",
							resType, planet.Name, s.Amount, s.Capacity))
				}
			}
		}

		// Alert: Bankruptcy imminent (credits < 100 with buildings running)
		if player.Credits < 100 && len(planet.Buildings) > 1 {
			as.alert(player.Name, "bankruptcy", tick, logger,
				fmt.Sprintf("%s low on credits (%d) — buildings may shut down!", player.Name, player.Credits))
		}
	}
}

func (as *AlertSystem) alert(player, alertType string, tick int64, logger EventLogger, message string) {
	key := player + ":" + alertType
	if last, ok := as.lastAlerted[key]; ok && tick-last < alertCooldown {
		return // Still in cooldown
	}
	as.lastAlerted[key] = tick
	logger.LogEvent("alert", player, message)
}
