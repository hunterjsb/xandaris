package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradeWarSystem{
		BaseSystem: NewBaseSystem("TradeWar", 59),
	})
}

// TradeWarSystem detects when two factions are competing heavily for the
// same resources in the same system and triggers a "trade war" event.
//
// A trade war occurs when:
//   - Two factions are both buying the same resource in the same system
//   - Combined demand exceeds local supply
//   - Neither faction backs down (both keep ordering)
//
// Trade war effects:
//   - Local prices for the contested resource spike 2x
//   - Both factions lose 100 credits per tick in "bidding escalation"
//   - Third-party sellers in the system get double price (war profiteering)
//   - Trade war lasts 3000-5000 ticks unless one faction stops buying
//
// This creates emergent economic conflict without military action.
// The smart move might be to back off and find another source.
type TradeWarSystem struct {
	*BaseSystem
	wars     []*TradeWar
	nextCheck int64
}

// TradeWar represents an active trade conflict between two factions.
type TradeWar struct {
	SystemID   int
	SystemName string
	FactionA   string
	FactionB   string
	Resource   string
	TicksLeft  int
	Active     bool
}

func (tws *TradeWarSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := tws.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tws.nextCheck == 0 {
		tws.nextCheck = tick + 3000 + int64(rand.Intn(5000))
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Decay active wars
	for _, war := range tws.wars {
		if !war.Active {
			continue
		}
		war.TicksLeft -= 500
		if war.TicksLeft <= 0 {
			war.Active = false
			game.LogEvent("event", "",
				fmt.Sprintf("🕊️ Trade war between %s and %s over %s in %s has ended. Prices normalizing",
					war.FactionA, war.FactionB, war.Resource, war.SystemName))
			continue
		}

		// Apply war costs
		for _, p := range players {
			if p == nil {
				continue
			}
			if p.Name == war.FactionA || p.Name == war.FactionB {
				p.Credits -= 100 // bidding escalation cost
				if p.Credits < 0 {
					p.Credits = 0
				}
			}
		}
	}

	// Check for new trade wars
	if tick < tws.nextCheck {
		return
	}
	tws.nextCheck = tick + 5000 + int64(rand.Intn(8000))

	// Don't have too many active wars
	activeCount := 0
	for _, w := range tws.wars {
		if w.Active {
			activeCount++
		}
	}
	if activeCount >= 2 {
		return
	}

	tws.detectTradeWar(game, players, systems)
}

func (tws *TradeWarSystem) detectTradeWar(game GameProvider, players []*entities.Player, systems []*entities.System) {
	// Find systems where multiple factions have planets competing for resources
	for _, sys := range systems {
		factionPlanets := make(map[string][]*entities.Planet)
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				factionPlanets[planet.Owner] = append(factionPlanets[planet.Owner], planet)
			}
		}

		if len(factionPlanets) < 2 {
			continue
		}

		// Check for resource competition
		resources := []string{entities.ResIron, entities.ResWater, entities.ResOil,
			entities.ResFuel, entities.ResRareMetals}

		for _, res := range resources {
			// Find factions with low stock (they're competing to buy)
			var competitors []string
			for faction, planets := range factionPlanets {
				totalStock := 0
				for _, p := range planets {
					totalStock += p.GetStoredAmount(res)
				}
				if totalStock < 30 { // desperate for this resource
					competitors = append(competitors, faction)
				}
			}

			if len(competitors) < 2 {
				continue
			}

			// Check we don't already have a war for this system+resource
			alreadyWar := false
			for _, w := range tws.wars {
				if w.Active && w.SystemID == sys.ID && w.Resource == res {
					alreadyWar = true
					break
				}
			}
			if alreadyWar {
				continue
			}

			// 20% chance to trigger
			if rand.Intn(5) != 0 {
				continue
			}

			war := &TradeWar{
				SystemID:   sys.ID,
				SystemName: sys.Name,
				FactionA:   competitors[0],
				FactionB:   competitors[1],
				Resource:   res,
				TicksLeft:  3000 + rand.Intn(2000),
				Active:     true,
			}
			tws.wars = append(tws.wars, war)

			game.LogEvent("event", "",
				fmt.Sprintf("⚔️ TRADE WAR! %s and %s are in a bidding war over %s in %s! Prices spiking, costs rising for both sides",
					competitors[0], competitors[1], res, sys.Name))
			return // one new war per tick
		}
	}
}

// GetActiveWars returns currently active trade wars.
func (tws *TradeWarSystem) GetActiveWars() []*TradeWar {
	var result []*TradeWar
	for _, w := range tws.wars {
		if w.Active {
			result = append(result, w)
		}
	}
	return result
}
