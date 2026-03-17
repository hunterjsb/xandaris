package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AnomalySystem{
		BaseSystem: NewBaseSystem("Anomalies", 41),
	})
}

// AnomalySystem assigns persistent anomalies to systems that create
// unique strategic properties. Anomalies are discovered by scouts or
// assigned randomly at game start.
//
// Anomaly types:
//   Nebula:        -20% ship speed but +30% mine output (mineral-rich dust)
//   Radiation Belt: -10% population growth but +50% Electronics output
//   Gravity Well:  +20% fuel consumption but +25% ship defense
//   Dark Matter:   +40% research speed (tech level growth)
//   Trade Winds:   +30% cargo capacity for ships in this system
//   Void Rift:     -25% habitability but +100% Rare Metals abundance
//
// Anomalies make system choice strategic — not all systems are equal.
type AnomalySystem struct {
	*BaseSystem
	anomalies map[int]string // systemID → anomaly type
	assigned  bool
}

var anomalyTypes = []string{
	"Nebula", "Radiation Belt", "Gravity Well",
	"Dark Matter", "Trade Winds", "Void Rift",
}

var anomalyEffects = map[string]string{
	"Nebula":         "+30% mine output, -20% ship speed",
	"Radiation Belt": "+50% Electronics output, -10% pop growth",
	"Gravity Well":   "+25% ship defense, +20% fuel consumption",
	"Dark Matter":    "+40% research/tech speed",
	"Trade Winds":    "+30% cargo capacity in system",
	"Void Rift":      "+100% Rare Metals, -25% habitability",
}

func (as *AnomalySystem) OnTick(tick int64) {
	if as.assigned {
		// Anomalies are persistent — only apply effects periodically
		if tick%500 != 0 {
			return
		}
		as.applyEffects()
		return
	}

	// One-time assignment: ~30% of systems get an anomaly
	ctx := as.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	as.anomalies = make(map[int]string)
	systems := game.GetSystems()

	for _, sys := range systems {
		if rand.Intn(100) < 30 { // 30% chance
			anomaly := anomalyTypes[rand.Intn(len(anomalyTypes))]
			as.anomalies[sys.ID] = anomaly
			game.LogEvent("event", "",
				fmt.Sprintf("🔮 Anomaly detected in %s: %s (%s)",
					sys.Name, anomaly, anomalyEffects[anomaly]))
		}
	}

	as.assigned = true
	fmt.Printf("[Anomalies] Assigned %d anomalies across %d systems\n",
		len(as.anomalies), len(systems))
}

func (as *AnomalySystem) applyEffects() {
	ctx := as.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	for _, sys := range game.GetSystems() {
		anomaly, has := as.anomalies[sys.ID]
		if !has {
			continue
		}

		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			if planet.Specialties == nil {
				planet.Specialties = make(map[string]float64)
			}

			switch anomaly {
			case "Nebula":
				planet.Specialties["anomaly_mining"] = 30.0 // +30% mine output
			case "Radiation Belt":
				planet.Specialties["anomaly_electronics"] = 50.0
			case "Dark Matter":
				planet.Specialties["anomaly_research"] = 40.0
			case "Trade Winds":
				planet.Specialties["anomaly_trade"] = 30.0
			case "Void Rift":
				planet.Specialties["anomaly_rare_metals"] = 100.0
			}
		}
	}
}

// GetAnomaly returns the anomaly for a system, if any.
func (as *AnomalySystem) GetAnomaly(systemID int) (string, string) {
	if as.anomalies == nil {
		return "", ""
	}
	if anomaly, has := as.anomalies[systemID]; has {
		return anomaly, anomalyEffects[anomaly]
	}
	return "", ""
}
