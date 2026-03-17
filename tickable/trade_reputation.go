package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&TradeReputationSystem{
		BaseSystem: NewBaseSystem("TradeReputation", 46),
	})
}

// TradeReputationSystem tracks each faction's trading reputation across
// the galaxy. Reputation increases from completed trades and supply
// deliveries, and decreases from piracy, blockades, and contract defaults.
//
// Reputation tiers:
//   Unknown    (0-99):   Standard prices, no perks
//   Known      (100-499): 2% price discount on purchases
//   Trusted    (500-999): 5% discount, priority order matching
//   Respected  (1000-2499): 10% discount, convoy contract access
//   Renowned   (2500-4999): 15% discount, NPC factions seek you out
//   Legendary  (5000+):    20% discount, galactic trade council seat
//
// Reputation also affects which NPC factions will trade with you.
// Blockading or raiding other factions degrades your reputation.
type TradeReputationSystem struct {
	*BaseSystem
	reputation    map[string]int    // playerName → reputation score
	lastAnnounce  map[string]string // playerName → last tier announced
	tradeCount    map[string]int    // trades this interval
}

var reputationTiers = []struct {
	threshold int
	name      string
	discount  float64
}{
	{5000, "Legendary", 0.20},
	{2500, "Renowned", 0.15},
	{1000, "Respected", 0.10},
	{500, "Trusted", 0.05},
	{100, "Known", 0.02},
	{0, "Unknown", 0.00},
}

func (trs *TradeReputationSystem) OnTick(tick int64) {
	if tick%500 != 0 {
		return
	}

	ctx := trs.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if trs.reputation == nil {
		trs.reputation = make(map[string]int)
		trs.lastAnnounce = make(map[string]string)
		trs.tradeCount = make(map[string]int)
	}

	players := ctx.GetPlayers()

	for _, player := range players {
		if player == nil {
			continue
		}

		// Gain reputation from economic activity
		trs.accumulateReputation(player, game)

		// Check for tier changes
		trs.checkTierChange(player, game)
	}

	// Passive decay for inactive traders (keeps reputation relevant)
	if tick%5000 == 0 {
		for name, rep := range trs.reputation {
			if rep > 100 {
				decay := rep / 50 // 2% decay
				trs.reputation[name] -= decay
			}
		}
	}
}

func (trs *TradeReputationSystem) accumulateReputation(player *entities.Player, game GameProvider) {
	rep := 0

	// Credits earned = trading activity proxy
	if player.Credits > 50000 {
		rep += 5
	}
	if player.Credits > 200000 {
		rep += 10
	}

	// Having cargo ships means active logistics
	cargoShips := 0
	for _, ship := range player.OwnedShips {
		if ship != nil && ship.ShipType == entities.ShipTypeCargo {
			cargoShips++
		}
	}
	rep += cargoShips * 3

	// Planet count means economic base
	planetCount := 0
	systems := game.GetSystems()
	for _, sys := range systems {
		for _, e := range sys.Entities {
			if planet, ok := e.(*entities.Planet); ok && planet.Owner == player.Name {
				planetCount++
				// Trading Post presence boosts reputation
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok && b.BuildingType == entities.BuildingTradingPost && b.IsOperational {
						rep += b.Level * 2
					}
				}
			}
		}
	}
	rep += planetCount * 2

	// Small random factor
	if rand.Intn(3) == 0 {
		rep += rand.Intn(5)
	}

	trs.reputation[player.Name] += rep
}

func (trs *TradeReputationSystem) checkTierChange(player *entities.Player, game GameProvider) {
	rep := trs.reputation[player.Name]
	currentTier := "Unknown"
	for _, tier := range reputationTiers {
		if rep >= tier.threshold {
			currentTier = tier.name
			break
		}
	}

	lastTier := trs.lastAnnounce[player.Name]
	if currentTier != lastTier && currentTier != "Unknown" {
		trs.lastAnnounce[player.Name] = currentTier
		discount := 0.0
		for _, tier := range reputationTiers {
			if tier.name == currentTier {
				discount = tier.discount
				break
			}
		}
		game.LogEvent("event", player.Name,
			fmt.Sprintf("⭐ %s's trade reputation is now %s! (%.0f%% trade discount, rep: %d)",
				player.Name, currentTier, discount*100, rep))
	}
}

// GetReputation returns a faction's reputation score.
func (trs *TradeReputationSystem) GetReputation(playerName string) int {
	if trs.reputation == nil {
		return 0
	}
	return trs.reputation[playerName]
}

// GetTierName returns the tier name for a reputation score.
func GetReputationTier(rep int) string {
	for _, tier := range reputationTiers {
		if rep >= tier.threshold {
			return tier.name
		}
	}
	return "Unknown"
}

// GetDiscount returns the trade discount for a reputation score.
func GetReputationDiscount(rep int) float64 {
	for _, tier := range reputationTiers {
		if rep >= tier.threshold {
			return tier.discount
		}
	}
	return 0
}

// ModifyReputation adjusts a faction's reputation (for external events like blockades).
func (trs *TradeReputationSystem) ModifyReputation(playerName string, delta int) {
	if trs.reputation == nil {
		trs.reputation = make(map[string]int)
	}
	trs.reputation[playerName] += delta
	if trs.reputation[playerName] < 0 {
		trs.reputation[playerName] = 0
	}
}
