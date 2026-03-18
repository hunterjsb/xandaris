package tickable

import (
	"fmt"
	"math"
	"math/rand"
)

func init() {
	RegisterSystem(&CreditVelocitySystem{
		BaseSystem: NewBaseSystem("CreditVelocity", 206),
	})
}

// CreditVelocitySystem measures how fast credits are changing hands
// across the galaxy. High velocity = active economy. Low velocity =
// stagnation (everyone hoarding).
//
// Velocity = sum of absolute credit changes across all factions /
//            total credits in the galaxy
//
// High velocity (>0.3): "Hot economy" — lots of trading, building, spending
// Medium (0.1-0.3): "Healthy" — balanced growth
// Low (<0.1): "Cold economy" — everyone hoarding, not investing
//
// Also detects "flash crashes" — sudden velocity spikes from a single
// faction losing a lot of credits quickly.
type CreditVelocitySystem struct {
	*BaseSystem
	prevCredits map[string]int
	nextReport  int64
}

func (cvs *CreditVelocitySystem) OnTick(tick int64) {
	ctx := cvs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if cvs.prevCredits == nil {
		cvs.prevCredits = make(map[string]int)
	}

	if cvs.nextReport == 0 {
		cvs.nextReport = tick + 5000
	}
	if tick < cvs.nextReport {
		return
	}
	cvs.nextReport = tick + 6000 + int64(rand.Intn(4000))

	players := game.GetPlayers()

	totalCredits := 0
	totalChange := 0
	biggestDrop := 0
	biggestDropper := ""

	for _, p := range players {
		if p == nil {
			continue
		}
		totalCredits += p.Credits

		prev := cvs.prevCredits[p.Name]
		if prev > 0 {
			change := int(math.Abs(float64(p.Credits - prev)))
			totalChange += change

			drop := prev - p.Credits
			if drop > biggestDrop {
				biggestDrop = drop
				biggestDropper = p.Name
			}
		}
		cvs.prevCredits[p.Name] = p.Credits
	}

	if totalCredits <= 0 {
		return
	}

	velocity := float64(totalChange) / float64(totalCredits)

	status := "Cold 🧊"
	if velocity > 0.3 {
		status = "Hot 🔥"
	} else if velocity > 0.1 {
		status = "Healthy 💚"
	}

	game.LogEvent("intel", "",
		fmt.Sprintf("💱 Credit Velocity: %.1f%% (%s) | %dcr changing hands, %dcr total galaxy wealth",
			velocity*100, status, totalChange, totalCredits))

	// Flash crash detection
	if biggestDrop > 200000 {
		game.LogEvent("alert", biggestDropper,
			fmt.Sprintf("💥 FLASH CRASH: %s lost %dcr this period! What happened?",
				biggestDropper, biggestDrop))
	}
}
