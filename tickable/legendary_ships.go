package tickable

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&LegendaryShipSystem{
		BaseSystem: NewBaseSystem("LegendaryShips", 44),
	})
}

// LegendaryShipSystem spawns unique legendary ships that can be claimed.
// Legendary ships appear as derelicts in random systems — the first faction
// to send a Scout or military ship there claims it.
//
// Legendary ships have special abilities that normal ships don't:
//   The Leviathan:    Cruiser with 2x HP and cargo hold (combat + trade)
//   Ghost Runner:     Cargo ship with 3x speed and stealth (immune to pirates)
//   Star Forge:       Mobile station that acts as a Shipyard anywhere
//   Void Walker:      Scout that can jump 3 systems at once
//   Trade King:       Cargo with 1000 capacity and auto-sell at best price
type LegendaryShipSystem struct {
	*BaseSystem
	spawned map[string]bool // legendaryName → already spawned
	nextCheck int64
}

type legendaryDef struct {
	name     string
	shipType entities.ShipType
	hp       int
	cargo    int
	attack   int
	speed    float64
	desc     string
}

var legendaries = []legendaryDef{
	{"The Leviathan", entities.ShipTypeCruiser, 700, 300, 80, 0.8, "Ancient war-trader with massive hull and cargo bay"},
	{"Ghost Runner", entities.ShipTypeCargo, 60, 800, 0, 3.0, "Stealth freighter immune to pirate raids"},
	{"Star Forge", entities.ShipTypeCargo, 500, 100, 10, 0.5, "Mobile shipyard — builds ships anywhere"},
	{"Void Walker", entities.ShipTypeScout, 80, 30, 8, 2.0, "Phase-shifting scout that jumps 3 systems"},
	{"Trade King", entities.ShipTypeCargo, 120, 1000, 5, 1.0, "Legendary merchant vessel with enormous hold"},
}

func (lss *LegendaryShipSystem) OnTick(tick int64) {
	if lss.nextCheck == 0 {
		lss.nextCheck = tick + 10000 + int64(rand.Intn(10000))
	}
	if tick < lss.nextCheck {
		return
	}
	lss.nextCheck = tick + 15000 + int64(rand.Intn(15000))

	ctx := lss.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if lss.spawned == nil {
		lss.spawned = make(map[string]bool)
	}

	systems := game.GetSystems()
	players := ctx.GetPlayers()

	// Pick a legendary that hasn't spawned yet
	var available []legendaryDef
	for _, l := range legendaries {
		if !lss.spawned[l.name] {
			available = append(available, l)
		}
	}
	if len(available) == 0 {
		return // all legendaries already spawned
	}

	legend := available[rand.Intn(len(available))]
	targetSys := systems[rand.Intn(len(systems))]

	// Check if any player has a ship in this system to claim it
	var claimer *entities.Player
	for _, p := range players {
		if p == nil {
			continue
		}
		for _, ship := range p.OwnedShips {
			if ship != nil && ship.CurrentSystem == targetSys.ID &&
				(ship.ShipType == entities.ShipTypeScout ||
					ship.ShipType == entities.ShipTypeFrigate ||
					ship.ShipType == entities.ShipTypeDestroyer ||
					ship.ShipType == entities.ShipTypeCruiser) {
				claimer = p
				break
			}
		}
		if claimer != nil {
			break
		}
	}

	if claimer == nil {
		// No one there to claim — announce the derelict location
		game.LogEvent("event", "",
			fmt.Sprintf("⭐ LEGENDARY SHIP DETECTED: %s — \"%s\" found drifting in %s! Send a Scout or military ship to claim it!",
				legend.name, legend.desc, targetSys.Name))
		return
	}

	// Instant claim!
	lss.spawned[legend.name] = true
	ship := entities.NewShip(
		rand.Intn(900000000)+100000000,
		legend.name,
		legend.shipType,
		targetSys.ID,
		claimer.Name,
		color.RGBA{255, 215, 0, 255}, // gold for legendary
	)
	ship.MaxHealth = legend.hp
	ship.CurrentHealth = legend.hp
	ship.MaxCargo = legend.cargo
	ship.AttackPower = legend.attack
	ship.Speed = legend.speed
	ship.Status = entities.ShipStatusOrbiting

	claimer.OwnedShips = append(claimer.OwnedShips, ship)
	targetSys.Entities = append(targetSys.Entities, ship)

	game.LogEvent("event", claimer.Name,
		fmt.Sprintf("⭐ %s claimed the legendary %s in %s! \"%s\"",
			claimer.Name, legend.name, targetSys.Name, legend.desc))
}
