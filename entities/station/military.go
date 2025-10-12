package station

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&MilitaryGenerator{})
}

type MilitaryGenerator struct{}

func (g *MilitaryGenerator) GetWeight() float64 {
	return 8.0 // Military stations are fairly common
}

func (g *MilitaryGenerator) GetEntityType() string {
	return "Station"
}

func (g *MilitaryGenerator) GetSubType() string {
	return "Military"
}

func (g *MilitaryGenerator) Generate(params entities.GenerationParams) interface{} {
	capacity := 1500 + rand.Intn(750)
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
		Name:          fmt.Sprintf("Fortress %s", []string{"Alpha", "Beta", "Prime", "Guardian"}[rand.Intn(4)]),
		Type:          "Military",
		OrbitDistance: params.OrbitDistance,
		OrbitAngle:    params.OrbitAngle,
		Capacity:      capacity,
		CurrentPop:    rand.Intn(capacity),
		DefenseLevel:  8 + rand.Intn(3), // 8-10
		Owner:         []string{"Military Corp", "Defense Alliance", "Sector Command"}[rand.Intn(3)],
		Services:      []string{"Docking", "Fuel", "Repairs", "Weapon Systems", "Fleet Command", "Training"},
		TradeGoods:    []string{"Weapons", "Armor", "Military Supplies", "Ammunition"},
	}
}
