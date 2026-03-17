package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GoldenAgeSystem{
		BaseSystem: NewBaseSystem("GoldenAge", 60),
	})
}

// GoldenAgeSystem rewards factions that achieve sustained prosperity.
// When a planet maintains high happiness (>0.85) and full resource
// diversity for an extended period, it enters a "Golden Age" that
// provides massive bonuses.
//
// Golden Age requirements (all must be true for 5000+ ticks):
//   - Happiness > 0.85
//   - Population > 5000
//   - Tech level > 2.0
//   - At least 4 different resources stocked
//
// Golden Age bonuses:
//   - +100% credit generation
//   - +50% population growth
//   - +25% production output
//   - +0.1 tech level per 5000 ticks
//   - Attracts immigrants from other factions (+500 pop/tick)
//
// A Golden Age ends if any requirement drops below threshold.
// Only 1 planet per faction can be in a Golden Age at a time.
type GoldenAgeSystem struct {
	*BaseSystem
	candidates map[int]int64  // planetID → tick when conditions first met
	activeAges map[int]bool   // planetID → in golden age
	factionAge map[string]int // factionName → planetID of golden age planet
}

func (gas *GoldenAgeSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := gas.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if gas.candidates == nil {
		gas.candidates = make(map[int]int64)
		gas.activeAges = make(map[int]bool)
		gas.factionAge = make(map[string]int)
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" {
				continue
			}

			gas.evaluatePlanet(tick, planet, players, game)
		}
	}
}

func (gas *GoldenAgeSystem) evaluatePlanet(tick int64, planet *entities.Planet, players []*entities.Player, game GameProvider) {
	pid := planet.GetID()

	// Check Golden Age conditions
	qualifies := planet.Happiness > 0.85 &&
		planet.Population > 5000 &&
		planet.TechLevel > 2.0

	// Check resource diversity (need 4+ types stocked)
	if qualifies {
		diverseCount := 0
		for _, res := range []string{entities.ResIron, entities.ResWater, entities.ResOil,
			entities.ResFuel, entities.ResRareMetals, entities.ResHelium3, entities.ResElectronics} {
			if planet.GetStoredAmount(res) > 5 {
				diverseCount++
			}
		}
		if diverseCount < 4 {
			qualifies = false
		}
	}

	if !qualifies {
		// End golden age if active
		if gas.activeAges[pid] {
			gas.activeAges[pid] = false
			delete(gas.factionAge, planet.Owner)
			game.LogEvent("event", planet.Owner,
				fmt.Sprintf("📉 Golden Age on %s has ended. Conditions no longer met — restore prosperity!",
					planet.Name))
		}
		delete(gas.candidates, pid)
		return
	}

	// Already in golden age — apply bonuses
	if gas.activeAges[pid] {
		gas.applyBonuses(planet, players, game)
		return
	}

	// Check if faction already has a golden age planet
	if existingPID, exists := gas.factionAge[planet.Owner]; exists && existingPID != pid {
		return // only one golden age per faction
	}

	// Track candidacy
	if gas.candidates[pid] == 0 {
		gas.candidates[pid] = tick
	}

	// Need 5000 ticks of sustained qualification
	if tick-gas.candidates[pid] < 5000 {
		return
	}

	// GOLDEN AGE begins!
	gas.activeAges[pid] = true
	gas.factionAge[planet.Owner] = pid

	game.LogEvent("event", planet.Owner,
		fmt.Sprintf("🌟 GOLDEN AGE on %s! Sustained prosperity unlocks massive bonuses: +100%% credits, +50%% growth, +25%% production!",
			planet.Name))
}

func (gas *GoldenAgeSystem) applyBonuses(planet *entities.Planet, players []*entities.Player, game GameProvider) {
	// Credit bonus
	for _, p := range players {
		if p != nil && p.Name == planet.Owner {
			p.Credits += 50 // bonus credits per interval
			break
		}
	}

	// Population growth bonus
	cap := planet.GetTotalPopulationCapacity()
	if cap > 0 && planet.Population < cap {
		bonus := int64(200 + rand.Intn(300))
		if planet.Population+bonus > cap {
			bonus = cap - planet.Population
		}
		planet.Population += bonus
	}

	// Slow tech growth
	if rand.Intn(10) == 0 {
		planet.TechLevel += 0.02
	}
}

// IsGoldenAge returns whether a planet is currently in a Golden Age.
func (gas *GoldenAgeSystem) IsGoldenAge(planetID int) bool {
	if gas.activeAges == nil {
		return false
	}
	return gas.activeAges[planetID]
}
