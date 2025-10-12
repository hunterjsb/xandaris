package station

import (
	"fmt"
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

func (g *TradingGenerator) GetEntityType() string {
	return "Station"
}

func (g *TradingGenerator) GetSubType() string {
	return "Trading"
}

func (g *TradingGenerator) Generate(params entities.GenerationParams) interface{} {
	capacity := 2000 + rand.Intn(1000)
	return struct {
		ID            int
		Name          string
		Type          string
		OrbitDistance float64
		OrbitAngle    float64
		Capacity      int
		CurrentPop    int
		DefenseLevel  int
		Owner         string
		Services      []string
		TradeGoods    []string
	}{
		ID:            params.SystemID*10000 + 999,
		Name:          fmt.Sprintf("%s %s", []string{"Commerce", "Trade", "Market", "Exchange"}[rand.Intn(4)], []string{"Hub", "Central", "Prime", "Station"}[rand.Intn(4)]),
		Type:          "Trading",
		OrbitDistance: params.OrbitDistance,
		OrbitAngle:    params.OrbitAngle,
		Capacity:      capacity,
		CurrentPop:    rand.Intn(capacity),
		DefenseLevel:  3 + rand.Intn(3), // 3-5
		Owner:         []string{"Trade Union", "Independent", "Commerce Guild", "Merchant Alliance"}[rand.Intn(4)],
		Services:      []string{"Docking", "Fuel", "Repairs", "Trading Post", "Cargo Storage", "Market Access", "Banking"},
		TradeGoods:    []string{"Food", "Electronics", "Luxury Items", "Textiles", "Spices", "Art"},
	}
}
