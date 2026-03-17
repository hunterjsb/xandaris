package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GalaxyChronicleSystem{
		BaseSystem: NewBaseSystem("GalaxyChronicle", 124),
	})
}

// GalaxyChronicleSystem generates narrative "history entries" that
// document the galaxy's story. Unlike news/stats systems that report
// current state, the chronicle tells the STORY of what happened.
//
// Chronicle entries:
//   "In the 23rd minute, Llama Logistics rose from near-bankruptcy
//    to claim the #1 position, a stunning reversal of fortune."
//   "The Great Power Crisis of Minute 20 plunged the galaxy into
//    darkness before new solar technology saved civilization."
//
// Triggers on significant state changes:
//   - Rank changes (#1 position swap)
//   - Economic milestones (total galactic credits passing thresholds)
//   - Major events (first war, first alliance, first victory)
//   - Population milestones
type GalaxyChronicleSystem struct {
	*BaseSystem
	prevLeader  string
	prevTotalPop int64
	prevCredits int
	entries     []string
	nextCheck   int64
}

func (gcs *GalaxyChronicleSystem) OnTick(tick int64) {
	if tick%5000 != 0 {
		return
	}

	ctx := gcs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Find current leader (by credits)
	leader := ""
	leaderCredits := 0
	totalCredits := 0
	totalPop := int64(0)

	for _, p := range players {
		if p == nil {
			continue
		}
		totalCredits += p.Credits
		if p.Credits > leaderCredits {
			leaderCredits = p.Credits
			leader = p.Name
		}
	}

	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				totalPop += planet.Population
			}
		}
	}

	minutes := tick / 600

	// Leadership change
	if gcs.prevLeader != "" && leader != gcs.prevLeader {
		entry := fmt.Sprintf("📜 Chronicle: In minute %d, %s overtook %s for galactic supremacy. A new era begins.",
			minutes, leader, gcs.prevLeader)
		game.LogEvent("event", "", entry)
		gcs.entries = append(gcs.entries, entry)
	}
	gcs.prevLeader = leader

	// Population milestones
	popMilestones := []int64{10000, 25000, 50000, 100000, 250000}
	for _, m := range popMilestones {
		if totalPop >= m && gcs.prevTotalPop < m {
			entry := fmt.Sprintf("📜 Chronicle: Galactic population reached %d in minute %d. The void fills with life.",
				m, minutes)
			game.LogEvent("event", "", entry)
			gcs.entries = append(gcs.entries, entry)
		}
	}
	gcs.prevTotalPop = totalPop

	// Credit milestones
	creditMilestones := []int{1000000, 5000000, 10000000, 50000000}
	for _, m := range creditMilestones {
		if totalCredits >= m && gcs.prevCredits < m {
			entry := fmt.Sprintf("📜 Chronicle: Total galactic wealth exceeded %d credits in minute %d. Prosperity spreads!",
				m, minutes)
			game.LogEvent("event", "", entry)
		}
	}
	gcs.prevCredits = totalCredits

	// Random flavor chronicle
	if rand.Intn(10) == 0 && len(gcs.entries) > 0 {
		game.LogEvent("intel", "",
			fmt.Sprintf("📜 This galaxy's history spans %d minutes and %d chronicle entries. Legends are being written.",
				minutes, len(gcs.entries)))
	}
}
