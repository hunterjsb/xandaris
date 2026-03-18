package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&MostWantedSystem{
		BaseSystem: NewBaseSystem("MostWanted", 205),
	})
}

// MostWantedSystem publishes a "most wanted" list of galaxy needs.
// Instead of abstract market data, it tells factions in plain language
// what the galaxy needs most right now.
//
// "MOST WANTED: Fuel (32 units galaxy-wide, 52 planets need it).
//  REWARD: 3x market price for Fuel deliveries this period!"
//
// The most wanted resource gets a temporary price multiplier through
// market demand injection. Factions that produce the wanted resource
// profit enormously.
type MostWantedSystem struct {
	*BaseSystem
	currentWanted string
	nextUpdate    int64
}

func (mws *MostWantedSystem) OnTick(tick int64) {
	ctx := mws.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if mws.nextUpdate == 0 {
		mws.nextUpdate = tick + 3000 + int64(rand.Intn(3000))
	}
	if tick < mws.nextUpdate {
		return
	}
	mws.nextUpdate = tick + 6000 + int64(rand.Intn(4000))

	systems := game.GetSystems()
	market := game.GetMarketEngine()
	if market == nil {
		return
	}

	resources := []string{entities.ResIron, entities.ResWater, entities.ResOil,
		entities.ResFuel, entities.ResRareMetals, entities.ResHelium3, entities.ResElectronics}

	// Find scarcest resource
	scarcest := ""
	scarcestTotal := 999999
	needCount := 0

	for _, res := range resources {
		total := 0
		needing := 0
		for _, sys := range systems {
			for _, e := range sys.Entities {
				if planet, ok := e.(*entities.Planet); ok && planet.Owner != "" {
					amt := planet.GetStoredAmount(res)
					total += amt
					if amt < 10 {
						needing++
					}
				}
			}
		}

		if total < scarcestTotal {
			scarcestTotal = total
			scarcest = res
			needCount = needing
		}
	}

	if scarcest == "" {
		return
	}

	mws.currentWanted = scarcest

	// Inject demand to raise prices
	market.AddTradeVolume(scarcest, 50, true)

	price := market.GetSellPrice(scarcest)
	game.LogEvent("intel", "",
		fmt.Sprintf("🎯 MOST WANTED: %s! Only %d units galaxy-wide, %d planets in need. Current price: %.0fcr. Produce and profit!",
			scarcest, scarcestTotal, needCount, price))
}
