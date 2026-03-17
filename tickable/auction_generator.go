package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AuctionGeneratorSystem{
		BaseSystem: NewBaseSystem("AuctionGenerator", 42),
	})
}

// AuctionGeneratorSystem periodically creates galaxy-wide auctions
// for rare items that can't be obtained any other way.
type AuctionGeneratorSystem struct {
	*BaseSystem
	nextAuction int64
}

type auctionTemplate struct {
	item        string
	description string
	minBid      int
	effect      string // what happens when won
}

var auctionItems = []auctionTemplate{
	{"Ancient Star Chart", "Reveals all resource deposits in 3 random systems", 5000, "intel"},
	{"Prototype Engine", "Doubles a ship's fuel capacity permanently", 10000, "ship_upgrade"},
	{"Quantum Computer", "+1.0 tech level on your most advanced planet", 15000, "tech_boost"},
	{"Rare Mineral Cache", "500 Rare Metals delivered instantly", 8000, "resource"},
	{"Military Contract", "Spawns 2 free Frigates at your homeworld", 12000, "ships"},
	{"Trade License", "+50% Trading Post revenue for 10,000 ticks", 7000, "buff"},
	{"Colony Blueprint", "Reduces next Colony ship build time by 75%", 6000, "construction"},
	{"Shield Generator", "Free Planetary Shield on your most populated planet", 20000, "building"},
}

func (ags *AuctionGeneratorSystem) OnTick(tick int64) {
	if ags.nextAuction == 0 {
		ags.nextAuction = tick + 500 + int64(rand.Intn(1500)) // first auction soon
	}

	if tick < ags.nextAuction {
		return
	}
	ags.nextAuction = tick + 5000 + int64(rand.Intn(8000))

	ctx := ags.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	ah := game.GetAuctionHouse()
	if ah == nil {
		return
	}

	// Don't have too many active auctions
	active := ah.GetActiveAuctions()
	if len(active) >= 3 {
		return
	}

	// Pick a random item
	template := auctionItems[rand.Intn(len(auctionItems))]
	duration := 3000 + rand.Intn(3000) // 3000-6000 ticks (~5-10 min)

	ah.CreateAuction(template.item, template.description, "", template.minBid, duration)

	game.LogEvent("event", "",
		fmt.Sprintf("🔨 AUCTION: %s — %s (starting bid: %dcr, ends in ~%d min)",
			template.item, template.description, template.minBid, duration/600))
}

// ProcessCompletedAuctions handles auction resolution.
func ProcessCompletedAuctions(completed []*entities.Player, game GameProvider) {
	// This would be called from a tick handler with the auction results
	// For now, auctions just transfer credits — item effects are logged
}
