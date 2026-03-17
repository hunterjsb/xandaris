package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&TradeSanctionsSystem{
		BaseSystem: NewBaseSystem("TradeSanctions", 40),
	})
}

// TradeSanctionsSystem implements galactic trade sanctions against
// aggressive factions. When a faction conquers a planet or destroys
// ships, the galaxy may impose trade sanctions.
//
// Sanctions effects:
//   - Local exchange trades with sanctioned faction reduced by 75%
//   - Black market prices increased 2x for sanctioned faction
//   - Other factions get a "sanctions enforcer" credit bonus
//   - Sanctions last 10,000-20,000 ticks
//
// Sanctions are triggered by:
//   - Conquering 2+ planets in 5000 ticks (territorial aggression)
//   - Destroying 5+ ships in 5000 ticks (military aggression)
//   - Blockading 2+ systems simultaneously
//
// This creates consequences for pure military play: you can conquer
// everything, but your economy suffers from isolation.
type TradeSanctionsSystem struct {
	*BaseSystem
	sanctions   map[string]*Sanction // playerName → active sanction
	aggression  map[string]int       // playerName → aggression score
	nextDecay   int64
}

// Sanction represents active trade sanctions against a faction.
type Sanction struct {
	Target    string
	Reason    string
	TicksLeft int
	Severity  float64 // 0.25-1.0 trade reduction
}

func (tss *TradeSanctionsSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := tss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tss.sanctions == nil {
		tss.sanctions = make(map[string]*Sanction)
		tss.aggression = make(map[string]int)
	}

	players := ctx.GetPlayers()

	// Decay existing sanctions
	for name, sanction := range tss.sanctions {
		sanction.TicksLeft -= 1000
		if sanction.TicksLeft <= 0 {
			delete(tss.sanctions, name)
			game.LogEvent("event", name,
				fmt.Sprintf("✅ Trade sanctions against %s have been lifted. Welcome back to galactic commerce!",
					name))
		}
	}

	// Decay aggression scores
	if tss.nextDecay == 0 || tick >= tss.nextDecay {
		tss.nextDecay = tick + 5000
		for name, score := range tss.aggression {
			if score > 0 {
				tss.aggression[name] = score / 2 // halve every 5000 ticks
			}
			if tss.aggression[name] <= 0 {
				delete(tss.aggression, name)
			}
		}
	}

	// Check for faction aggression that triggers sanctions
	for _, player := range players {
		if player == nil {
			continue
		}
		if _, sanctioned := tss.sanctions[player.Name]; sanctioned {
			continue // already sanctioned
		}

		score := tss.aggression[player.Name]
		if score >= 10 {
			// Trigger sanctions
			severity := 0.50
			if score >= 20 {
				severity = 0.75
			}

			tss.sanctions[player.Name] = &Sanction{
				Target:    player.Name,
				Reason:    "galactic aggression",
				TicksLeft: 10000 + rand.Intn(10000),
				Severity:  severity,
			}
			tss.aggression[player.Name] = 0

			game.LogEvent("event", "",
				fmt.Sprintf("🚫 TRADE SANCTIONS imposed on %s for galactic aggression! Trade reduced by %.0f%% until sanctions expire",
					player.Name, severity*100))
		}
	}
}

// RecordAggression adds to a faction's aggression score.
// Called by combat, siege, and conquest systems.
func (tss *TradeSanctionsSystem) RecordAggression(playerName string, points int) {
	if tss.aggression == nil {
		tss.aggression = make(map[string]int)
	}
	tss.aggression[playerName] += points
}

// GetTradeMultiplier returns the trade effectiveness for a sanctioned faction.
// Returns 1.0 for unsanctioned factions, lower for sanctioned.
func (tss *TradeSanctionsSystem) GetTradeMultiplier(playerName string) float64 {
	if tss.sanctions == nil {
		return 1.0
	}
	sanction, exists := tss.sanctions[playerName]
	if !exists {
		return 1.0
	}
	return 1.0 - sanction.Severity
}

// IsSanctioned returns whether a faction is currently under sanctions.
func (tss *TradeSanctionsSystem) IsSanctioned(playerName string) bool {
	if tss.sanctions == nil {
		return false
	}
	_, exists := tss.sanctions[playerName]
	return exists
}

// GetSanctions returns all active sanctions.
func (tss *TradeSanctionsSystem) GetSanctions() []*Sanction {
	var result []*Sanction
	for _, s := range tss.sanctions {
		result = append(result, s)
	}
	return result
}
