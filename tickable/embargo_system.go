package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&EmbargoSystem{
		BaseSystem: NewBaseSystem("Embargo", 117),
	})
}

// EmbargoSystem lets factions embargo specific resources from specific
// rivals. An embargo prevents local exchange and auto-orders from
// trading that resource with the embargoed faction.
//
// Embargoes form automatically when:
//   - Two factions have Hostile (-2) relations
//   - Both have planets in the same system
//   - One faction has surplus of a resource the other needs
//
// Embargo effects:
//   - Local exchange system skips embargoed pairs
//   - Embargoed faction must import from elsewhere or produce locally
//   - Embargoing faction loses potential trade income
//
// Embargoes lift when relations improve to Neutral (0) or better.
// Creates economic warfare without ships.
type EmbargoSystem struct {
	*BaseSystem
	embargoes []*Embargo
	nextCheck int64
}

// Embargo represents a trade restriction between two factions.
type Embargo struct {
	Enforcer  string
	Target    string
	Resource  string
	SystemID  int
	SysName   string
	Active    bool
}

func (es *EmbargoSystem) OnTick(tick int64) {
	if tick%3000 != 0 {
		return
	}

	ctx := es.GetContext()
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

	// Lift embargoes where relations improved
	for _, emb := range es.embargoes {
		if !emb.Active {
			continue
		}
		rel := dm.GetRelation(emb.Enforcer, emb.Target)
		if rel >= 0 {
			emb.Active = false
			game.LogEvent("trade", emb.Target,
				fmt.Sprintf("✅ %s lifted %s embargo on %s. Trade resumes in %s!",
					emb.Enforcer, emb.Resource, emb.Target, emb.SysName))
		}
	}

	// Check for new embargoes
	if es.nextCheck == 0 {
		es.nextCheck = tick + 10000
	}
	if tick < es.nextCheck {
		return
	}
	es.nextCheck = tick + 10000 + int64(rand.Intn(10000))

	// Max 3 active embargoes
	activeCount := 0
	for _, emb := range es.embargoes {
		if emb.Active {
			activeCount++
		}
	}
	if activeCount >= 3 {
		return
	}

	for _, sys := range systems {
		factions := make(map[string]bool)
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				factions[planet.Owner] = true
			}
		}

		factionList := make([]string, 0, len(factions))
		for f := range factions {
			factionList = append(factionList, f)
		}

		for i := 0; i < len(factionList); i++ {
			for j := i + 1; j < len(factionList); j++ {
				a, b := factionList[i], factionList[j]
				rel := dm.GetRelation(a, b)
				if rel > -2 {
					continue
				}

				// Already embargoed?
				exists := false
				for _, emb := range es.embargoes {
					if emb.Active && emb.SystemID == sys.ID &&
						((emb.Enforcer == a && emb.Target == b) ||
							(emb.Enforcer == b && emb.Target == a)) {
						exists = true
						break
					}
				}
				if exists {
					continue
				}

				// Pick a resource to embargo
				resources := []string{entities.ResIron, entities.ResWater, entities.ResOil,
					entities.ResFuel, entities.ResRareMetals}
				res := resources[rand.Intn(len(resources))]

				es.embargoes = append(es.embargoes, &Embargo{
					Enforcer: a, Target: b,
					Resource: res, SystemID: sys.ID, SysName: sys.Name,
					Active: true,
				})

				game.LogEvent("trade", "",
					fmt.Sprintf("🚫 EMBARGO: %s blocks %s trade with %s in %s! Relations must improve to lift",
						a, res, b, sys.Name))
				return
			}
		}
	}
}
