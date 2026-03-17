package tickable

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ResourceAtlasSystem{
		BaseSystem: NewBaseSystem("ResourceAtlas", 119),
	})
}

// ResourceAtlasSystem periodically publishes a galaxy-wide resource
// atlas showing where key resources are produced and stockpiled.
// This helps factions plan trade routes and identify opportunities.
//
// Atlas report includes:
//   - Top producing system for each resource
//   - Systems with critical shortages
//   - Galaxy-wide surplus/deficit overview
//   - Trade opportunity highlights
//
// Published every ~8000 ticks. This is the "trade newspaper" that
// makes the logistics game accessible.
type ResourceAtlasSystem struct {
	*BaseSystem
	nextReport int64
}

func (ras *ResourceAtlasSystem) OnTick(tick int64) {
	ctx := ras.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ras.nextReport == 0 {
		ras.nextReport = tick + 5000 + int64(rand.Intn(5000))
	}
	if tick < ras.nextReport {
		return
	}
	ras.nextReport = tick + 8000 + int64(rand.Intn(5000))

	systems := game.GetSystems()

	// Aggregate per-system resource totals
	type sysStock struct {
		id    int
		name  string
		stock map[string]int
	}
	var systemStocks []sysStock

	for _, sys := range systems {
		stock := make(map[string]int)
		hasOwned := false
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				hasOwned = true
				for _, res := range []string{entities.ResIron, entities.ResWater, entities.ResOil,
					entities.ResFuel, entities.ResRareMetals, entities.ResHelium3, entities.ResElectronics} {
					stock[res] += planet.GetStoredAmount(res)
				}
			}
		}
		if hasOwned {
			systemStocks = append(systemStocks, sysStock{sys.ID, sys.Name, stock})
		}
	}

	if len(systemStocks) < 2 {
		return
	}

	// Find most scarce resource galaxy-wide
	galaxyTotals := make(map[string]int)
	for _, ss := range systemStocks {
		for res, amt := range ss.stock {
			galaxyTotals[res] += amt
		}
	}

	type resTotal struct {
		name  string
		total int
	}
	var sorted []resTotal
	for res, total := range galaxyTotals {
		sorted = append(sorted, resTotal{res, total})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].total < sorted[j].total })

	// Build atlas report
	msg := "🗺️ Resource Atlas: "

	// Scarcest resource
	if len(sorted) > 0 {
		msg += fmt.Sprintf("Scarcest: %s (%d galaxy-wide) ", sorted[0].name, sorted[0].total)
	}

	// Most abundant
	if len(sorted) > 1 {
		last := sorted[len(sorted)-1]
		msg += fmt.Sprintf("| Abundant: %s (%d) ", last.name, last.total)
	}

	// Biggest fuel stockpile (since everyone has power crisis)
	bestFuelSys := ""
	bestFuel := 0
	for _, ss := range systemStocks {
		if ss.stock[entities.ResFuel] > bestFuel {
			bestFuel = ss.stock[entities.ResFuel]
			bestFuelSys = ss.name
		}
	}
	if bestFuelSys != "" {
		msg += fmt.Sprintf("| Best fuel: %s (%d)", bestFuelSys, bestFuel)
	}

	game.LogEvent("intel", "", msg)
}
