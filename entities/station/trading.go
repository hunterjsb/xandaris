package station

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&TradingGenerator{})
}

type TradingGenerator struct{}

func (g *TradingGenerator) GetWeight() float64 {
	return 12.0 // Trading stations are very common
}

func (g *TradingGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeStation
}

func (g *TradingGenerator) GetSubType() string {
	return "Trading"
}

func (g *TradingGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*10000 + rand.Intn(1000)

	// Generate name
	prefixes := []string{"Commerce", "Trade", "Market", "Exchange"}
	suffixes := []string{"Hub", "Central", "Prime", "Station", "Complex"}
	name := fmt.Sprintf("%s %s", prefixes[rand.Intn(len(prefixes))], suffixes[rand.Intn(len(suffixes))])

	// Trading station color (gold/yellow tones)
	stationColor := color.RGBA{
		R: 220,
		G: 180,
		B: 80,
		A: 255,
	}

	// Create the station
	station := entities.NewStation(
		id,
		name,
		"Trading",
		params.OrbitDistance,
		params.OrbitAngle,
		stationColor,
	)

	// Set trading-specific properties
	station.Capacity = 2000 + rand.Intn(1000) // 2000-3000 capacity
	station.CurrentPop = rand.Intn(station.Capacity)
	station.DefenseLevel = 3 + rand.Intn(3) // 3-5 (moderate defense)

	// Trading station owners
	owners := []string{"Trade Union", "Independent", "Commerce Guild", "Merchant Alliance"}
	station.Owner = owners[rand.Intn(len(owners))]

	// Services typical for trading stations
	station.Services = []string{
		"Docking",
		"Fuel",
		"Repairs",
		"Trading Post",
		"Cargo Storage",
		"Market Access",
		"Banking",
		"Commodity Exchange",
	}

	// Trade goods for trading stations
	tradeGoods := []string{
		"Food",
		"Water",
		"Electronics",
		"Luxury Items",
		"Textiles",
		"Spices",
		"Art",
		"Consumer Goods",
		"Medical Supplies",
		"Raw Materials",
	}
	// Select 4-6 random trade goods
	numGoods := 4 + rand.Intn(3)
	station.TradeGoods = selectRandomItems(tradeGoods, numGoods)

	return station
}

// selectRandomItems picks random items from a pool
func selectRandomItems(pool []string, count int) []string {
	if count > len(pool) {
		count = len(pool)
	}

	result := make([]string, 0, count)
	used := make(map[int]bool)

	for len(result) < count {
		idx := rand.Intn(len(pool))
		if !used[idx] {
			result = append(result, pool[idx])
			used[idx] = true
		}
	}

	return result
}
