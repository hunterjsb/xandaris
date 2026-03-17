package tickable

import (
	"fmt"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TechSharingSystem{
		BaseSystem: NewBaseSystem("TechSharing", 61),
	})
}

// TechSharingSystem allows allied factions with planets in the same
// system to share technology. The lower-tech planet slowly catches up
// to the higher-tech one.
//
// Requirements:
//   - Both factions must have planets in the same system
//   - Diplomacy must be Friendly (1) or Allied (2)
//   - Both planets must have operational Trading Posts (tech flows through trade)
//
// Mechanics:
//   - Tech transfers at 0.01 per tick (very gradual)
//   - Lower-tech planet gains, higher-tech planet doesn't lose
//   - Cap: shared tech can't exceed 80% of the donor's level
//   - This creates incentive for alliances beyond just military cooperation
//
// Combined with the existing tech system (Electronics → tech level),
// this means allied factions can bootstrap each other. A tech-3.0 ally
// sharing with a tech-1.0 partner helps them reach 2.4 (80% of 3.0).
type TechSharingSystem struct {
	*BaseSystem
}

func (tss *TechSharingSystem) OnTick(tick int64) {
	if tick%200 != 0 {
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

	dm := game.GetDiplomacyManager()
	if dm == nil {
		return
	}

	systems := game.GetSystems()

	for _, sys := range systems {
		// Collect planets by faction
		factionPlanets := make(map[string]*entities.Planet)
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}
			// Use the highest-tech planet per faction in this system
			existing := factionPlanets[planet.Owner]
			if existing == nil || planet.TechLevel > existing.TechLevel {
				// Check for Trading Post
				hasTP := false
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
						hasTP = true
						break
					}
				}
				if hasTP {
					factionPlanets[planet.Owner] = planet
				}
			}
		}

		if len(factionPlanets) < 2 {
			continue
		}

		// Check all pairs for tech sharing
		factions := make([]string, 0, len(factionPlanets))
		for name := range factionPlanets {
			factions = append(factions, name)
		}

		for i := 0; i < len(factions); i++ {
			for j := i + 1; j < len(factions); j++ {
				a, b := factions[i], factions[j]
				relation := dm.GetRelation(a, b)
				if relation < 1 {
					continue // need Friendly or Allied
				}

				planetA := factionPlanets[a]
				planetB := factionPlanets[b]

				// Transfer tech from higher to lower
				if planetA.TechLevel > planetB.TechLevel {
					tss.transferTech(planetA, planetB, a, b, sys, game)
				} else if planetB.TechLevel > planetA.TechLevel {
					tss.transferTech(planetB, planetA, b, a, sys, game)
				}
			}
		}
	}
}

func (tss *TechSharingSystem) transferTech(donor, receiver *entities.Planet, donorFaction, receiverFaction string, sys *entities.System, game GameProvider) {
	cap := donor.TechLevel * 0.8 // can only reach 80% of donor's level
	if receiver.TechLevel >= cap {
		return
	}

	transfer := 0.01
	receiver.TechLevel += transfer

	if receiver.TechLevel > cap {
		receiver.TechLevel = cap
	}

	// Announce occasionally (not every tick)
	// Use modular check on tech level milestones
	if int(receiver.TechLevel*10)%5 == 0 && int((receiver.TechLevel-transfer)*10)%5 != 0 {
		game.LogEvent("event", receiverFaction,
			fmt.Sprintf("🔬 Tech sharing in %s: %s's tech rising to %.1f (shared knowledge from %s at %.1f)",
				sys.Name, receiver.Name, receiver.TechLevel, donor.Name, donor.TechLevel))
	}
}
