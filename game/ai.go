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

	if !PlanetHasTradingPost(planet) {
		AddTradingPostToPlanet(planet, player.Name, systemID)
	}

	SeedInitialCommodities(planet, player.Name)

	// Build mines on all owned resources (AI gets productive immediately)
	if buildMines {
		BuildMinesOnResources(planet, player.Name, systemID)
	}

	// Give AI a starting refinery so they produce Fuel
	if buildMines {
		AddBuildingToPlanet(planet, "Refinery", player.Name, systemID)
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
		if gen.GetSubType() == "Mine" {
			mineGen = gen
			break
		}
	}
	if mineGen == nil {
		return
	}

	// Build mines on essential deposits — Water (survival), Iron (building), Oil (refining)
	targetTypes := map[string]bool{"Water": true, "Iron": true, "Oil": true}

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

// PlanetHasTradingPost checks if a planet has a Trading Post building.
func PlanetHasTradingPost(planet *entities.Planet) bool {
	for _, entity := range planet.Buildings {
		if building, ok := entity.(*entities.Building); ok {
			if building.BuildingType == "Trading Post" {
				return true
			}
		}
	}
	return false
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
	AddBuildingToPlanet(planet, "Trading Post", owner, systemID)
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
