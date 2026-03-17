package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalacticGDPSystem{
		BaseSystem: NewBaseSystem("GalacticGDP", 116),
	})
}

// GalacticGDPSystem computes the total economic output of the galaxy
// and each faction's share of it. This is the ultimate economic
// scoreboard — not just credits, but total productive capacity.
//
// GDP components per faction:
//   - Credit income per interval (from all sources)
//   - Resource production value (units * market price)
//   - Trade volume (buy + sell value)
//   - Population productivity (pop * happiness * tech)
//   - Infrastructure value (buildings * level)
//
// Announced as faction GDP share: "Gemini 35%, Opus 25%, Llama 15%..."
// Also tracks GDP growth rate: expanding vs contracting economies.
type GalacticGDPSystem struct {
	*BaseSystem
	prevGDP    map[string]int
	nextReport int64
}

func (ggdp *GalacticGDPSystem) OnTick(tick int64) {
	if tick%5000 != 0 {
		return
	}

	ctx := ggdp.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ggdp.prevGDP == nil {
		ggdp.prevGDP = make(map[string]int)
	}

	if ggdp.nextReport == 0 {
		ggdp.nextReport = tick + 5000
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	market := game.GetMarketEngine()

	// Calculate GDP per faction
	factionGDP := make(map[string]int)
	totalGDP := 0

	for _, p := range players {
		if p == nil {
			continue
		}

		gdp := 0

		// Credits (liquid wealth)
		gdp += p.Credits / 10

		// Ships (capital assets)
		gdp += len(p.OwnedShips) * 50

		// Planet productivity
		for _, sys := range systems {
			for _, e := range sys.Entities {
				planet, ok := e.(*entities.Planet)
				if !ok || planet.Owner != p.Name {
					continue
				}

				// Population output
				gdp += int(float64(planet.Population) * planet.Happiness * planet.TechLevel)

				// Resource value
				if market != nil {
					for _, res := range []string{entities.ResIron, entities.ResWater, entities.ResOil,
						entities.ResFuel, entities.ResRareMetals, entities.ResHelium3, entities.ResElectronics} {
						stored := planet.GetStoredAmount(res)
						price := market.GetSellPrice(res)
						gdp += int(float64(stored) * price / 10)
					}
				}

				// Building value
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.IsOperational {
						gdp += b.Level * 100
					}
				}
			}
		}

		factionGDP[p.Name] = gdp
		totalGDP += gdp
	}

	// Report
	if tick < ggdp.nextReport {
		ggdp.prevGDP = factionGDP
		return
	}
	ggdp.nextReport = tick + 8000 + int64(rand.Intn(5000))

	if totalGDP == 0 {
		return
	}

	msg := fmt.Sprintf("📊 Galactic GDP: %d total | ", totalGDP)

	// Top 3 by GDP share
	type gdpEntry struct {
		name   string
		gdp    int
		growth string
	}
	var entries []gdpEntry
	for name, gdp := range factionGDP {
		growth := "→"
		if prev, ok := ggdp.prevGDP[name]; ok {
			if gdp > prev+prev/20 {
				growth = "📈"
			} else if gdp < prev-prev/20 {
				growth = "📉"
			}
		}
		entries = append(entries, gdpEntry{name, gdp, growth})
	}

	// Sort by GDP desc (simple)
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].gdp > entries[i].gdp {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	for i, e := range entries {
		if i >= 4 {
			break
		}
		share := float64(e.gdp) / float64(totalGDP) * 100
		msg += fmt.Sprintf("%s %.0f%% %s ", e.name, share, e.growth)
	}

	game.LogEvent("intel", "", msg)
	ggdp.prevGDP = factionGDP
}
