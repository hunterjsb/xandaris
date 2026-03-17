package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&HyperspaceStormSystem{
		BaseSystem: NewBaseSystem("HyperspaceStorms", 19),
	})
}

// HyperspaceStormSystem generates temporary storms that disrupt travel
// along specific hyperlanes. Ships caught in a storm take damage,
// consume extra fuel, and travel slower.
//
// Storms create tactical decisions:
//   - Reroute shipping through longer but safe paths?
//   - Risk the storm for a time-critical delivery?
//   - Exploit storms to trap enemy fleets in unfavorable positions?
//
// Storm types:
//   Ion Storm:     2x fuel consumption, 10% hull damage per tick
//   Gravity Well:  50% speed reduction, ship may be pulled off course
//   Radiation Burst: No damage but forces shields up (cargo exposed to theft)
//
// Storms last 3000-8000 ticks and affect all ships on that hyperlane.
type HyperspaceStormSystem struct {
	*BaseSystem
	storms    []*HyperspaceStorm
	nextStorm int64
}

// HyperspaceStorm represents a disruption on a hyperlane.
type HyperspaceStorm struct {
	SystemA   int
	SystemB   int
	NameA     string
	NameB     string
	StormType string // "ion", "gravity", "radiation"
	TicksLeft int
	Active    bool
}

func (hss *HyperspaceStormSystem) OnTick(tick int64) {
	if tick%100 != 0 {
		return
	}

	ctx := hss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if hss.nextStorm == 0 {
		hss.nextStorm = tick + 3000 + int64(rand.Intn(5000))
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()
	hyperlanes := game.GetHyperlanes()

	// Decay storms
	for _, storm := range hss.storms {
		if !storm.Active {
			continue
		}
		storm.TicksLeft -= 100
		if storm.TicksLeft <= 0 {
			storm.Active = false
			game.LogEvent("event", "",
				fmt.Sprintf("✅ Hyperspace %s storm between %s and %s has dissipated. Travel is safe again",
					storm.StormType, storm.NameA, storm.NameB))
		}
	}

	// Apply storm effects to ships in transit
	for _, storm := range hss.storms {
		if !storm.Active {
			continue
		}
		hss.applyStormEffects(storm, players)
	}

	// Spawn new storm
	if tick >= hss.nextStorm && len(hyperlanes) > 0 {
		hss.nextStorm = tick + 10000 + int64(rand.Intn(10000))

		// Max 2 active storms
		activeCount := 0
		for _, s := range hss.storms {
			if s.Active {
				activeCount++
			}
		}
		if activeCount >= 2 {
			return
		}

		hss.spawnStorm(game, systems, hyperlanes)
	}
}

func (hss *HyperspaceStormSystem) applyStormEffects(storm *HyperspaceStorm, players []*entities.Player) {
	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil || ship.Status != entities.ShipStatusMoving {
				continue
			}
			// Check if ship is on the affected hyperlane
			if !((ship.CurrentSystem == storm.SystemA && ship.TargetSystem == storm.SystemB) ||
				(ship.CurrentSystem == storm.SystemB && ship.TargetSystem == storm.SystemA)) {
				continue
			}

			switch storm.StormType {
			case "ion":
				// Extra fuel consumption + hull damage
				ship.ConsumeFuel(2)
				ship.CurrentHealth -= 3
				if ship.CurrentHealth < 1 {
					ship.CurrentHealth = 1 // storms don't destroy, just damage
				}
			case "gravity":
				// Slow down: reduce travel progress
				ship.TravelProgress -= 0.003
				if ship.TravelProgress < 0.01 {
					ship.TravelProgress = 0.01
				}
			case "radiation":
				// Extra fuel consumption only
				ship.ConsumeFuel(1)
			}
		}
	}
}

func (hss *HyperspaceStormSystem) spawnStorm(game GameProvider, systems []*entities.System, hyperlanes []entities.Hyperlane) {
	lane := hyperlanes[rand.Intn(len(hyperlanes))]
	sysMap := game.GetSystemsMap()

	nameA := fmt.Sprintf("SYS-%d", lane.From+1)
	nameB := fmt.Sprintf("SYS-%d", lane.To+1)
	if s, ok := sysMap[lane.From]; ok {
		nameA = s.Name
	}
	if s, ok := sysMap[lane.To]; ok {
		nameB = s.Name
	}

	stormTypes := []string{"ion", "gravity", "radiation"}
	stormType := stormTypes[rand.Intn(len(stormTypes))]
	duration := 3000 + rand.Intn(5000)

	storm := &HyperspaceStorm{
		SystemA:   lane.From,
		SystemB:   lane.To,
		NameA:     nameA,
		NameB:     nameB,
		StormType: stormType,
		TicksLeft: duration,
		Active:    true,
	}
	hss.storms = append(hss.storms, storm)

	emoji := "⚡"
	desc := "increased fuel consumption and hull damage"
	switch stormType {
	case "gravity":
		emoji = "🌀"
		desc = "ships travel at half speed"
	case "radiation":
		emoji = "☢️"
		desc = "increased fuel consumption"
	}

	game.LogEvent("event", "",
		fmt.Sprintf("%s HYPERSPACE STORM between %s and %s! %s storm — %s. Reroute ships or brace for impact! (~%d min)",
			emoji, nameA, nameB, stormType, desc, duration/600))
}

// GetActiveStorms returns currently active storms.
func (hss *HyperspaceStormSystem) GetActiveStorms() []*HyperspaceStorm {
	var result []*HyperspaceStorm
	for _, s := range hss.storms {
		if s.Active {
			result = append(result, s)
		}
	}
	return result
}

// IsStormActive checks if a specific hyperlane has an active storm.
func (hss *HyperspaceStormSystem) IsStormActive(sysA, sysB int) bool {
	for _, s := range hss.storms {
		if !s.Active {
			continue
		}
		if (s.SystemA == sysA && s.SystemB == sysB) ||
			(s.SystemA == sysB && s.SystemB == sysA) {
			return true
		}
	}
	return false
}
