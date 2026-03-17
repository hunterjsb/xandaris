package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&MegaprojectSystem{
		BaseSystem: NewBaseSystem("Megaprojects", 79),
	})
}

// MegaprojectSystem creates galaxy-wide cooperative construction
// projects that any faction can contribute to. When completed,
// all contributors share the benefits proportionally.
//
// Megaprojects:
//   Galactic Relay Network: All factions contribute credits → when funded,
//     all ships get +10% speed permanently
//     Cost: 5,000,000 credits total
//
//   Universal Trade Index: All factions contribute credits → when funded,
//     market prices become 20% more stable (less volatility)
//     Cost: 3,000,000 credits total
//
//   Hyperspace Stabilizer: All factions contribute credits → when funded,
//     hyperspace storms become 50% less frequent
//     Cost: 4,000,000 credits total
//
// Contributors are tracked by percentage. Benefits scale with contribution.
// Megaprojects take a long time but transform the galaxy when complete.
type MegaprojectSystem struct {
	*BaseSystem
	projects  []*Megaproject
	nextCheck int64
}

// Megaproject represents a galaxy-wide construction effort.
type Megaproject struct {
	Name          string
	Description   string
	TotalCost     int
	Funded        int
	Contributors  map[string]int // factionName → credits contributed
	Active        bool
	Completed     bool
}

var megaprojectDefs = []struct {
	name string
	desc string
	cost int
}{
	{"Galactic Relay Network", "+10% ship speed for all factions", 5000000},
	{"Universal Trade Index", "20% less market price volatility", 3000000},
	{"Hyperspace Stabilizer", "50% fewer hyperspace storms", 4000000},
}

func (ms *MegaprojectSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := ms.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ms.nextCheck == 0 {
		ms.nextCheck = tick + 10000 + int64(rand.Intn(10000))
	}

	players := ctx.GetPlayers()

	// Auto-contribute from wealthy factions
	for _, project := range ms.projects {
		if !project.Active || project.Completed {
			continue
		}

		for _, p := range players {
			if p == nil || p.Credits < 50000 {
				continue
			}

			// Factions contribute 0.1% of credits per interval (voluntary)
			contribution := p.Credits / 1000
			if contribution > 5000 {
				contribution = 5000 // cap per interval
			}
			if contribution <= 0 {
				continue
			}

			remaining := project.TotalCost - project.Funded
			if contribution > remaining {
				contribution = remaining
			}

			p.Credits -= contribution
			project.Funded += contribution
			if project.Contributors == nil {
				project.Contributors = make(map[string]int)
			}
			project.Contributors[p.Name] += contribution

			// Check completion
			if project.Funded >= project.TotalCost {
				project.Completed = true
				project.Active = false

				// Announce with contributor list
				msg := fmt.Sprintf("🏗️ MEGAPROJECT COMPLETE: %s! Benefit: %s. Contributors: ",
					project.Name, project.Description)
				for name, amount := range project.Contributors {
					pct := float64(amount) / float64(project.TotalCost) * 100
					msg += fmt.Sprintf("%s (%.0f%%) ", name, pct)
				}
				game.LogEvent("event", "", msg)
				break
			}
		}

		// Progress update every 10 intervals
		if project.Active && project.Funded > 0 && rand.Intn(10) == 0 {
			pct := float64(project.Funded) / float64(project.TotalCost) * 100
			game.LogEvent("event", "",
				fmt.Sprintf("🏗️ %s: %.0f%% funded (%d/%d credits). All factions can contribute!",
					project.Name, pct, project.Funded, project.TotalCost))
		}
	}

	// Launch new megaproject
	if tick >= ms.nextCheck {
		ms.nextCheck = tick + 30000 + int64(rand.Intn(20000))

		// Check if any project is already active
		for _, p := range ms.projects {
			if p.Active {
				return
			}
		}

		// Pick a project that hasn't been completed
		completed := make(map[string]bool)
		for _, p := range ms.projects {
			if p.Completed {
				completed[p.Name] = true
			}
		}

		var available []struct{ name, desc string; cost int }
		for _, def := range megaprojectDefs {
			if !completed[def.name] {
				available = append(available, def)
			}
		}
		if len(available) == 0 {
			return
		}

		def := available[rand.Intn(len(available))]
		project := &Megaproject{
			Name:         def.name,
			Description:  def.desc,
			TotalCost:    def.cost,
			Contributors: make(map[string]int),
			Active:       true,
		}
		ms.projects = append(ms.projects, project)

		_ = players // used above

		game.LogEvent("event", "",
			fmt.Sprintf("🏗️ MEGAPROJECT ANNOUNCED: %s! Cost: %d credits (shared by all factions). Benefit: %s. Wealthy factions will auto-contribute!",
				def.name, def.cost, def.desc))
	}
}

// GetActiveProject returns the current megaproject.
func (ms *MegaprojectSystem) GetActiveProject() *Megaproject {
	for _, p := range ms.projects {
		if p.Active {
			return p
		}
	}
	return nil
}
