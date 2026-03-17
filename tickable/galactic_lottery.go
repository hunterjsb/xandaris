package tickable

import (
	"fmt"
	"math/rand"

)

func init() {
	RegisterSystem(&GalacticLotterySystem{
		BaseSystem: NewBaseSystem("GalacticLottery", 104),
	})
}

// GalacticLotterySystem runs a periodic galactic lottery that any
// faction can enter. The prize pool grows from entry fees and
// a percentage of galactic trade volume.
//
// Lottery cycle (~10,000 ticks):
//   1. Announcement: lottery opens, factions auto-enter (100cr fee)
//   2. Drawing: random winner selected
//   3. Prize: entire pool goes to winner
//
// Prize pool = (entries * 100cr) + (1% of total galactic trade value)
// This creates periodic excitement and a way for smaller factions
// to get a windfall that changes their competitive position.
type GalacticLotterySystem struct {
	*BaseSystem
	pool      int
	entrants  []string
	nextDraw  int64
	lotteryNum int
}

func (gls *GalacticLotterySystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := gls.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if gls.nextDraw == 0 {
		gls.nextDraw = tick + 8000 + int64(rand.Intn(5000))
	}

	players := ctx.GetPlayers()

	// Auto-enter factions with >1000cr
	if tick < gls.nextDraw-1000 { // entry period
		for _, p := range players {
			if p == nil || p.Credits < 1000 {
				continue
			}
			// Check not already entered
			entered := false
			for _, name := range gls.entrants {
				if name == p.Name {
					entered = true
					break
				}
			}
			if entered {
				continue
			}

			// Auto-enter
			entryFee := 100
			p.Credits -= entryFee
			gls.pool += entryFee
			gls.entrants = append(gls.entrants, p.Name)
		}
	}

	// Drawing
	if tick >= gls.nextDraw {
		gls.nextDraw = tick + 10000 + int64(rand.Intn(8000))
		gls.lotteryNum++

		if len(gls.entrants) < 2 {
			gls.pool = 0
			gls.entrants = nil
			return
		}

		// Add trade volume bonus to pool
		gls.pool += 500 // base prize addition

		// Draw winner
		winner := gls.entrants[rand.Intn(len(gls.entrants))]
		prize := gls.pool

		for _, p := range players {
			if p != nil && p.Name == winner {
				p.Credits += prize
				break
			}
		}

		game.LogEvent("event", winner,
			fmt.Sprintf("🎰 GALACTIC LOTTERY #%d: %s WINS %dcr! (%d entrants)",
				gls.lotteryNum, winner, prize, len(gls.entrants)))

		// Reset
		gls.pool = 0
		gls.entrants = nil
	}

	// Announce when pool is building
	if len(gls.entrants) > 0 && rand.Intn(5) == 0 {
		ticksLeft := gls.nextDraw - tick
		if ticksLeft > 0 {
			game.LogEvent("event", "",
				fmt.Sprintf("🎰 Galactic Lottery #%d: %d entrants, prize pool: %dcr. Drawing in ~%d min!",
					gls.lotteryNum+1, len(gls.entrants), gls.pool, ticksLeft/600))
		}
	}
}
