package station

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&MiningGenerator{})
}

type MiningGenerator struct{}

func (g *MiningGenerator) GetWeight() float64 {
	return 8.0 // Mining stations are moderately common
}

func (g *MiningGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeStation
}

func (g *MiningGenerator) GetSubType() string {
	return "Mining"
}

func (g *MiningGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*10000 + rand.Intn(1000)

	// Generate name
	prefixes := []string{"Excavator", "Harvester", "Extractor", "Drill", "Prospector", "Miner"}
	suffixes := []string{"Station", "Outpost", "Facility", "Complex", "Base", "Platform"}
	name := fmt.Sprintf("%s %s", prefixes[rand.Intn(len(prefixes))], suffixes[rand.Intn(len(suffixes))])

	// Mining station color (brown/grey metallic tones)
	stationColor := color.RGBA{
		R: 140,
		G: 120,
		B: 100,
		A: 255,
	}

	// Create the station
	station := entities.NewStation(
		id,
		name,
		"Mining",
		params.OrbitDistance,
		params.OrbitAngle,
		stationColor,
	)

	// Set mining-specific properties
	station.Capacity = 1200 + rand.Intn(800) // 1200-2000 capacity (industrial workforce)
	station.CurrentPop = rand.Intn(station.Capacity)
	station.DefenseLevel = 2 + rand.Intn(3) // 2-4 (low defense, industrial focus)

	// Mining station owners
	owners := []string{"Mining Consortium", "Independent Miners", "Resource Corp", "Asteroid Mining Guild", "Industrial Alliance"}
	station.Owner = owners[rand.Intn(len(owners))]

	// Services typical for mining stations
	station.Services = []string{
		"Docking",
		"Fuel",
		"Repairs",
		"Ore Processing",
		"Equipment Rental",
		"Surveying",
		"Cargo Transport",
		"Refinery Access",
		"Mineral Analysis",
	}

	// Trade goods for mining stations (raw materials and equipment)
	tradeGoods := []string{
		"Raw Ore",
		"Precious Metals",
		"Industrial Minerals",
		"Crystals",
		"Rare Earth Elements",
		"Iron",
		"Copper",
		"Titanium",
		"Mining Equipment",
		"Excavation Tools",
		"Ore Containers",
	}
	// Select 4-6 random trade goods
	numGoods := 4 + rand.Intn(3)
	station.TradeGoods = selectRandomItems(tradeGoods, numGoods)

	return station
}
