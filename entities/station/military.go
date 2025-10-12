package station

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&MilitaryGenerator{})
}

type MilitaryGenerator struct{}

func (g *MilitaryGenerator) GetWeight() float64 {
	return 6.0 // Military stations are less common than trading
}

func (g *MilitaryGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeStation
}

func (g *MilitaryGenerator) GetSubType() string {
	return "Military"
}

func (g *MilitaryGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*10000 + rand.Intn(1000)

	// Generate name
	prefixes := []string{"Fortress", "Guardian", "Sentinel", "Bastion", "Aegis", "Citadel"}
	suffixes := []string{"Alpha", "Prime", "One", "Station", "Outpost", "Command"}
	name := fmt.Sprintf("%s %s", prefixes[rand.Intn(len(prefixes))], suffixes[rand.Intn(len(suffixes))])

	// Military station color (red/grey tones)
	stationColor := color.RGBA{
		R: 180,
		G: 50,
		B: 50,
		A: 255,
	}

	// Create the station
	station := entities.NewStation(
		id,
		name,
		"Military",
		params.OrbitDistance,
		params.OrbitAngle,
		stationColor,
	)

	// Set military-specific properties
	station.Capacity = 1500 + rand.Intn(1000) // 1500-2500 capacity (smaller but more fortified)
	station.CurrentPop = rand.Intn(station.Capacity)
	station.DefenseLevel = 8 + rand.Intn(3) // 8-10 (very high defense)

	// Military station owners
	owners := []string{"Military Corp", "Defense Coalition", "Fleet Command", "Sector Defense Force"}
	station.Owner = owners[rand.Intn(len(owners))]

	// Services typical for military stations
	station.Services = []string{
		"Docking",
		"Fuel",
		"Repairs",
		"Weapon Systems",
		"Fleet Command",
		"Intelligence",
		"Training Facilities",
		"Tactical Analysis",
	}

	// Trade goods for military stations (weapons and military equipment)
	tradeGoods := []string{
		"Weapons",
		"Armor",
		"Military Supplies",
		"Ammunition",
		"Defense Systems",
		"Tactical Equipment",
		"Military Rations",
		"Combat Drones",
	}
	// Select 3-5 random trade goods
	numGoods := 3 + rand.Intn(3)
	station.TradeGoods = selectRandomItems(tradeGoods, numGoods)

	return station
}
