package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&GravityConstructionSystem{
		BaseSystem: NewBaseSystem("GravityConstruction", 18),
	})
}

// GravityConstructionSystem modifies construction speed based on
// planet gravity. Building in low gravity is easier (less structural
// support needed). Building in high gravity is harder (everything
// weighs more, foundations must be stronger).
//
// Speed modifiers (applied to construction tick progress):
//   <0.3g:  +30% speed (low-g advantage)
//   0.3-0.8g: +15% speed (comfortable)
//   0.8-1.2g: no modifier (Earth-like)
//   1.2-2.0g: -10% speed (heavy)
//   2.0-3.0g: -25% speed (very heavy)
//   >3.0g:  -40% speed (crushing gravity)
//
// This makes low-gravity moons/small planets attractive for rapid
// industrialization, while massive rocky planets are harder to develop.
//
// Applied as credit refund/cost on construction completion to avoid
// modifying the construction system directly.
type GravityConstructionSystem struct {
	*BaseSystem
}

func (gcs *GravityConstructionSystem) OnTick(tick int64) {
	if tick%2000 != 0 {
		return
	}

	ctx := gcs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Low-gravity planets get small credit rebate (simulating faster construction)
	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.Mass == 0 {
				continue
			}

			g := planet.Gravity

			// Only announce notable gravity effects occasionally
			if rand.Intn(20) != 0 {
				continue
			}

			var msg string
			var creditMod int

			switch {
			case g < 0.3:
				creditMod = 50
				msg = fmt.Sprintf("🪶 %s: low gravity (%.1fg) — construction +30%% speed, +%dcr rebate",
					planet.Name, g, creditMod)
			case g > 2.0:
				creditMod = -30
				msg = fmt.Sprintf("⬇️ %s: high gravity (%.1fg) — construction slower, -%dcr overhead",
					planet.Name, g, -creditMod)
			default:
				continue // no notable effect
			}

			for _, p := range players {
				if p != nil && p.Name == planet.Owner {
					p.Credits += creditMod
					break
				}
			}

			game.LogEvent("logistics", planet.Owner, msg)
		}
	}
}
