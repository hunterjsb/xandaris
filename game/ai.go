package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/utils"
)

const (
	DefaultAIPlayerCount = 5
)

var (
	AIPlayerNames = []string{
		"Llama Logistics",      // Meta Llama
		"DeepSeek Ventures",    // DeepSeek
		"Gemini Exchange",      // Google Gemini
		"Grok Industries",      // xAI Grok
		"Opus Cartel",          // Anthropic Claude
		"Mistral Trading Co.",  // Mistral AI
	}
)

// InitializeAIPlayers seeds the galaxy with AI-controlled factions.
func InitializeAIPlayers(state *State) {
	availableColors := utils.GetAIPlayerColors()
	if len(availableColors) == 0 {
		availableColors = []color.RGBA{utils.PlayerGreen}
	}

	nextID := len(state.Players)
	rand.Seed(time.Now().UnixNano())

	aiCount := DefaultAIPlayerCount
	colorCount := len(availableColors)

	for i := 0; i < aiCount; i++ {
		name := AIPlayerNames[i%len(AIPlayerNames)]
		if i >= len(AIPlayerNames) {
			name = fmt.Sprintf("Frontier Syndicate %d", i+1)
		}

		playerColor := availableColors[i%colorCount]
		aiPlayer := entities.NewPlayer(nextID+i, name, playerColor, entities.PlayerTypeAI)

		entities.InitializePlayer(aiPlayer, state.Systems)
		if aiPlayer.HomePlanet == nil {
			continue
		}

		state.Players = append(state.Players, aiPlayer)
		PrepareHomeworld(aiPlayer, !aiPlayer.IsHuman())
		if aiPlayer.HomePlanet != nil {
			fmt.Printf("[AI] %s established trade hub on %s\n", aiPlayer.Name, aiPlayer.HomePlanet.Name)
		}
	}
}

// PrepareHomeworld gives a player Trading Post, initial commodities, and optionally mines + cargo ship.
func PrepareHomeworld(player *entities.Player, buildMines bool) {
	if player == nil || player.HomePlanet == nil {
		return
	}

	planet := player.HomePlanet
	systemID := 0
	if player.HomeSystem != nil {
		systemID = player.HomeSystem.ID
	}

	if !planet.HasBuilding(entities.BuildingTradingPost) {
		AddTradingPostToPlanet(planet, player.Name, systemID)
	}

	// Ensure home planet has key resources for production chains
	EnsureResourceDeposit(planet, "Rare Metals", player.Name) // Factory: RM+Iron → Electronics
	EnsureResourceDeposit(planet, "Helium-3", player.Name)    // Fusion Reactor: He-3 → 200MW

	SeedInitialCommodities(planet, player.Name)

	// Build mines on all owned resources (AI gets productive immediately)
	if buildMines {
		BuildMinesOnResources(planet, player.Name, systemID)
	}

	// Give AI a starting refinery so they produce Fuel + a generator for power
	if buildMines {
		AddBuildingToPlanet(planet, entities.BuildingRefinery, player.Name, systemID)
		AddBuildingToPlanet(planet, entities.BuildingGenerator, player.Name, systemID)
	}

	// Give non-human players a starting cargo ship for logistics
	if !player.IsHuman() && player.HomeSystem != nil {
		cargoShip := entities.NewShip(
			2000+player.ID,
			player.Name+" Hauler",
			entities.ShipTypeCargo,
			player.HomeSystem.ID,
			player.Name,
			player.Color,
		)
		cargoShip.OrbitDistance = planet.GetOrbitDistance()
		cargoShip.OrbitAngle = rand.Float64() * 2 * math.Pi
		cargoShip.Status = entities.ShipStatusOrbiting
		player.HomeSystem.AddEntity(cargoShip)
		player.AddOwnedShip(cargoShip)
	}
}

// BuildMinesOnResources creates mines on Water and Iron deposits (the essentials).
// Other resources get mined later by the AI building system when prices rise.
func BuildMinesOnResources(planet *entities.Planet, owner string, systemID int) {
	generators := entities.GetGeneratorsByType(entities.EntityTypeBuilding)
	var mineGen entities.EntityGenerator
	for _, gen := range generators {
		if gen.GetSubType() == entities.BuildingMine {
			mineGen = gen
			break
		}
	}
	if mineGen == nil {
		return
	}

	// Build mines on essential deposits — Water (survival), Iron (building), Oil (refining)
	targetTypes := map[string]bool{entities.ResWater: true, entities.ResIron: true, entities.ResOil: true}

	for _, resourceEntity := range planet.Resources {
		resource, ok := resourceEntity.(*entities.Resource)
		if !ok || resource.Owner != owner {
			continue
		}
		if !targetTypes[resource.ResourceType] {
			continue
		}

		params := entities.GenerationParams{
			SystemID:     systemID,
			OrbitDistance: resource.GetOrbitDistance(),
			OrbitAngle:   resource.GetOrbitAngle(),
			SystemSeed:   time.Now().UnixNano(),
		}

		buildingEntity := mineGen.Generate(params)
		if mine, ok := buildingEntity.(*entities.Building); ok {
			mine.Owner = owner
			mine.AttachedTo = fmt.Sprintf("%d", resource.GetID())
			mine.AttachmentType = "Resource"
			mine.IsOperational = true
			planet.Buildings = append(planet.Buildings, mine)
		}
	}

	planet.RebalanceWorkforce()
}

// GetBuildingCost returns the credit cost for a building type by querying the generator.
func GetBuildingCost(buildingType string) int {
	generators := entities.GetGeneratorsByType(entities.EntityTypeBuilding)
	for _, gen := range generators {
		if gen.GetSubType() == buildingType {
			// Generate a temporary building to read its cost
			params := entities.GenerationParams{SystemID: 0, SystemSeed: 1}
			if b, ok := gen.Generate(params).(*entities.Building); ok {
				return b.BuildCost
			}
		}
	}
	return 500 // default fallback
}

// AddTradingPostToPlanet creates and attaches a Trading Post to a planet.
func AddTradingPostToPlanet(planet *entities.Planet, owner string, systemID int) {
	AddBuildingToPlanet(planet, entities.BuildingTradingPost, owner, systemID)
}

// EnsureResourceDeposit adds a resource deposit to a planet if it doesn't have one of that type.
func EnsureResourceDeposit(planet *entities.Planet, resType string, owner string) {
	for _, res := range planet.Resources {
		if r, ok := res.(*entities.Resource); ok && r.ResourceType == resType {
			return // Already has this type
		}
	}

	// Create a new deposit
	deposit := &entities.Resource{
		BaseEntity: entities.BaseEntity{
			ID:           rand.Intn(100000) + 900000,
			Name:         fmt.Sprintf("%s Deposit", resType),
			Type:         entities.EntityTypeResource,
			SubType:      resType,
			Color:        entities.ResourceColor(resType),
			OrbitDistance: 6 + rand.Float64()*4,
			OrbitAngle:   rand.Float64() * 2 * math.Pi,
		},
		ResourceType:   resType,
		Abundance:      40 + rand.Intn(30),
		ExtractionRate: math.Round((0.5+rand.Float64()*0.5)*10) / 10,
		Rarity:         "Uncommon",
		Size:           3,
		Quality:        50 + rand.Intn(40),
		Owner:          owner,
		NodePosition:   rand.Float64() * 2 * math.Pi,
	}

	planet.Resources = append(planet.Resources, deposit)
}

// AddBuildingToPlanet creates and attaches a building of the given type to a planet.
func AddBuildingToPlanet(planet *entities.Planet, buildingType string, owner string, systemID int) {
	generators := entities.GetGeneratorsByType(entities.EntityTypeBuilding)
	var gen entities.EntityGenerator
	for _, g := range generators {
		if g.GetSubType() == buildingType {
			gen = g
			break
		}
	}
	if gen == nil {
		return
	}

	params := entities.GenerationParams{
		SystemID:     systemID,
		OrbitDistance: 12 + rand.Float64()*6,
		OrbitAngle:   rand.Float64() * 2 * math.Pi,
		SystemSeed:   time.Now().UnixNano(),
	}

	buildingEntity := gen.Generate(params)
	if building, ok := buildingEntity.(*entities.Building); ok {
		building.Owner = owner
		building.AttachedTo = fmt.Sprintf("%d", planet.GetID())
		building.AttachmentType = "Planet"
		building.IsOperational = true
		planet.Buildings = append(planet.Buildings, building)
		planet.RebalanceWorkforce()
	}
}

// SeedInitialCommodities gives a planet modest starting resources.
func SeedInitialCommodities(planet *entities.Planet, owner string) {
	if planet == nil {
		return
	}

	// Seed from natural resources on the planet (small starting buffer)
	for _, resourceEntity := range planet.Resources {
		if resource, ok := resourceEntity.(*entities.Resource); ok {
			if resource.Owner != owner {
				continue
			}
			amount := 50 + rand.Intn(50)
			planet.AddStoredResource(resource.ResourceType, amount)
		}
	}

	// Seed all resource types so consumption creates demand for everything.
	// Rare resources get more seeding since they can't be produced on most planets.
	essentials := map[string]int{
		"Water":       150,
		"Iron":        80,
		"Oil":         60,
		"Fuel":        30,
		"Rare Metals": 100,
		"Helium-3":    80,
	}
	for res, base := range essentials {
		if planet.GetStoredAmount(res) < base/2 {
			planet.AddStoredResource(res, base+rand.Intn(base/2))
		}
	}
}
