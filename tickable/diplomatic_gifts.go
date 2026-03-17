package tickable

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterSystem(&DiplomaticGiftSystem{
		BaseSystem: NewBaseSystem("DiplomaticGifts", 143),
	})
}

// DiplomaticGiftSystem enables wealthy factions to improve relations
// by sending diplomatic gifts. When a faction has 3x+ the credits
// of a Neutral or Unfriendly neighbor, they auto-send a gift to
// improve relations.
//
// Gift mechanics:
//   - Cost: 2000cr per gift
//   - Effect: +1 diplomacy level with target
//   - Cooldown: 20,000 ticks between gifts to same faction
//   - Only triggers for Neutral (0) or Unfriendly (-1) relations
//   - Never gifts to Hostile (-2) factions (too far gone)
//
// This creates diplomatic spending as a strategy: wealthy factions
// can buy friends, opening trade agreements and tech sharing.
type DiplomaticGiftSystem struct {
	*BaseSystem
	lastGift map[string]int64 // "a→b" key → last gift tick
}

func (dgs *DiplomaticGiftSystem) OnTick(tick int64) {
	if tick%5000 != 0 {
		return
	}

	ctx := dgs.GetContext()
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

	if dgs.lastGift == nil {
		dgs.lastGift = make(map[string]int64)
	}

	players := ctx.GetPlayers()

	for _, giver := range players {
		if giver == nil || giver.Credits < 10000 {
			continue
		}

		for _, receiver := range players {
			if receiver == nil || receiver.Name == giver.Name {
				continue
			}

			rel := dm.GetRelation(giver.Name, receiver.Name)
			if rel < -1 || rel >= 1 {
				continue // only gift to Neutral or Unfriendly
			}

			// Must have 3x their credits
			if giver.Credits < receiver.Credits*3 {
				continue
			}

			key := giver.Name + "→" + receiver.Name
			if tick-dgs.lastGift[key] < 20000 {
				continue
			}

			// 20% chance per check
			if rand.Intn(5) != 0 {
				continue
			}

			giver.Credits -= 2000
			dgs.lastGift[key] = tick
			dm.ImproveRelation(giver.Name, receiver.Name)

			game.LogEvent("event", giver.Name,
				fmt.Sprintf("🎁 %s sent a diplomatic gift to %s! (-2000cr, relations improved)",
					giver.Name, receiver.Name))
			return // one gift per tick
		}
	}
}
