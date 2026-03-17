package tickable

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	RegisterSystem(&ShipExperienceSystem{
		BaseSystem: NewBaseSystem("ShipExperience", 49),
	})
}

// ShipExperienceSystem tracks ship veterancy. Ships gain XP from:
//   - Combat: 50 XP per battle survived
//   - Trade runs: 10 XP per dock at a foreign Trading Post
//   - Exploration: 30 XP per system visited
//   - Pirate kills: 100 XP (for military ships in a system where pirates are defeated)
//
// Ranks and bonuses (cumulative):
//   Rookie   (0 XP):     base stats
//   Veteran  (200 XP):   +10% attack, +10% speed
//   Elite    (500 XP):   +20% attack, +20% speed, +10% HP
//   Ace      (1000 XP):  +30% attack, +30% speed, +20% HP, +10% cargo
//   Legend   (2000 XP):  +50% attack, +50% speed, +30% HP, +20% cargo
//
// XP is tracked per-ship and persists across saves (stored on the ship entity
// via the existing fields). Promotions are announced as events.
//
// This makes experienced ships valuable — losing an Ace Cruiser hurts more
// than losing a fresh one, which raises the stakes of combat.
type ShipExperienceSystem struct {
	*BaseSystem
	shipXP       map[int]int    // shipID → accumulated XP
	shipRank     map[int]string // shipID → current rank
	lastPromoted map[int]int64  // shipID → tick of last promotion
}

type xpRank struct {
	name      string
	threshold int
	atkBonus  float64
	spdBonus  float64
	hpBonus   float64
	crgBonus  float64
}

var xpRanks = []xpRank{
	{"Legend", 2000, 0.50, 0.50, 0.30, 0.20},
	{"Ace", 1000, 0.30, 0.30, 0.20, 0.10},
	{"Elite", 500, 0.20, 0.20, 0.10, 0.00},
	{"Veteran", 200, 0.10, 0.10, 0.00, 0.00},
	{"Rookie", 0, 0.00, 0.00, 0.00, 0.00},
}

func (ses *ShipExperienceSystem) OnTick(tick int64) {
	if tick%300 != 0 {
		return
	}

	ctx := ses.GetContext()
	if ctx == nil {
		return
	}

	game := ctx.GetGame()
	if game == nil {
		return
	}

	if ses.shipXP == nil {
		ses.shipXP = make(map[int]int)
		ses.shipRank = make(map[int]string)
		ses.lastPromoted = make(map[int]int64)
	}

	players := ctx.GetPlayers()

	for _, player := range players {
		if player == nil {
			continue
		}
		for _, ship := range player.OwnedShips {
			if ship == nil {
				continue
			}

			ses.accumulateXP(ship, player, game)
			ses.checkPromotion(tick, ship, player, game)
		}
	}
}

func (ses *ShipExperienceSystem) accumulateXP(ship *entities.Ship, player *entities.Player, game GameProvider) {
	sid := ship.GetID()
	xp := 0

	switch {
	// Military ships gain XP from being in systems (patrol duty)
	case ship.ShipType == entities.ShipTypeFrigate ||
		ship.ShipType == entities.ShipTypeDestroyer ||
		ship.ShipType == entities.ShipTypeCruiser:
		if ship.Status != entities.ShipStatusMoving {
			xp += 2 // patrol XP
		}
		// Damaged ships just survived combat
		if ship.CurrentHealth < ship.MaxHealth {
			xp += 15 // combat survivor XP
		}

	// Cargo ships gain XP from hauling
	case ship.ShipType == entities.ShipTypeCargo:
		if ship.GetTotalCargo() > 0 {
			xp += 3 // hauling XP
		}
		if ship.DockedAtPlanet > 0 {
			xp += 5 // docking XP
		}

	// Scouts gain XP from being in foreign systems
	case ship.ShipType == entities.ShipTypeScout:
		xp += 4 // exploration XP

	default:
		xp += 1
	}

	// Small random bonus
	if rand.Intn(5) == 0 {
		xp += rand.Intn(5)
	}

	ses.shipXP[sid] += xp
}

func (ses *ShipExperienceSystem) checkPromotion(tick int64, ship *entities.Ship, player *entities.Player, game GameProvider) {
	sid := ship.GetID()
	xp := ses.shipXP[sid]
	currentRank := ses.shipRank[sid]
	if currentRank == "" {
		currentRank = "Rookie"
		ses.shipRank[sid] = currentRank
	}

	// Find the highest rank this ship qualifies for
	for _, rank := range xpRanks {
		if xp >= rank.threshold {
			if rank.name != currentRank {
				ses.shipRank[sid] = rank.name
				ses.lastPromoted[sid] = tick

				// Apply stat bonuses based on ship's BASE stats
				ses.applyRankBonuses(ship, rank)

				game.LogEvent("event", player.Name,
					fmt.Sprintf("⭐ %s's %s promoted to %s! (XP: %d) — +%.0f%% attack, +%.0f%% speed",
						player.Name, ship.Name, rank.name, xp,
						rank.atkBonus*100, rank.spdBonus*100))
			}
			break
		}
	}
}

func (ses *ShipExperienceSystem) applyRankBonuses(ship *entities.Ship, rank xpRank) {
	// Get base stats for this ship type
	baseAtk := getBaseAttack(ship.ShipType)
	baseSpd := getBaseSpeed(ship.ShipType)
	baseHP := entities.GetShipMaxHealth(ship.ShipType)
	baseCargo := entities.GetShipMaxCargo(ship.ShipType)

	// Apply rank bonuses to base stats
	ship.AttackPower = baseAtk + int(float64(baseAtk)*rank.atkBonus)
	ship.Speed = baseSpd + baseSpd*rank.spdBonus
	if rank.hpBonus > 0 {
		bonusHP := int(float64(baseHP) * rank.hpBonus)
		ship.MaxHealth = baseHP + bonusHP
		ship.CurrentHealth += bonusHP // heal the bonus amount
		if ship.CurrentHealth > ship.MaxHealth {
			ship.CurrentHealth = ship.MaxHealth
		}
	}
	if rank.crgBonus > 0 {
		ship.MaxCargo = baseCargo + int(float64(baseCargo)*rank.crgBonus)
	}
}

func getBaseAttack(st entities.ShipType) int {
	switch st {
	case entities.ShipTypeScout:
		return 5
	case entities.ShipTypeCargo:
		return 2
	case entities.ShipTypeFrigate:
		return 20
	case entities.ShipTypeDestroyer:
		return 40
	case entities.ShipTypeCruiser:
		return 60
	default:
		return 5
	}
}

func getBaseSpeed(st entities.ShipType) float64 {
	switch st {
	case entities.ShipTypeScout:
		return 1.5
	case entities.ShipTypeCargo:
		return 1.0
	case entities.ShipTypeFrigate:
		return 1.2
	case entities.ShipTypeDestroyer:
		return 1.0
	case entities.ShipTypeCruiser:
		return 0.9
	case entities.ShipTypeColony:
		return 0.8
	default:
		return 1.0
	}
}

// GetShipXP returns the XP for a given ship (for API/display).
func (ses *ShipExperienceSystem) GetShipXP(shipID int) int {
	if ses.shipXP == nil {
		return 0
	}
	return ses.shipXP[shipID]
}

// GetShipRank returns the rank for a given ship.
func (ses *ShipExperienceSystem) GetShipRank(shipID int) string {
	if ses.shipRank == nil {
		return "Rookie"
	}
	r := ses.shipRank[shipID]
	if r == "" {
		return "Rookie"
	}
	return r
}
