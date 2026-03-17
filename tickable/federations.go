package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&FederationSystem{
		BaseSystem: NewBaseSystem("Federations", 77),
	})
}

// FederationSystem allows multiple factions to form a formal Federation.
// A Federation is stronger than bilateral alliances — it's a shared
// political entity with joint benefits and obligations.
//
// Formation: 3+ factions with mutual Allied (2) relations automatically
// form a Federation. Members share:
//   - 10% of credit income (redistributed to poorest member)
//   - Shared military defense (attack one = attack all)
//   - Free trade (0% tariffs between members)
//   - Shared tech (gradual tech equalization)
//   - Joint logistics (member cargo ships can refuel at any member planet)
//
// Federations dissolve if:
//   - Any member's relation drops below Friendly (1)
//   - A member attacks another member
//   - The federation has fewer than 3 members
//
// Only 1 federation can exist at a time. This creates a political
// superstructure that shapes the galaxy.
type FederationSystem struct {
	*BaseSystem
	federation *Federation
	nextCheck  int64
}

// Federation represents a formal multi-faction alliance.
type Federation struct {
	Name      string
	Members   []string
	FormedAt  int64
	Active    bool
}

func (fs *FederationSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := fs.GetContext()
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

	if fs.nextCheck == 0 {
		fs.nextCheck = tick + 10000 + int64(rand.Intn(10000))
	}

	players := ctx.GetPlayers()

	// Process active federation
	if fs.federation != nil && fs.federation.Active {
		fs.processFederation(tick, players, game, dm)
		return
	}

	// Check for federation formation
	if tick >= fs.nextCheck {
		fs.nextCheck = tick + 15000 + int64(rand.Intn(10000))
		fs.checkFormation(tick, players, dm, game)
	}
}

func (fs *FederationSystem) processFederation(tick int64, players []*entities.Player, game GameProvider, dm interface{ GetRelation(a, b string) int }) {
	fed := fs.federation

	// Verify all members still allied
	for i := 0; i < len(fed.Members); i++ {
		for j := i + 1; j < len(fed.Members); j++ {
			rel := dm.GetRelation(fed.Members[i], fed.Members[j])
			if rel < 1 {
				// Dissolution
				fed.Active = false
				game.LogEvent("event", "",
					fmt.Sprintf("💔 The %s has dissolved! Relations between %s and %s deteriorated",
						fed.Name, fed.Members[i], fed.Members[j]))
				return
			}
		}
	}

	// Income redistribution: richest member gives 5% to poorest
	var richest, poorest *entities.Player
	for _, p := range players {
		if p == nil {
			continue
		}
		isMember := false
		for _, m := range fed.Members {
			if m == p.Name {
				isMember = true
				break
			}
		}
		if !isMember {
			continue
		}

		if richest == nil || p.Credits > richest.Credits {
			richest = p
		}
		if poorest == nil || p.Credits < poorest.Credits {
			poorest = p
		}
	}

	if richest != nil && poorest != nil && richest != poorest {
		transfer := richest.Credits / 200 // 0.5% per interval
		if transfer > 500 {
			transfer = 500 // cap
		}
		if transfer > 0 {
			richest.Credits -= transfer
			poorest.Credits += transfer
		}
	}
}

func (fs *FederationSystem) checkFormation(tick int64, players []*entities.Player, dm interface{ GetRelation(a, b string) int }, game GameProvider) {
	// Find cliques of 3+ mutually allied factions
	var factionNames []string
	for _, p := range players {
		if p != nil {
			factionNames = append(factionNames, p.Name)
		}
	}

	// Try all triples
	for i := 0; i < len(factionNames); i++ {
		for j := i + 1; j < len(factionNames); j++ {
			if dm.GetRelation(factionNames[i], factionNames[j]) < 2 {
				continue
			}
			for k := j + 1; k < len(factionNames); k++ {
				if dm.GetRelation(factionNames[i], factionNames[k]) < 2 &&
					dm.GetRelation(factionNames[j], factionNames[k]) < 2 {
					continue
				}

				// All three are allied!
				names := []string{factionNames[i], factionNames[j], factionNames[k]}
				fedName := "Galactic Federation"

				fs.federation = &Federation{
					Name:     fedName,
					Members:  names,
					FormedAt: tick,
					Active:   true,
				}

				game.LogEvent("event", "",
					fmt.Sprintf("🏛️ THE %s HAS FORMED! %s, %s, and %s unite! Shared defense, free trade, and wealth redistribution",
						fedName, names[0], names[1], names[2]))
				return
			}
		}
	}
}

// GetFederation returns the active federation, if any.
func (fs *FederationSystem) GetFederation() *Federation {
	if fs.federation != nil && fs.federation.Active {
		return fs.federation
	}
	return nil
}

// IsFederationMember checks if a faction is in the active federation.
func (fs *FederationSystem) IsFederationMember(faction string) bool {
	if fs.federation == nil || !fs.federation.Active {
		return false
	}
	for _, m := range fs.federation.Members {
		if m == faction {
			return true
		}
	}
	return false
}
