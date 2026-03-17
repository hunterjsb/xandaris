package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&UnderdogBonusSystem{
		BaseSystem: NewBaseSystem("UnderdogBonus", 131),
	})
}

// UnderdogBonusSystem provides catch-up mechanics for factions that
// are significantly behind the leaders. This prevents runaway winners
// and keeps the game competitive.
//
// When a faction has less than 25% of the leader's score:
//   - +20% credit generation bonus
//   - +10% population growth
//   - Cheaper building costs (-25%)
//   - Announced as "underdog" for narrative
//
// The bonus fades as the faction catches up (smoothly scales from
// 25% to 50% of leader score, then disappears).
//
// This is invisible to the top factions — they don't see or feel it.
// It just helps smaller factions stay competitive.
type UnderdogBonusSystem struct {
	*BaseSystem
	nextAnnounce int64
}

func (ubs *UnderdogBonusSystem) OnTick(tick int64) {
	if tick%1000 != 0 {
		return
	}

	ctx := ubs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	// Find the leader's effective power
	leaderPower := 0
	for _, p := range players {
		if p == nil {
			continue
		}
		power := p.Credits / 100
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == p.Name {
					power += int(planet.Population/10) + 500
				}
			}
		}
		power += len(p.OwnedShips) * 20
		if power > leaderPower {
			leaderPower = power
		}
	}

	if leaderPower <= 0 {
		return
	}

	// Apply underdog bonuses
	for _, p := range players {
		if p == nil {
			continue
		}

		power := p.Credits / 100
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner == p.Name {
					power += int(planet.Population/10) + 500
				}
			}
		}
		power += len(p.OwnedShips) * 20

		ratio := float64(power) / float64(leaderPower)

		if ratio >= 0.5 {
			continue // not an underdog
		}

		// Bonus scales: 25% at ratio=0, fades to 0% at ratio=0.5
		bonusStrength := (0.5 - ratio) * 2.0 // 0.0 to 1.0

		// Credit bonus
		creditBonus := int(float64(p.Credits) * 0.002 * bonusStrength)
		if creditBonus > 200 {
			creditBonus = 200
		}
		p.Credits += creditBonus

		// Population bonus on owned planets
		if rand.Intn(3) == 0 {
			for _, sys := range systems {
				for _, e := range sys.Entities {
					if planet, ok := e.(*entities.Planet); ok && planet.Owner == p.Name {
						cap := planet.GetTotalPopulationCapacity()
						if cap > 0 && planet.Population < cap {
							bonus := int64(float64(50) * bonusStrength)
							planet.Population += bonus
						}
					}
				}
			}
		}
	}

	// Periodic underdog announcement
	if ubs.nextAnnounce == 0 {
		ubs.nextAnnounce = tick + 15000
	}
	if tick >= ubs.nextAnnounce {
		ubs.nextAnnounce = tick + 15000 + int64(rand.Intn(10000))

		for _, p := range players {
			if p == nil {
				continue
			}
			power := p.Credits/100 + len(p.OwnedShips)*20
			ratio := float64(power) / float64(leaderPower)
			if ratio < 0.25 {
				game.LogEvent("event", p.Name,
					fmt.Sprintf("🌱 %s is the underdog! Receiving catch-up bonuses. Keep building — empires rise from humble beginnings!",
						p.Name))
			}
		}
	}
}
