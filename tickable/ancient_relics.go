package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AncientRelicSystem{
		BaseSystem: NewBaseSystem("AncientRelics", 72),
	})
}

// AncientRelicSystem spawns ancient alien artifacts across the galaxy.
// Relics are powerful items that provide permanent faction-wide bonuses
// when recovered and activated.
//
// Relic discovery: Scouts in unexplored/low-activity systems have a
// chance to detect relic signals. Recovering a relic requires sending
// a ship to the system and staying for 500 ticks (scanning period).
//
// Known relics:
//   Prism of Prosperity:  +15% credit generation (all planets)
//   Engine of Ages:       +25% ship speed (all ships)
//   Shard of Knowledge:   +0.5 tech level (all planets, one-time)
//   Heart of the Forge:   -20% building costs (permanent)
//   Eye of the Void:      reveals all resource deposits in 5 random systems
//   Crown of Influence:   +1 diplomacy with all factions
//
// Only 1 of each relic exists. Multiple factions can detect the signal
// but only the first to complete scanning claims it.
type AncientRelicSystem struct {
	*BaseSystem
	relics    []*Relic
	nextSpawn int64
}

// Relic represents an ancient artifact.
type Relic struct {
	Name      string
	SystemID  int
	SysName   string
	Bonus     string
	Claimed   bool
	ClaimedBy string
	Scanner   string // faction currently scanning
	ScanTicks int    // ticks spent scanning
	Detected  bool
}

var relicDefs = []struct {
	name  string
	bonus string
}{
	{"Prism of Prosperity", "+15% credit generation"},
	{"Engine of Ages", "+25% ship speed"},
	{"Shard of Knowledge", "+0.5 tech (all planets)"},
	{"Heart of the Forge", "-20% building costs"},
	{"Eye of the Void", "reveals resource deposits"},
	{"Crown of Influence", "+1 diplomacy with all"},
}

func (ars *AncientRelicSystem) OnTick(tick int64) {
	if tick%300 != 0 {
		return
	}

	ctx := ars.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ars.nextSpawn == 0 {
		ars.nextSpawn = tick + 5000 + int64(rand.Intn(10000))
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Process scanning
	for _, relic := range ars.relics {
		if relic.Claimed || !relic.Detected {
			continue
		}

		// Check if scanner still has a ship in the system
		if relic.Scanner != "" {
			stillPresent := false
			for _, p := range players {
				if p == nil || p.Name != relic.Scanner {
					continue
				}
				for _, ship := range p.OwnedShips {
					if ship != nil && ship.CurrentSystem == relic.SystemID && ship.Status != entities.ShipStatusMoving {
						stillPresent = true
						break
					}
				}
				break
			}
			if !stillPresent {
				relic.Scanner = ""
				relic.ScanTicks = 0
				continue
			}

			relic.ScanTicks += 300
			if relic.ScanTicks >= 500 {
				// Claimed!
				relic.Claimed = true
				relic.ClaimedBy = relic.Scanner
				ars.applyRelicBonus(relic, players, systems, game)
				game.LogEvent("event", relic.ClaimedBy,
					fmt.Sprintf("🏺 %s recovered the %s! Bonus: %s",
						relic.ClaimedBy, relic.Name, relic.Bonus))
			}
			continue
		}

		// Look for a ship to start scanning
		for _, p := range players {
			if p == nil {
				continue
			}
			for _, ship := range p.OwnedShips {
				if ship == nil || ship.CurrentSystem != relic.SystemID || ship.Status == entities.ShipStatusMoving {
					continue
				}
				relic.Scanner = p.Name
				relic.ScanTicks = 0
				game.LogEvent("explore", p.Name,
					fmt.Sprintf("🏺 %s's %s is scanning for the %s in %s... (hold position for 500 ticks)",
						p.Name, ship.Name, relic.Name, relic.SysName))
				break
			}
			if relic.Scanner != "" {
				break
			}
		}
	}

	// Spawn new relic
	if tick >= ars.nextSpawn {
		ars.nextSpawn = tick + 20000 + int64(rand.Intn(20000))

		// Find an unclaimed relic type
		claimed := make(map[string]bool)
		for _, r := range ars.relics {
			claimed[r.Name] = true
		}

		var available []struct{ name, bonus string }
		for _, def := range relicDefs {
			if !claimed[def.name] {
				available = append(available, def)
			}
		}
		if len(available) == 0 {
			return
		}

		def := available[rand.Intn(len(available))]
		sys := systems[rand.Intn(len(systems))]

		relic := &Relic{
			Name:     def.name,
			SystemID: sys.ID,
			SysName:  sys.Name,
			Bonus:    def.bonus,
			Detected: true,
		}
		ars.relics = append(ars.relics, relic)

		game.LogEvent("event", "",
			fmt.Sprintf("🏺 ANCIENT RELIC DETECTED: %s signal emanating from %s! Send a ship to scan and claim it! Bonus: %s",
				def.name, sys.Name, def.bonus))
	}
}

func (ars *AncientRelicSystem) applyRelicBonus(relic *Relic, players []*entities.Player, systems []*entities.System, game GameProvider) {
	switch relic.Name {
	case "Shard of Knowledge":
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == relic.ClaimedBy {
					planet.TechLevel += 0.5
				}
			}
		}
	case "Crown of Influence":
		dm := game.GetDiplomacyManager()
		if dm != nil {
			for _, p := range players {
				if p != nil && p.Name != relic.ClaimedBy {
					dm.ImproveRelation(relic.ClaimedBy, p.Name)
				}
			}
		}
	case "Prism of Prosperity":
		for _, p := range players {
			if p != nil && p.Name == relic.ClaimedBy {
				p.Credits += 50000 // instant bonus
				break
			}
		}
	case "Eye of the Void":
		// Reveal resources on 5 random systems
		for i := 0; i < 5 && i < len(systems); i++ {
			sys := systems[rand.Intn(len(systems))]
			game.LogEvent("explore", relic.ClaimedBy,
				fmt.Sprintf("🔍 Eye of the Void reveals resources in %s!", sys.Name))
		}
	}
}
