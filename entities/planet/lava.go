package planet

import (
	"fmt"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

func init() {
	entities.RegisterGenerator(&LavaGenerator{})
}

type LavaGenerator struct{}

func (g *LavaGenerator) GetWeight() float64 {
	return 5.0 // Lava planets are less common
}

func (g *LavaGenerator) GetEntityType() string {
	return "Planet"
}

func (g *LavaGenerator) GetSubType() string {
	return "Lava"
}

func (g *LavaGenerator) Generate(params entities.GenerationParams) interface{} {
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
		Type:          "Lava",
		OrbitDistance: params.OrbitDistance,
		OrbitAngle:    params.OrbitAngle,
		Temperature:   800 + rand.Intn(500), // 800-1300°C
		Atmosphere:    "Corrosive",
		Population:    0, // Uninhabitable
		Habitability:  0,
		Size:          4 + rand.Intn(3), // 4-6
	}
}
