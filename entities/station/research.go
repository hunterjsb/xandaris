package station

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&ResearchGenerator{})
}

type ResearchGenerator struct{}

func (g *ResearchGenerator) GetWeight() float64 {
	return 7.0 // Research stations are moderately common
}

func (g *ResearchGenerator) GetEntityType() entities.EntityType {
	return entities.EntityTypeStation
}

func (g *ResearchGenerator) GetSubType() string {
	return "Research"
}

func (g *ResearchGenerator) Generate(params entities.GenerationParams) entities.Entity {
	// Generate ID
	id := params.SystemID*10000 + rand.Intn(1000)

	// Generate name
	prefixes := []string{"Discovery", "Insight", "Laboratory", "Observatory", "Science", "Research"}
	suffixes := []string{"Station", "Complex", "Institute", "Facility", "Center", "Hub"}
	name := fmt.Sprintf("%s %s", prefixes[rand.Intn(len(prefixes))], suffixes[rand.Intn(len(suffixes))])

	// Research station color (blue/cyan tones)
	stationColor := color.RGBA{
		R: 60,
		G: 140,
		B: 220,
		A: 255,
	}

	// Create the station
	station := entities.NewStation(
		id,
		name,
		"Research",
		params.OrbitDistance,
		params.OrbitAngle,
		stationColor,
	)

	// Set research-specific properties
	station.Capacity = 800 + rand.Intn(800) // 800-1600 capacity (smaller, focused on research)
	station.CurrentPop = rand.Intn(station.Capacity)
	station.DefenseLevel = 1 + rand.Intn(3) // 1-3 (low defense, focused on science)

	// Research station owners
	owners := []string{"Research Guild", "Scientific Consortium", "Academy of Sciences", "Independent Researchers", "Tech Institute"}
	station.Owner = owners[rand.Intn(len(owners))]

	// Services typical for research stations
	station.Services = []string{
		"Docking",
		"Fuel",
		"Repairs",
		"Laboratory Access",
		"Data Analysis",
		"Prototype Testing",
		"Academic Library",
		"Specimen Storage",
		"Quantum Computing",
	}

	// Trade goods for research stations (scientific equipment and data)
	tradeGoods := []string{
		"Scientific Equipment",
		"Data Cores",
		"Rare Elements",
		"Prototypes",
		"Research Papers",
		"Experimental Tech",
		"Biological Samples",
		"Sensor Arrays",
		"Computing Modules",
		"Lab Supplies",
	}
	// Select 3-5 random trade goods
	numGoods := 3 + rand.Intn(3)
	station.TradeGoods = selectRandomItems(tradeGoods, numGoods)

	return station
}
