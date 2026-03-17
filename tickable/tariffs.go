package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TariffSystem{
		BaseSystem: NewBaseSystem("Tariffs", 27),
	})
}

// TariffSystem lets factions set import tariffs on their systems.
// When a cargo ship sells at a dock in your system, you collect a tariff
// on top of the normal docking fee. This creates trade policy as gameplay.
//
// Tariffs are set per-resource per-system:
//   - 0% (free trade): encourages imports, grows your economy
//   - 10-25% (moderate): balanced revenue without discouraging trade
//   - 50%+ (protectionist): discourages foreign goods, protects local industry
//
// By default all tariffs are 0%. Factions set them via API/agent tool.
// Tariff revenue goes to the system controller (faction owning most planets).
//
// This system also implements export tariffs: when YOUR cargo ships take
// resources FROM your planets to foreign systems, you can tax the export.
// This prevents resource drain from over-trading.
//
// Auto-tariff: if a resource's local stock drops below 50, an emergency
// 25% export tariff is automatically applied to prevent depletion.
type TariffSystem struct {
	*BaseSystem
	importTariffs map[int]map[string]float64 // systemID → resource → tariff rate
	exportTariffs map[int]map[string]float64 // systemID → resource → tariff rate
	autoTariffLog map[int]int64              // systemID → last auto-tariff tick
}

func (ts *TariffSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := ts.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ts.importTariffs == nil {
		ts.importTariffs = make(map[int]map[string]float64)
		ts.exportTariffs = make(map[int]map[string]float64)
		ts.autoTariffLog = make(map[int]int64)
	}

	systems := game.GetSystems()

	// Auto-tariff: protect systems with critically low resources
	for _, sys := range systems {
		ts.evaluateAutoTariffs(tick, sys, game)
	}
}

func (ts *TariffSystem) evaluateAutoTariffs(tick int64, sys *entities.System, game GameProvider) {
	// Find the system controller (faction with most planets)
	ownerCount := make(map[string]int)
	for _, e := range sys.Entities {
		if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
			ownerCount[planet.Owner]++
		}
	}
	if len(ownerCount) == 0 {
		return
	}

	controller := ""
	maxPlanets := 0
	for name, count := range ownerCount {
		if count > maxPlanets {
			maxPlanets = count
			controller = name
		}
	}

	// Check each resource on the controller's planets
	resources := []string{entities.ResIron, entities.ResWater, entities.ResOil,
		entities.ResFuel, entities.ResRareMetals}

	for _, res := range resources {
		totalStock := 0
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner == controller {
				totalStock += planet.GetStoredAmount(res)
			}
		}

		// Auto export tariff if stock critically low
		if totalStock < 50 {
			if ts.exportTariffs[sys.ID] == nil {
				ts.exportTariffs[sys.ID] = make(map[string]float64)
			}
			if ts.exportTariffs[sys.ID][res] < 0.25 {
				ts.exportTariffs[sys.ID][res] = 0.25

				// Rate-limit announcements
				if tick-ts.autoTariffLog[sys.ID] > 5000 {
					ts.autoTariffLog[sys.ID] = tick
					game.LogEvent("trade", controller,
						fmt.Sprintf("📜 Auto-tariff: 25%% export tariff on %s in %s (stock critically low: %d)",
							res, sys.Name, totalStock))
				}
			}
		} else if totalStock > 200 {
			// Lift auto-tariff when stock recovers
			if ts.exportTariffs[sys.ID] != nil {
				delete(ts.exportTariffs[sys.ID], res)
			}
		}
	}
}

// GetImportTariff returns the import tariff rate for a resource in a system.
func (ts *TariffSystem) GetImportTariff(systemID int, resource string) float64 {
	if ts.importTariffs == nil {
		return 0
	}
	if m, ok := ts.importTariffs[systemID]; ok {
		return m[resource]
	}
	return 0
}

// GetExportTariff returns the export tariff rate for a resource in a system.
func (ts *TariffSystem) GetExportTariff(systemID int, resource string) float64 {
	if ts.exportTariffs == nil {
		return 0
	}
	if m, ok := ts.exportTariffs[systemID]; ok {
		return m[resource]
	}
	return 0
}

// SetImportTariff sets a faction's import tariff (called via API).
func (ts *TariffSystem) SetImportTariff(systemID int, resource string, rate float64) {
	if ts.importTariffs == nil {
		ts.importTariffs = make(map[int]map[string]float64)
	}
	if ts.importTariffs[systemID] == nil {
		ts.importTariffs[systemID] = make(map[string]float64)
	}
	if rate <= 0 {
		delete(ts.importTariffs[systemID], resource)
	} else {
		if rate > 1.0 {
			rate = 1.0 // cap at 100%
		}
		ts.importTariffs[systemID][resource] = rate
	}
}

// SetExportTariff sets a faction's export tariff.
func (ts *TariffSystem) SetExportTariff(systemID int, resource string, rate float64) {
	if ts.exportTariffs == nil {
		ts.exportTariffs = make(map[int]map[string]float64)
	}
	if ts.exportTariffs[systemID] == nil {
		ts.exportTariffs[systemID] = make(map[string]float64)
	}
	if rate <= 0 {
		delete(ts.exportTariffs[systemID], resource)
	} else {
		if rate > 1.0 {
			rate = 1.0
		}
		ts.exportTariffs[systemID][resource] = rate
	}
}

// GenerateTariffRevenue is called periodically to distribute tariff income.
// Returns the total tariff revenue generated for a faction.
func (ts *TariffSystem) GenerateTariffRevenue(systemID int, controller string, tradeValue int) int {
	if ts.importTariffs == nil {
		return 0
	}
	m, ok := ts.importTariffs[systemID]
	if !ok {
		return 0
	}
	totalRate := 0.0
	for _, rate := range m {
		totalRate += rate
	}
	avgRate := totalRate / float64(len(m))
	return int(float64(tradeValue) * avgRate)
}

// init auto-generates moderate tariffs for AI factions
func generateRandomTariff() float64 {
	return 0.05 + rand.Float64()*0.15 // 5-20%
}

// Unused but keeping for API integration
var _ = generateRandomTariff
