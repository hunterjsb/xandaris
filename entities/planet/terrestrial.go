package planet

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&TerrestrialGenerator{})
}

type TerrestrialGenerator struct{}

func (g *TerrestrialGenerator) GetWeight() float64 {
	return 15.0 // Terrestrial planets are common
}

func (g *TerrestrialGenerator) GetEntityType() string {
	return "Planet"
}

func (g *TerrestrialGenerator) GetSubType() string {
	return "Terrestrial"
}

func (g *TerrestrialGenerator) Generate(params entities.GenerationParams) interface{} {
	// Placeholder: This will create a Planet struct once we refactor
	// For now, returning a simple struct with the data
	return struct {
		ID            int
		Name          string
		Type          string
		OrbitDistance float64
		OrbitAngle    float64
		Temperature   int
		Atmosphere    string
		Population    int64
		Habitability  int
		Size          int
	}{
		ID:            params.SystemID*1000 + rand.Intn(1000),
		Name:          fmt.Sprintf("Planet %d", rand.Intn(100)),
		Type:          "Terrestrial",
		OrbitDistance: params.OrbitDistance,
		OrbitAngle:    params.OrbitAngle,
		Temperature:   -20 + rand.Intn(60), // -20 to 40°C
		Atmosphere:    []string{"Breathable", "Toxic", "Thin"}[rand.Intn(3)],
		Population:    int64(rand.Intn(2000000000)),
		Habitability:  60 + rand.Intn(30), // 60-90%
		Size:          5 + rand.Intn(3),   // 5-7
	}
}
