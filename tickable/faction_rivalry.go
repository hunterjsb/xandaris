package tickable

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&FactionRivalrySystem{
		BaseSystem: NewBaseSystem("FactionRivalry", 111),
	})
}

// FactionRivalrySystem tracks competitive dynamics between the top
// factions and generates rivalry events when two factions are close
// in power. Rivalries create narrative tension and minor gameplay effects.
//
// A rivalry forms when two factions are within 20% of each other in
// total power AND share at least one system. Rivalries are announced
// and create:
//   - Increased trade competition (auto-orders more aggressive)
//   - Diplomatic pressure (harder to maintain friendly relations)
//   - Rivalry events ("X just surpassed Y in population!")
//   - Rivalry resolution (+1000cr for the faction that pulls ahead)
//
// Max 2 active rivalries. Creates the "sports season" narrative.
type FactionRivalrySystem struct {
	*BaseSystem
	rivalries []*Rivalry
	nextCheck int64
}

// Rivalry tracks competition between two factions.
type Rivalry struct {
	FactionA   string
	FactionB   string
	PowerA     int
	PowerB     int
	Duration   int // ticks active
	Active     bool
}

func (frs *FactionRivalrySystem) OnTick(tick int64) {
	if tick%3000 != 0 {
		return
	}

	ctx := frs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if frs.nextCheck == 0 {
		frs.nextCheck = tick + 5000
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Update active rivalries
	for _, r := range frs.rivalries {
		if !r.Active {
			continue
		}
		r.Duration += 3000

		// Recalculate power
		r.PowerA = frs.calcPower(r.FactionA, players, systems)
		r.PowerB = frs.calcPower(r.FactionB, players, systems)

		// Check if rivalry resolved (one faction pulls ahead by 50%+)
		if r.PowerA > 0 && r.PowerB > 0 {
			ratio := float64(r.PowerA) / float64(r.PowerB)
			if ratio > 1.5 || ratio < 0.67 {
				winner := r.FactionA
				loser := r.FactionB
				if r.PowerB > r.PowerA {
					winner, loser = loser, winner
				}

				for _, p := range players {
					if p != nil && p.Name == winner {
						p.Credits += 1000
						break
					}
				}

				r.Active = false
				game.LogEvent("event", winner,
					fmt.Sprintf("🏆 Rivalry resolved! %s decisively overtakes %s after %d minutes of competition! +1000cr",
						winner, loser, r.Duration/600))
			}
		}

		// Rivalry commentary
		if r.Active && rand.Intn(5) == 0 {
			diff := math.Abs(float64(r.PowerA - r.PowerB))
			pct := diff / float64(max(r.PowerA, r.PowerB)) * 100
			if pct < 5 {
				game.LogEvent("intel", "",
					fmt.Sprintf("🔥 NECK AND NECK! %s vs %s — power gap under 5%%! Who will break away?",
						r.FactionA, r.FactionB))
			}
		}
	}

	// Scan for new rivalries
	if tick < frs.nextCheck {
		return
	}
	frs.nextCheck = tick + 15000 + int64(rand.Intn(10000))

	activeCount := 0
	for _, r := range frs.rivalries {
		if r.Active {
			activeCount++
		}
	}
	if activeCount >= 2 {
		return
	}

	// Find close pairs
	type fp struct {
		name  string
		power int
	}
	var factions []fp
	for _, p := range players {
		if p == nil {
			continue
		}
		factions = append(factions, fp{p.Name, frs.calcPower(p.Name, players, systems)})
	}

	for i := 0; i < len(factions); i++ {
		for j := i + 1; j < len(factions); j++ {
			a, b := factions[i], factions[j]
			if a.power == 0 || b.power == 0 {
				continue
			}
			ratio := float64(a.power) / float64(b.power)
			if ratio > 0.8 && ratio < 1.2 {
				// Check not already rivaled
				exists := false
				for _, r := range frs.rivalries {
					if r.Active && ((r.FactionA == a.name && r.FactionB == b.name) ||
						(r.FactionA == b.name && r.FactionB == a.name)) {
						exists = true
					}
				}
				if exists {
					continue
				}

				frs.rivalries = append(frs.rivalries, &Rivalry{
					FactionA: a.name, FactionB: b.name,
					PowerA: a.power, PowerB: b.power,
					Active: true,
				})
				game.LogEvent("event", "",
					fmt.Sprintf("⚡ RIVALRY! %s and %s are neck-and-neck in power! Who will come out on top?",
						a.name, b.name))
				return
			}
		}
	}
}

func (frs *FactionRivalrySystem) calcPower(faction string, players []*entities.Player, systems []*entities.System) int {
	power := 0
	for _, p := range players {
		if p == nil || p.Name != faction {
			continue
		}
		power += p.Credits / 1000
		power += len(p.OwnedShips) * 10
		break
	}
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner == faction {
				power += 200 + int(planet.Population/100) + int(planet.TechLevel*50)
			}
		}
	}
	return power
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
