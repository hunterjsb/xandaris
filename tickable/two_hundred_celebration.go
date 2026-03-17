package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&TwoHundredCelebrationSystem{
		BaseSystem: NewBaseSystem("Celebration200", 200),
	})
}

// TwoHundredCelebrationSystem is the 200th tickable system in Xandaris.
// It exists to celebrate the absurd depth of this game by periodically
// reminding the galaxy how many systems are running simultaneously.
//
// Every ~15,000 ticks it announces a fun fact about the game's systems.
type TwoHundredCelebrationSystem struct {
	*BaseSystem
	nextFact int64
	factIdx  int
}

var funFacts = []string{
	"This galaxy runs on 200 simultaneous game systems. That's more than most AAA 4X games ship with.",
	"From bankruptcy protection to extinction events, from ship naming to galactic lotteries — 200 systems shape your destiny.",
	"200 systems: economy, military, diplomacy, exploration, weather, seasons, futures markets, stock exchanges, and more.",
	"Every 10 ticks, 200 systems evaluate the galaxy. That's 20 system-ticks per game-tick. The void is busy.",
	"Systems #1-50 handle the basics. Systems #51-100 add depth. Systems #100-150 create narrative. Systems #150-200? Pure madness.",
}

func (tcs *TwoHundredCelebrationSystem) OnTick(tick int64) {
	ctx := tcs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if tcs.nextFact == 0 {
		tcs.nextFact = tick + 10000 + int64(rand.Intn(10000))
	}
	if tick < tcs.nextFact {
		return
	}
	tcs.nextFact = tick + 15000 + int64(rand.Intn(15000))

	fact := funFacts[tcs.factIdx%len(funFacts)]
	tcs.factIdx++

	game.LogEvent("intel", "",
		fmt.Sprintf("🎮 %s", fact))
}
