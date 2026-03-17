package tickable

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradeIntelSystem{
		BaseSystem: NewBaseSystem("TradeIntel", 47),
	})
}

// TradeIntelSystem generates actionable trade intelligence for players.
// Every ~2000 ticks it scans the galaxy and identifies:
//
//   - Price arbitrage: resource cheap in system A, expensive in system B
//   - Supply gaps: systems where a resource is needed but not produced
//   - Demand spikes: planets consuming faster than they produce
//   - Route profitability: which shipping routes would be most profitable
//
// Intel is announced as events so all players (including LLM agents) can act on it.
// This drives emergent trade competition: multiple factions see the same opportunity
// and race to fulfill it.
type TradeIntelSystem struct {
	*BaseSystem
	nextScan int64
}

type tradeOpp struct {
	resource  string
	fromSys   string
	toSys     string
	fromSysID int
	toSysID   int
	surplus   int
	deficit   int
	profit    int // estimated credits per trip
}

func (tis *TradeIntelSystem) OnTick(tick int64) {
	ctx := tis.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tis.nextScan == 0 {
		tis.nextScan = tick + 1000 + int64(rand.Intn(2000))
	}
	if tick < tis.nextScan {
		return
	}
	tis.nextScan = tick + 5000 + int64(rand.Intn(5000))

	systems := game.GetSystems()
	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	// Scan all systems for resource surplus/deficit
	type systemStock struct {
		sysID   int
		sysName string
		stock   map[string]int // resource → total stored
	}
	var systemStocks []systemStock

	for _, sys := range systems {
		stock := make(map[string]int)
		hasOwned := false
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}
			hasOwned = true
			for _, res := range []string{entities.ResIron, entities.ResWater, entities.ResOil,
				entities.ResFuel, entities.ResHelium3,
				entities.ResRareMetals, entities.ResElectronics} {
				stock[res] += planet.GetStoredAmount(res)
			}
		}
		if hasOwned {
			systemStocks = append(systemStocks, systemStock{sys.ID, sys.Name, stock})
		}
	}

	if len(systemStocks) < 2 {
		return
	}

	// Find best arbitrage opportunities
	var opportunities []tradeOpp
	resources := []string{entities.ResIron, entities.ResWater, entities.ResOil,
		entities.ResFuel, entities.ResRareMetals, entities.ResElectronics}

	for _, res := range resources {
		// Find system with most surplus and system with most deficit
		var bestSurplus, worstDeficit *systemStock
		maxSurplus := 0
		minStock := 999999

		for i := range systemStocks {
			ss := &systemStocks[i]
			amt := ss.stock[res]
			if amt > maxSurplus {
				maxSurplus = amt
				bestSurplus = ss
			}
			if amt < minStock {
				minStock = amt
				worstDeficit = ss
			}
		}

		if bestSurplus != nil && worstDeficit != nil &&
			bestSurplus.sysID != worstDeficit.sysID &&
			maxSurplus > 100 && maxSurplus-minStock > 50 {
			price := market.GetSellPrice(res)
			profit := int(price * float64(maxSurplus-minStock) * 0.5)
			opportunities = append(opportunities, tradeOpp{
				resource:  res,
				fromSys:   bestSurplus.sysName,
				toSys:     worstDeficit.sysName,
				fromSysID: bestSurplus.sysID,
				toSysID:   worstDeficit.sysID,
				surplus:   maxSurplus,
				deficit:   minStock,
				profit:    profit,
			})
		}
	}

	if len(opportunities) == 0 {
		return
	}

	// Sort by profit and announce top 2
	sort.Slice(opportunities, func(i, j int) bool {
		return opportunities[i].profit > opportunities[j].profit
	})

	for i, opp := range opportunities {
		if i >= 2 {
			break
		}
		game.LogEvent("intel", "",
			fmt.Sprintf("📊 TRADE INTEL: %s surplus in %s (%d units), deficit in %s (%d units). Est. profit: %d cr/trip",
				opp.resource, opp.fromSys, opp.surplus, opp.toSys, opp.deficit, opp.profit))
	}
}
