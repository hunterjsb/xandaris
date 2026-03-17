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

const alertCooldown = 3000 // ticks between repeated alerts (~5 minutes, was 500)

func (as *AlertSystem) OnTick(tick int64) {
	if tick%50 != 0 {
		return
	}

	ctx := as.GetContext()
	if ctx == nil {
		return
	}

	logger := ctx.GetGame()
	if logger == nil {
		return
	}

	players := ctx.GetPlayers()
	for _, player := range players {
		if player == nil {
			continue
		}
		as.checkPlayerAlerts(player, tick, logger)
	}

	// Planet-specific alerts from system entities (authoritative)
	for _, sys := range logger.GetSystems() {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				as.checkPlanetAlerts(planet, tick, logger)
			}
		}
	}
}

// checkPlayerAlerts checks player-level alerts (credits, etc.)
func (as *AlertSystem) checkPlayerAlerts(player *entities.Player, tick int64, logger GameProvider) {
	if player.Credits < 100 {
		hasBuildingsRunning := false
		for _, planet := range player.OwnedPlanets {
			if planet != nil && len(planet.Buildings) > 1 {
				hasBuildingsRunning = true
				break
			}
		}
		if hasBuildingsRunning {
			as.alert(player.Name, "bankruptcy", tick, logger,
				fmt.Sprintf("%s low on credits (%d) — buildings may shut down!", player.Name, player.Credits))
		}
	}
}

// checkPlanetAlerts checks planet-level alerts using system entity data (authoritative).
func (as *AlertSystem) checkPlanetAlerts(planet *entities.Planet, tick int64, logger GameProvider) {
	owner := planet.Owner

	// Alert: Water critically low (< 20 units with population > 500)
	water := planet.GetStoredAmount(entities.ResWater)
	if water < 20 && planet.Population > 500 {
		as.alert(owner, "water_critical", tick, logger,
			fmt.Sprintf("Water critical on %s! (%d units, %d pop)", planet.Name, water, planet.Population))
	}

	// Alert: Fuel depleted (0 units, has buildings that need it)
	fuel := planet.GetStoredAmount(entities.ResFuel)
	if fuel == 0 {
		hasFuelBuilding := false
		for _, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok {
				if b.BuildingType == entities.BuildingBase || b.BuildingType == entities.BuildingHabitat || b.BuildingType == entities.BuildingShipyard {
					hasFuelBuilding = true
					break
				}
			}
		}
		if hasFuelBuilding {
			as.alert(owner, "fuel_depleted", tick, logger,
				fmt.Sprintf("Fuel depleted on %s! Buildings will shut down", planet.Name))
		}
	}

	// Alert: Happiness very low (< 25%)
	if planet.Happiness < 0.25 && planet.Population > 1000 {
		as.alert(owner, "unhappy", tick, logger,
			fmt.Sprintf("%s is miserable (%.0f%% happy) — productivity at %.1fx",
				planet.Name, planet.Happiness*100, planet.ProductivityBonus))
	}

	// Alert: Storage nearly full (> 90% on any resource)
	for resType, s := range planet.StoredResources {
		if s != nil && s.Capacity > 0 && s.Amount > 0 {
			pct := float64(s.Amount) / float64(s.Capacity)
			if pct > 0.90 {
				as.alert(owner, "storage_full_"+resType, tick, logger,
					fmt.Sprintf("%s storage full on %s (%d/%d) — sell or expand!",
						resType, planet.Name, s.Amount, s.Capacity))
			}
		}
	}

	// Alert: Power crisis (< 50% power with buildings needing it)
	if planet.PowerConsumed > 0 && planet.GetPowerRatio() < 0.5 && planet.Population > 500 {
		as.alert(owner, "power_crisis_"+fmt.Sprintf("%d", planet.GetID()), tick, logger,
			fmt.Sprintf("Power crisis on %s! %.0f%% capacity — production severely impacted",
				planet.Name, planet.GetPowerRatio()*100))
	}

	// Alert: Tech stagnation (decaying tech with no Electronics)
	if planet.TechLevel > 0.5 && planet.GetStoredAmount(entities.ResElectronics) == 0 && planet.Population > 2000 {
		as.alert(owner, "tech_decay_"+fmt.Sprintf("%d", planet.GetID()), tick, logger,
			fmt.Sprintf("Tech declining on %s (%.1f) — no Electronics supply!",
				planet.Name, planet.TechLevel))
	}
}

func (as *AlertSystem) alert(player, alertType string, tick int64, logger GameProvider, message string) {
	key := player + ":" + alertType
	if last, ok := as.lastAlerted[key]; ok && tick-last < alertCooldown {
		return // Still in cooldown
	}
	as.lastAlerted[key] = tick
	logger.LogEvent("alert", player, message)
}
