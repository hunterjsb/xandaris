package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&AlienEncounterSystem{
		BaseSystem: NewBaseSystem("AlienEncounters", 74),
	})
}

// AlienEncounterSystem generates encounters with NPC alien species
// that exist outside the player factions. These aliens offer unique
// trade opportunities, challenges, and story events.
//
// Alien species:
//   The Traders of Zyl: appear offering exotic resource bundles at
//     premium prices. Buy = get rare resources instantly.
//
//   The Watchers: observe a system, then offer tech knowledge if
//     the faction has maintained peace. War = they leave.
//
//   The Void Born: aggressive scavengers that challenge military ships.
//     Defeat them = salvage alien tech (+0.3 tech). Lose = ship damaged.
//
//   The Ancients: appear near planets with tech 4.0+. Offer a test:
//     sacrifice 10,000 credits → receive a permanent production bonus.
//
// Each encounter is a one-time event. Responses are automatic based
// on faction's capabilities (has military? has credits? has tech?).
type AlienEncounterSystem struct {
	*BaseSystem
	nextEncounter int64
	encountered   map[string]int // encounterType → count
}

func (aes *AlienEncounterSystem) OnTick(tick int64) {
	ctx := aes.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if aes.encountered == nil {
		aes.encountered = make(map[string]int)
	}

	if aes.nextEncounter == 0 {
		aes.nextEncounter = tick + 5000 + int64(rand.Intn(10000))
	}
	if tick < aes.nextEncounter {
		return
	}
	aes.nextEncounter = tick + 10000 + int64(rand.Intn(15000))

	players := ctx.GetPlayers()
	systems := game.GetSystems()

	encounterType := rand.Intn(4)
	switch encounterType {
	case 0:
		aes.tradersOfZyl(players, systems, game)
	case 1:
		aes.theWatchers(players, systems, game)
	case 2:
		aes.voidBorn(players, systems, game)
	case 3:
		aes.theAncients(players, systems, game)
	}
}

func (aes *AlienEncounterSystem) tradersOfZyl(players []*entities.Player, systems []*entities.System, game GameProvider) {
	// Find a faction with credits to trade
	var buyer *entities.Player
	for _, p := range players {
		if p != nil && p.Credits > 5000 {
			if buyer == nil || p.Credits > buyer.Credits {
				buyer = p
			}
		}
	}
	if buyer == nil {
		return
	}

	sys := systems[rand.Intn(len(systems))]
	cost := 3000 + rand.Intn(5000)

	// Find buyer's planet to deposit resources
	for _, e := range sys.Entities {
		if planet, ok := e.(*entities.Planet); ok && planet.Owner == buyer.Name {
			if buyer.Credits >= cost {
				buyer.Credits -= cost
				// Deliver exotic resource bundle
				planet.AddStoredResource(entities.ResRareMetals, 50+rand.Intn(50))
				planet.AddStoredResource(entities.ResElectronics, 20+rand.Intn(30))
				planet.AddStoredResource(entities.ResHelium3, 30+rand.Intn(30))

				game.LogEvent("event", buyer.Name,
					fmt.Sprintf("👽 Traders of Zyl appeared in %s! %s purchased an exotic resource bundle for %dcr (+RM, +Electronics, +He-3)",
						sys.Name, buyer.Name, cost))
				return
			}
		}
	}

	game.LogEvent("event", "",
		fmt.Sprintf("👽 The Traders of Zyl passed through %s, but found no interested buyers...",
			sys.Name))
}

func (aes *AlienEncounterSystem) theWatchers(players []*entities.Player, systems []*entities.System, game GameProvider) {
	sys := systems[rand.Intn(len(systems))]

	// Check for peaceful faction in this system
	for _, e := range sys.Entities {
		planet, ok := e.(*entities.Planet)
		if !ok || planet.Owner == "" {
			continue
		}

		// Peaceful = happiness > 0.6 and no military ships
		if planet.Happiness < 0.6 {
			continue
		}

		hasMilitary := false
		for _, p := range players {
			if p == nil || p.Name != planet.Owner {
				continue
			}
			for _, ship := range p.OwnedShips {
				if ship != nil && ship.CurrentSystem == sys.ID &&
					(ship.ShipType == entities.ShipTypeFrigate ||
						ship.ShipType == entities.ShipTypeDestroyer ||
						ship.ShipType == entities.ShipTypeCruiser) {
					hasMilitary = true
					break
				}
			}
			break
		}

		if !hasMilitary {
			// Watchers gift tech
			planet.TechLevel += 0.2
			game.LogEvent("event", planet.Owner,
				fmt.Sprintf("👁️ The Watchers observed %s in %s and found it peaceful. They share ancient knowledge — tech +0.2 (now %.1f)",
					planet.Name, sys.Name, planet.TechLevel))
			return
		}
	}

	game.LogEvent("event", "",
		fmt.Sprintf("👁️ The Watchers appeared in %s, observed the military presence, and silently departed...",
			sys.Name))
}

func (aes *AlienEncounterSystem) voidBorn(players []*entities.Player, systems []*entities.System, game GameProvider) {
	sys := systems[rand.Intn(len(systems))]

	// Find military ships to challenge
	for _, p := range players {
		if p == nil {
			continue
		}
		for _, ship := range p.OwnedShips {
			if ship == nil || ship.CurrentSystem != sys.ID || ship.Status == entities.ShipStatusMoving {
				continue
			}
			if ship.ShipType != entities.ShipTypeFrigate &&
				ship.ShipType != entities.ShipTypeDestroyer &&
				ship.ShipType != entities.ShipTypeCruiser {
				continue
			}

			// Battle! 60% win chance for destroyers+, 40% for frigates
			winChance := 40
			if ship.ShipType == entities.ShipTypeDestroyer || ship.ShipType == entities.ShipTypeCruiser {
				winChance = 60
			}

			if rand.Intn(100) < winChance {
				// Victory — alien tech salvage
				p.Credits += 3000 + rand.Intn(5000)
				for _, s := range systems {
					for _, e := range s.Entities {
						if planet, ok := e.(*entities.Planet); ok && planet.Owner == p.Name {
							planet.TechLevel += 0.3
							game.LogEvent("event", p.Name,
								fmt.Sprintf("⚔️ %s's %s defeated a Void Born raider in %s! Salvaged alien tech (+0.3 tech on %s) +%dcr",
									p.Name, ship.Name, sys.Name, planet.Name, 3000))
							return
						}
					}
				}
			} else {
				// Defeat — ship damaged
				ship.CurrentHealth = ship.MaxHealth / 3
				game.LogEvent("event", p.Name,
					fmt.Sprintf("⚔️ %s's %s was overwhelmed by Void Born raiders in %s! Ship badly damaged (HP: %d/%d)",
						p.Name, ship.Name, sys.Name, ship.CurrentHealth, ship.MaxHealth))
			}
			return
		}
	}

	game.LogEvent("event", "",
		fmt.Sprintf("👾 Void Born raiders detected passing through %s — no military ships present to engage",
			sys.Name))
}

func (aes *AlienEncounterSystem) theAncients(players []*entities.Player, systems []*entities.System, game GameProvider) {
	// Find high-tech planets
	for _, sys := range systems {
		for _, e := range sys.Entities {
			planet, ok := e.(*entities.Planet)
			if !ok || planet.Owner == "" || planet.TechLevel < 3.5 {
				continue
			}

			for _, p := range players {
				if p == nil || p.Name != planet.Owner {
					continue
				}
				if p.Credits >= 10000 {
					// The test: sacrifice credits for permanent bonus
					p.Credits -= 10000
					planet.ProductivityBonus += 0.2
					game.LogEvent("event", p.Name,
						fmt.Sprintf("🌌 The Ancients appeared before %s on %s! They accepted an offering of 10,000cr and blessed the planet with +20%% permanent productivity",
							p.Name, planet.Name))
					return
				}
				break
			}
		}
	}

	game.LogEvent("event", "",
		"🌌 The Ancients briefly stirred in the void, but found no civilization worthy of their attention...")
}
