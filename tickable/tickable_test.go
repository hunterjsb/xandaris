package tickable

import (
	"image/color"
	"testing"

	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
)

// mockGameProvider implements GameProvider for testing.
type mockGameProvider struct {
	systems        []*entities.System
	systemsMap     map[int]*entities.System
	hyperlanes     []entities.Hyperlane
	market         *economy.Market
	tradeExec      *economy.TradeExecutor
	players        []*entities.Player
	events         []mockEvent
	standingOrders []StandingOrderInfo
	deliveryMgr    *economy.DeliveryManager
}

type mockEvent struct {
	eventType, player, message string
}

func (m *mockGameProvider) GetSystems() []*entities.System              { return m.systems }
func (m *mockGameProvider) GetSystemsMap() map[int]*entities.System     { return m.systemsMap }
func (m *mockGameProvider) GetHyperlanes() []entities.Hyperlane         { return m.hyperlanes }
func (m *mockGameProvider) GetMarketEngine() *economy.Market            { return m.market }
func (m *mockGameProvider) GetTradeExecutor() *economy.TradeExecutor    { return m.tradeExec }
func (m *mockGameProvider) GetPlayers() []*entities.Player              { return m.players }
func (m *mockGameProvider) GetConnectedSystems(fromSystemID int) []int  { return nil }
func (m *mockGameProvider) StartShipJourney(ship *entities.Ship, targetSystemID int) bool {
	return false
}
func (m *mockGameProvider) LoadCargo(ship *entities.Ship, planet *entities.Planet, resource string, qty int) (int, error) {
	return 0, nil
}
func (m *mockGameProvider) UnloadCargo(ship *entities.Ship, planet *entities.Planet, resource string, qty int) (int, error) {
	return 0, nil
}
func (m *mockGameProvider) AIBuildOnPlanet(planet *entities.Planet, buildingType string, owner string, systemID int) {
}
func (m *mockGameProvider) LogEvent(eventType string, player string, message string) {
	m.events = append(m.events, mockEvent{eventType, player, message})
}
func (m *mockGameProvider) GetStandingOrderInfos() []StandingOrderInfo { return m.standingOrders }
func (m *mockGameProvider) ExecuteStandingOrderTrade(order StandingOrderInfo, player *entities.Player) error {
	return nil
}
func (m *mockGameProvider) GetDeliveryManager() *economy.DeliveryManager { return m.deliveryMgr }

// mockSystemContext implements SystemContext for testing.
type mockSystemContext struct {
	game    GameProvider
	players []*entities.Player
	tick    int64
}

func (m *mockSystemContext) GetGame() GameProvider         { return m.game }
func (m *mockSystemContext) GetPlayers() []*entities.Player { return m.players }
func (m *mockSystemContext) GetTick() int64                { return m.tick }

var white = color.RGBA{255, 255, 255, 255}

// helper: create a planet with a mine attached to a resource
func testPlanetWithMine(owner string, resType string, abundance int) *entities.Planet {
	planet := entities.NewPlanet(1, "TestPlanet", "Terrestrial", 50.0, 0, white)
	planet.Owner = owner
	planet.Population = 2000

	res := &entities.Resource{
		BaseEntity: entities.BaseEntity{
			ID:   100,
			Name: resType + " Deposit",
			Type: entities.EntityTypeResource,
		},
		ResourceType:   resType,
		Abundance:      abundance,
		ExtractionRate: 1.0,
		Owner:          owner,
	}
	planet.Resources = append(planet.Resources, res)

	mine := &entities.Building{
		BaseEntity: entities.BaseEntity{
			ID:   200,
			Name: "Mine",
			Type: entities.EntityTypeBuilding,
		},
		BuildingType:    "Mine",
		Owner:           owner,
		Level:           1,
		IsOperational:   true,
		AttachedTo:      "100",
		AttachmentType:  "Resource",
		WorkersRequired: 10,
		WorkersAssigned: 10,
		ProductionBonus: 1.0,
	}
	planet.Buildings = append(planet.Buildings, mine)

	return planet
}

// TestResourceAccumulation verifies that mines produce resources.
func TestResourceAccumulation(t *testing.T) {
	ClearRegistry()
	ras := &ResourceAccumulationSystem{
		BaseSystem: NewBaseSystem("ResourceAccumulation", 10),
	}

	planet := testPlanetWithMine("TestPlayer", "Iron", 70)
	player := &entities.Player{Name: "TestPlayer", OwnedPlanets: []*entities.Planet{planet}}

	game := &mockGameProvider{players: []*entities.Player{player}}
	ctx := &mockSystemContext{game: game, players: []*entities.Player{player}, tick: 10}
	ras.Initialize(ctx)

	initialIron := planet.GetStoredAmount("Iron")
	ras.OnTick(10) // triggers at tick%10 == 0

	newIron := planet.GetStoredAmount("Iron")
	if newIron <= initialIron {
		t.Errorf("expected iron to increase from %d, got %d", initialIron, newIron)
	}
}

// TestPopulationGrowth verifies population increases with capacity and water.
func TestPopulationGrowth(t *testing.T) {
	ClearRegistry()
	pgs := &PopulationGrowthSystem{
		BaseSystem: NewBaseSystem("PopulationGrowth", 10),
	}

	planet := entities.NewPlanet(1, "TestPlanet", "Terrestrial", 50.0, 0, white)
	planet.Owner = "TestPlayer"
	planet.Population = 2000
	planet.Happiness = 0.7
	planet.ProductivityBonus = 1.2
	// Add water so the planet doesn't starve
	planet.AddStoredResource("Water", 500)
	// Add a base + habitat for capacity
	base := &entities.Building{
		BaseEntity:          entities.BaseEntity{ID: 300, Name: "Base", Type: entities.EntityTypeBuilding},
		BuildingType:        "Base",
		Owner:               "TestPlayer",
		Level:               1,
		IsOperational:       true,
		AttachmentType:      "Planet",
		AttachedTo:          "1",
		PopulationCapacity:  5000,
		WorkersRequired:     5,
		WorkersAssigned:     5,
	}
	hab := &entities.Building{
		BaseEntity:          entities.BaseEntity{ID: 301, Name: "Habitat", Type: entities.EntityTypeBuilding},
		BuildingType:        "Habitat",
		Owner:               "TestPlayer",
		Level:               1,
		IsOperational:       true,
		AttachmentType:      "Planet",
		AttachedTo:          "1",
		PopulationCapacity:  5000,
		WorkersRequired:     5,
		WorkersAssigned:     5,
	}
	planet.Buildings = append(planet.Buildings, base, hab)

	player := &entities.Player{Name: "TestPlayer", OwnedPlanets: []*entities.Planet{planet}}
	game := &mockGameProvider{players: []*entities.Player{player}}
	ctx := &mockSystemContext{game: game, players: []*entities.Player{player}, tick: 10}
	pgs.Initialize(ctx)

	initialPop := planet.Population
	// Manually tick to trigger (priority-based counter)
	pgs.tickCounter = 9
	pgs.OnTick(10)

	if planet.Population <= initialPop {
		t.Errorf("expected population to grow from %d, got %d", initialPop, planet.Population)
	}
}

// TestPowerSystem verifies generators produce power from fuel.
func TestPowerSystem(t *testing.T) {
	ClearRegistry()
	ps := &PowerSystem{
		BaseSystem: NewBaseSystem("Power", 7),
	}

	planet := entities.NewPlanet(1, "TestPlanet", "Terrestrial", 50.0, 0, white)
	planet.Owner = "TestPlayer"
	planet.Population = 1000
	planet.AddStoredResource("Fuel", 100)

	gen := &entities.Building{
		BaseEntity:      entities.BaseEntity{ID: 400, Name: "Generator", Type: entities.EntityTypeBuilding},
		BuildingType:    "Generator",
		Owner:           "TestPlayer",
		Level:           1,
		IsOperational:   true,
		WorkersRequired: 10,
		WorkersAssigned: 10,
		ProductionBonus: 1.0,
	}
	planet.Buildings = append(planet.Buildings, gen)

	player := &entities.Player{Name: "TestPlayer", OwnedPlanets: []*entities.Planet{planet}}
	game := &mockGameProvider{players: []*entities.Player{player}}
	ctx := &mockSystemContext{game: game, players: []*entities.Player{player}, tick: 10}
	ps.Initialize(ctx)

	ps.OnTick(10)

	if planet.PowerGenerated <= 0 {
		t.Errorf("expected power generation > 0, got %f", planet.PowerGenerated)
	}
	if planet.PowerRatio <= 0 {
		t.Errorf("expected power ratio > 0, got %f", planet.PowerRatio)
	}
	// Fuel should have been consumed
	if planet.GetStoredAmount("Fuel") >= 100 {
		t.Error("expected fuel to be consumed by generator")
	}
}

// TestSequentialTickOrdering verifies systems execute in priority order.
func TestSequentialTickOrdering(t *testing.T) {
	ClearRegistry()

	var order []string
	// Register systems with different priorities
	sys1 := &orderTrackingSystem{BaseSystem: NewBaseSystem("First", 1), order: &order}
	sys2 := &orderTrackingSystem{BaseSystem: NewBaseSystem("Second", 5), order: &order}
	sys3 := &orderTrackingSystem{BaseSystem: NewBaseSystem("Third", 10), order: &order}

	RegisterSystem(sys2)
	RegisterSystem(sys3)
	RegisterSystem(sys1)

	UpdateAllSystemsSequential(1)

	if len(order) != 3 {
		t.Fatalf("expected 3 systems to execute, got %d", len(order))
	}
	if order[0] != "First" || order[1] != "Second" || order[2] != "Third" {
		t.Errorf("expected [First Second Third], got %v", order)
	}
}

// orderTrackingSystem records when it was ticked.
type orderTrackingSystem struct {
	*BaseSystem
	order *[]string
}

func (o *orderTrackingSystem) OnTick(tick int64) {
	*o.order = append(*o.order, o.GetName())
}
