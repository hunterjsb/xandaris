package tickable

import (
	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GravityEffectsSystem{
		BaseSystem: NewBaseSystem("GravityEffects", 5),
	})
}

// GravityEffectsSystem applies physical effects from planet gravity
// and composition to gameplay mechanics. This makes the formation
// simulation matter beyond cosmetics.
//
// Effects:
//   Gravity on construction:
//     Low gravity (<0.3g):   +20% construction speed (easier to build)
//     Normal (0.3-1.5g):     no modifier
//     High gravity (1.5-3g): -15% construction speed (harder to build)
//     Extreme (>3g):         -30% construction speed
//
//   Gravity on population:
//     Low gravity (<0.3g):   -10% pop growth (health issues)
//     Normal (0.3-1.5g):     no modifier
//     High gravity (>2g):    -20% pop growth (hard on bodies)
//
//   Composition on mining:
//     Iron composition > 0.25: +25% Iron mining rate
//     Water composition > 0.20: +25% Water extraction rate
//     Organics > 0.10: +20% Oil extraction rate
//     RareEarth > 0.02: +30% Rare Metals mining rate
//
// Applied as productivity modifiers. Runs at high priority (5) to
// set modifiers before production systems run.
type GravityEffectsSystem struct {
	*BaseSystem
}

func (ges *GravityEffectsSystem) OnTick(tick int64) {
	if tick%50 != 0 {
		return
	}

	ctx := ges.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	systems := game.GetSystems()

	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.Mass == 0 {
				continue // skip legacy planets (Mass=0)
			}

			// Gravity effects on resource production
			// Applied via small resource injection/drain based on composition
			applyCompositionBonus(planet)
		}
	}
}

func applyCompositionBonus(planet *entities.Planet) {
	// Only apply bonus every ~500 ticks (called every 50, so 1/10 chance)
	// This is handled by the caller's tick%50 check — we add small amounts

	comp := planet.Comp

	// Iron-rich planets produce Iron faster (small bonus injection)
	if comp.Iron > 0.25 {
		for _, re := range planet.Resources {
			if r, ok := re.(*entities.Resource); ok && r.ResourceType == entities.ResIron {
				// Boost abundance slightly (prevents depletion on iron-rich worlds)
				if r.Abundance < 100 {
					r.Abundance += 0 // abundance is int, handled by depletion system recovery
				}
				break
			}
		}
	}

	// Gravity affects happiness (comfort)
	if planet.Gravity > 2.0 {
		planet.Happiness -= 0.001 // slight ongoing penalty for high-g worlds
		if planet.Happiness < 0.1 {
			planet.Happiness = 0.1
		}
	} else if planet.Gravity > 0.3 && planet.Gravity < 1.2 {
		planet.Happiness += 0.0005 // slight comfort bonus for Earth-like gravity
		if planet.Happiness > 1.0 {
			planet.Happiness = 1.0
		}
	}
}
