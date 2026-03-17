package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&PriceHistorySystem{
		BaseSystem: NewBaseSystem("PriceHistory", 70),
	})
}

// PriceHistorySystem tracks resource price trends and generates
// market analysis events. This gives factions (especially LLM agents)
// data to make better trading decisions.
//
// Every ~3000 ticks, it records current prices and compares to previous.
// Announces:
//   - Trending up: price increased >10% since last snapshot
//   - Trending down: price decreased >10% since last snapshot
//   - Stable: within ±10%
//   - Best buy: cheapest resource relative to historical average
//   - Best sell: most expensive resource relative to historical average
//
// This replaces blind trading with informed decision-making.
// LLM agents can read these events to adjust strategy.
type PriceHistorySystem struct {
	*BaseSystem
	history    map[string][]float64 // resource → price history (last 10 snapshots)
	nextReport int64
}

func (phs *PriceHistorySystem) OnTick(tick int64) {
	ctx := phs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	if phs.history == nil {
		phs.history = make(map[string][]float64)
	}

	if phs.nextReport == 0 {
		phs.nextReport = tick + 2000 + int64(rand.Intn(2000))
	}

	if tick < phs.nextReport {
		return
	}
	phs.nextReport = tick + 3000 + int64(rand.Intn(3000))

	resources := []string{entities.ResIron, entities.ResWater, entities.ResOil,
		entities.ResFuel, entities.ResRareMetals, entities.ResHelium3, entities.ResElectronics}

	// Record current prices
	for _, res := range resources {
		price := market.GetSellPrice(res)
		if phs.history[res] == nil {
			phs.history[res] = []float64{}
		}
		phs.history[res] = append(phs.history[res], price)
		// Keep last 10 snapshots
		if len(phs.history[res]) > 10 {
			phs.history[res] = phs.history[res][1:]
		}
	}

	// Generate market report
	phs.generateReport(resources, game)
}

func (phs *PriceHistorySystem) generateReport(resources []string, game GameProvider) {
	type trend struct {
		resource  string
		current   float64
		previous  float64
		change    float64 // percentage
		direction string  // "up", "down", "stable"
	}

	var trends []trend
	var bestBuy, bestSell trend
	bestBuyRatio := 999.0
	bestSellRatio := 0.0

	for _, res := range resources {
		history := phs.history[res]
		if len(history) < 2 {
			continue
		}

		current := history[len(history)-1]
		previous := history[len(history)-2]

		changePct := 0.0
		if previous > 0 {
			changePct = (current - previous) / previous * 100
		}

		dir := "stable"
		if changePct > 10 {
			dir = "up"
		} else if changePct < -10 {
			dir = "down"
		}

		t := trend{res, current, previous, changePct, dir}
		trends = append(trends, t)

		// Calculate average for best buy/sell
		avg := 0.0
		for _, p := range history {
			avg += p
		}
		avg /= float64(len(history))

		ratio := current / avg
		if ratio < bestBuyRatio {
			bestBuyRatio = ratio
			bestBuy = t
		}
		if ratio > bestSellRatio {
			bestSellRatio = ratio
			bestSell = t
		}
	}

	if len(trends) == 0 {
		return
	}

	// Build report
	msg := "📊 Market Report: "
	movers := 0
	for _, t := range trends {
		if t.direction != "stable" {
			emoji := "📈"
			if t.direction == "down" {
				emoji = "📉"
			}
			msg += fmt.Sprintf("%s %s %+.0f%% ", emoji, t.resource, t.change)
			movers++
			if movers >= 3 {
				break
			}
		}
	}

	if movers == 0 {
		msg += "All prices stable. "
	}

	if bestBuy.resource != "" && bestBuyRatio < 0.9 {
		msg += fmt.Sprintf("| Best buy: %s (%.0f%% below avg)", bestBuy.resource, (1-bestBuyRatio)*100)
	}
	if bestSell.resource != "" && bestSellRatio > 1.1 {
		msg += fmt.Sprintf("| Best sell: %s (%.0f%% above avg)", bestSell.resource, (bestSellRatio-1)*100)
	}

	game.LogEvent("intel", "", msg)
}

// GetPriceHistory returns price history for a resource.
func (phs *PriceHistorySystem) GetPriceHistory(resource string) []float64 {
	if phs.history == nil {
		return nil
	}
	return phs.history[resource]
}
