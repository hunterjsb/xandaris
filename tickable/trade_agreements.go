package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradeAgreementSystem{
		BaseSystem: NewBaseSystem("TradeAgreements", 69),
	})
}

// TradeAgreementSystem facilitates bilateral trade deals between factions.
// When two factions have Friendly+ relations and share a system, they can
// form a trade agreement that provides mutual benefits.
//
// Agreement types:
//   Resource Swap: Each faction provides their surplus, gets the other's surplus.
//     E.g., Faction A gives 50 Iron/interval, Faction B gives 50 Water/interval.
//
//   Most Favored Nation: Both factions get 10% discount on mutual trades.
//     Stacks with reputation discounts.
//
//   Joint Venture: Both factions contribute credits to a pool that generates
//     returns based on combined trade volume. Like a mutual fund.
//
// Agreements auto-form when conditions are met. They break if relations
// drop below Friendly or if one faction stops meeting obligations.
type TradeAgreementSystem struct {
	*BaseSystem
	agreements []*TradeAgreement
	nextCheck  int64
}

// TradeAgreement represents a bilateral trade deal.
type TradeAgreement struct {
	FactionA    string
	FactionB    string
	Type        string // "swap", "mfn", "joint_venture"
	SystemID    int
	SystemName  string
	ResourceA   string // what A provides (for swap)
	ResourceB   string // what B provides (for swap)
	TicksActive int
	Active      bool
}

func (tas *TradeAgreementSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := tas.GetContext()
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

	if tas.nextCheck == 0 {
		tas.nextCheck = tick + 3000 + int64(rand.Intn(5000))
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Process active agreements
	for _, ag := range tas.agreements {
		if !ag.Active {
			continue
		}

		// Check relations still friendly
		relation := dm.GetRelation(ag.FactionA, ag.FactionB)
		if relation < 1 {
			ag.Active = false
			game.LogEvent("trade", ag.FactionA,
				fmt.Sprintf("📜 Trade agreement between %s and %s dissolved — relations deteriorated",
					ag.FactionA, ag.FactionB))
			continue
		}

		ag.TicksActive += 1000
		tas.executeAgreement(ag, players, systems, game)
	}

	// Form new agreements
	if tick >= tas.nextCheck {
		tas.nextCheck = tick + 8000 + int64(rand.Intn(10000))
		tas.formNewAgreements(players, systems, dm, game)
	}
}

func (tas *TradeAgreementSystem) executeAgreement(ag *TradeAgreement, players []*entities.Player, systems []*entities.System, game GameProvider) {
	switch ag.Type {
	case "swap":
		tas.executeSwap(ag, players, systems, game)
	case "joint_venture":
		tas.executeJointVenture(ag, players, game)
	}
}

func (tas *TradeAgreementSystem) executeSwap(ag *TradeAgreement, players []*entities.Player, systems []*entities.System, game GameProvider) {
	// Find planets in the shared system
	var planetA, planetB *entities.Planet
	for _, sys := range systems {
		if sys.ID != ag.SystemID {
			continue
		}
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok {
				if planet.Owner == ag.FactionA && planetA == nil {
					planetA = planet
				}
				if planet.Owner == ag.FactionB && planetB == nil {
					planetB = planet
				}
			}
		}
		break
	}

	if planetA == nil || planetB == nil {
		ag.Active = false
		return
	}

	// Swap: A gives ResourceA to B, B gives ResourceB to A
	swapQty := 20
	stockA := planetA.GetStoredAmount(ag.ResourceA)
	stockB := planetB.GetStoredAmount(ag.ResourceB)

	if stockA >= swapQty && stockB >= swapQty {
		planetA.RemoveStoredResource(ag.ResourceA, swapQty)
		planetB.AddStoredResource(ag.ResourceA, swapQty)
		planetB.RemoveStoredResource(ag.ResourceB, swapQty)
		planetA.AddStoredResource(ag.ResourceB, swapQty)

		// Only log periodically
		if ag.TicksActive%10000 == 0 {
			game.LogEvent("trade", ag.FactionA,
				fmt.Sprintf("🤝 Resource swap in %s: %s↔%s (%s/%s), running for %d min",
					ag.SystemName, ag.FactionA, ag.FactionB,
					ag.ResourceA, ag.ResourceB, ag.TicksActive/600))
		}
	}
}

func (tas *TradeAgreementSystem) executeJointVenture(ag *TradeAgreement, players []*entities.Player, game GameProvider) {
	// Both factions contribute 50cr and get back 55cr if combined trade is high enough
	var playerA, playerB *entities.Player
	for _, p := range players {
		if p == nil {
			continue
		}
		if p.Name == ag.FactionA {
			playerA = p
		}
		if p.Name == ag.FactionB {
			playerB = p
		}
	}

	if playerA == nil || playerB == nil {
		ag.Active = false
		return
	}

	// Simple: both pay 50, both get 60 (10cr profit each from "synergy")
	if playerA.Credits >= 50 && playerB.Credits >= 50 {
		playerA.Credits -= 50
		playerB.Credits -= 50
		playerA.Credits += 60
		playerB.Credits += 60
	}
}

func (tas *TradeAgreementSystem) formNewAgreements(players []*entities.Player, systems []*entities.System, dm interface{ GetRelation(a, b string) int }, game GameProvider) {
	// Find pairs of friendly factions sharing a system
	for _, sys := range systems {
		factionPlanets := make(map[string]*entities.Planet)
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
				if factionPlanets[planet.Owner] == nil {
					factionPlanets[planet.Owner] = planet
				}
			}
		}

		if len(factionPlanets) < 2 {
			continue
		}

		factions := make([]string, 0, len(factionPlanets))
		for name := range factionPlanets {
			factions = append(factions, name)
		}

		for i := 0; i < len(factions); i++ {
			for j := i + 1; j < len(factions); j++ {
				a, b := factions[i], factions[j]
				if dm.GetRelation(a, b) < 1 {
					continue
				}

				// Check if agreement already exists
				exists := false
				for _, ag := range tas.agreements {
					if ag.Active &&
						((ag.FactionA == a && ag.FactionB == b) ||
							(ag.FactionA == b && ag.FactionB == a)) {
						exists = true
						break
					}
				}
				if exists {
					continue
				}

				// 10% chance to form
				if rand.Intn(10) != 0 {
					continue
				}

				planetA := factionPlanets[a]
				planetB := factionPlanets[b]

				// Determine best swap: find what each has surplus of
				resA, resB := findBestSwap(planetA, planetB)
				if resA == "" || resB == "" || resA == resB {
					// No good swap — try joint venture instead
					ag := &TradeAgreement{
						FactionA: a, FactionB: b,
						Type: "joint_venture", SystemID: sys.ID, SystemName: sys.Name,
						Active: true,
					}
					tas.agreements = append(tas.agreements, ag)
					game.LogEvent("trade", "",
						fmt.Sprintf("🤝 Joint venture formed between %s and %s in %s! Mutual investment pool generating returns",
							a, b, sys.Name))
				} else {
					ag := &TradeAgreement{
						FactionA: a, FactionB: b,
						Type: "swap", SystemID: sys.ID, SystemName: sys.Name,
						ResourceA: resA, ResourceB: resB,
						Active: true,
					}
					tas.agreements = append(tas.agreements, ag)
					game.LogEvent("trade", "",
						fmt.Sprintf("🤝 Resource swap agreement in %s: %s provides %s, %s provides %s",
							sys.Name, a, resA, b, resB))
				}
				return // one new agreement per check
			}
		}
	}
}

func findBestSwap(planetA, planetB *entities.Planet) (string, string) {
	resources := []string{entities.ResIron, entities.ResWater, entities.ResOil,
		entities.ResFuel, entities.ResRareMetals}

	bestA := ""
	bestAStock := 0
	bestB := ""
	bestBStock := 0

	for _, res := range resources {
		stockA := planetA.GetStoredAmount(res)
		stockB := planetB.GetStoredAmount(res)

		// A's surplus is B's deficit and vice versa
		if stockA > 100 && stockA > bestAStock {
			bestA = res
			bestAStock = stockA
		}
		if stockB > 100 && stockB > bestBStock {
			bestB = res
			bestBStock = stockB
		}
	}

	return bestA, bestB
}
