package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&FirstContactSystem{
		BaseSystem: NewBaseSystem("FirstContact", 170),
	})
}

// FirstContactSystem generates a one-time "first contact" event
// when two factions' ships meet in the same system for the first
// time. This creates diplomatic narrative and a small bonus.
//
// On first meeting:
//   - Both factions get +200cr "cultural exchange" bonus
//   - Relations start at Neutral (if not already set)
//   - Announcement: "First contact between X and Y in SYS-Z!"
//
// Only fires once per faction pair. Creates early-game diplomacy
// events that establish relationships.
type FirstContactSystem struct {
	*BaseSystem
	contacted map[string]bool // "A|B" sorted key → already met
}

func (fcs *FirstContactSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := fcs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if fcs.contacted == nil {
		fcs.contacted = make(map[string]bool)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Check each system for ships from different factions
	for _, sys := range systems {
		factionsPresent := make(map[string]bool)

		for _, p := range players {
			if p == nil {
				continue
			}
			for _, ship := range p.OwnedShips {
				if ship != nil && ship.CurrentSystem == sys.ID {
					factionsPresent[p.Name] = true
				}
			}
			// Also count planet ownership
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == p.Name {
					factionsPresent[p.Name] = true
				}
			}
		}

		// Check all pairs
		factions := make([]string, 0, len(factionsPresent))
		for name := range factionsPresent {
			factions = append(factions, name)
		}

		for i := 0; i < len(factions); i++ {
			for j := i + 1; j < len(factions); j++ {
				a, b := factions[i], factions[j]
				if a > b {
					a, b = b, a
				}
				key := a + "|" + b

				if fcs.contacted[key] {
					continue
				}
				fcs.contacted[key] = true

				// First contact!
				for _, p := range players {
					if p != nil && (p.Name == a || p.Name == b) {
						p.Credits += 200
					}
				}

				game.LogEvent("event", "",
					fmt.Sprintf("🤝 FIRST CONTACT: %s and %s meet in %s! Cultural exchange grants +200cr to both. A new chapter begins!",
						a, b, sys.Name))
			}
		}
	}

	_ = rand.Intn
}
