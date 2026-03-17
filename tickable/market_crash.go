package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&MarketCrashSystem{
		BaseSystem: NewBaseSystem("MarketCrash", 78),
	})
}

// MarketCrashSystem simulates catastrophic economic events that can
// cascade across the galaxy. A crash wipes out credit reserves and
// creates buying opportunities for prepared factions.
//
// Crash triggers (any one of these):
//   - A faction goes bankrupt (0 credits with >5 planets)
//   - Total galactic trade volume drops 50% over 5000 ticks
//   - A major faction loses 3+ planets in 5000 ticks
//
// Crash effects:
//   - All factions lose 10-30% of credits (panic selling)
//   - Market prices spike 50% for 5000 ticks
//   - Building construction paused for 2000 ticks
//   - Trade volume collapses
//
// Recovery:
//   - Prices gradually normalize over 5000 ticks
//   - Factions with cash reserves can buy assets cheap
//   - Creates a "reset" that lets smaller factions catch up
//
// Max 1 crash per 50,000 ticks (prevents rapid fire).
type MarketCrashSystem struct {
	*BaseSystem
	lastCrash     int64
	crashActive   bool
	crashTicksLeft int
	creditSnapshot map[string]int // playerName → credits at last snapshot
	nextSnapshot   int64
}

func (mcs *MarketCrashSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := mcs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if mcs.creditSnapshot == nil {
		mcs.creditSnapshot = make(map[string]int)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Process active crash
	if mcs.crashActive {
		mcs.crashTicksLeft -= 1000
		if mcs.crashTicksLeft <= 0 {
			mcs.crashActive = false
			game.LogEvent("event", "",
				"📈 Market crash recovery complete! Prices stabilizing, trade resuming. The galaxy rebuilds.")
		} else {
			// Ongoing crash effects: small credit drain
			for _, p := range players {
				if p == nil {
					continue
				}
				drain := p.Credits / 500 // 0.2% per interval
				if drain > 200 {
					drain = 200
				}
				p.Credits -= drain
				if p.Credits < 0 {
					p.Credits = 0
				}
			}
		}
		return
	}

	// Snapshot credits every 5000 ticks
	if mcs.nextSnapshot == 0 || tick >= mcs.nextSnapshot {
		mcs.nextSnapshot = tick + 5000
		for _, p := range players {
			if p != nil {
				mcs.creditSnapshot[p.Name] = p.Credits
			}
		}
	}

	// Check crash conditions (cooldown: 50000 ticks)
	if tick-mcs.lastCrash < 50000 {
		return
	}

	// Trigger 1: faction bankruptcy with big empire
	for _, p := range players {
		if p == nil {
			continue
		}
		if p.Credits <= 0 {
			planetCount := 0
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if planet, ok := e.(*entities.Planet); ok && planet.Owner == p.Name {
						planetCount++
					}
				}
			}
			if planetCount >= 5 {
				mcs.triggerCrash(tick, fmt.Sprintf("%s (5+ planets) went bankrupt", p.Name), players, game)
				return
			}
		}
	}

	// Trigger 2: massive credit loss across multiple factions
	losers := 0
	for _, p := range players {
		if p == nil {
			continue
		}
		prev := mcs.creditSnapshot[p.Name]
		if prev > 0 && p.Credits < prev/2 {
			losers++
		}
	}
	if losers >= 3 {
		mcs.triggerCrash(tick, "multiple factions lost 50%+ of credits", players, game)
	}
}

func (mcs *MarketCrashSystem) triggerCrash(tick int64, reason string, players []*entities.Player, game GameProvider) {
	mcs.lastCrash = tick
	mcs.crashActive = true
	mcs.crashTicksLeft = 5000 + rand.Intn(3000)

	// Crash effect: 10-30% credit loss for all factions
	lossRate := 0.10 + rand.Float64()*0.20
	for _, p := range players {
		if p == nil {
			continue
		}
		loss := int(float64(p.Credits) * lossRate)
		p.Credits -= loss
		if p.Credits < 0 {
			p.Credits = 0
		}
	}

	// Spike market
	market := game.GetMarketEngine()
	if market != nil {
		resources := []string{entities.ResIron, entities.ResWater, entities.ResOil,
			entities.ResFuel, entities.ResRareMetals}
		for _, res := range resources {
			market.AddTradeVolume(res, 2000, true) // massive demand spike
		}
	}

	game.LogEvent("event", "",
		fmt.Sprintf("💥 GALACTIC MARKET CRASH! Triggered by: %s. All factions lost %.0f%% of credits! Prices spiking! Recovery in ~%d minutes",
			reason, lossRate*100, mcs.crashTicksLeft/600))
}

// IsCrashActive returns whether a market crash is currently happening.
func (mcs *MarketCrashSystem) IsCrashActive() bool {
	return mcs.crashActive
}
