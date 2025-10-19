package core

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
	defaultAIPlayerCount = 3
)

var (
	aiPlayerNames = []string{
		"Orion Exchange",
		"Lyra Cartel",
		"Helios Commodities",
		"Ceres Brokers",
		"Nova Frontier Co.",
		"Zenith Trade League",
	}
)

// initializeAIPlayers seeds the galaxy with a set of AI-controlled factions
func (a *App) initializeAIPlayers() {
	availableColors := utils.GetAIPlayerColors()
	if len(availableColors) == 0 {
		availableColors = []color.RGBA{utils.PlayerGreen}
	}

	nextID := len(a.state.Players)
	rand.Seed(time.Now().UnixNano())

	aiCount := defaultAIPlayerCount
	colorCount := len(availableColors)

	for i := 0; i < aiCount; i++ {
		name := aiPlayerNames[i%len(aiPlayerNames)]
		if i >= len(aiPlayerNames) {
			name = fmt.Sprintf("Frontier Syndicate %d", i+1)
		}

		playerColor := availableColors[i%colorCount]

		aiPlayer := entities.NewPlayer(nextID+i, name, playerColor, entities.PlayerTypeAI)

		entities.InitializePlayer(aiPlayer, a.state.Systems)
		if aiPlayer.HomePlanet == nil {
			continue // Skip factions that couldn't secure a colony
		}

		a.state.Players = append(a.state.Players, aiPlayer)
		a.prepareAIHomeworld(aiPlayer)
		if aiPlayer.HomePlanet != nil {
			fmt.Printf("[AI] %s established trade hub on %s\n", aiPlayer.Name, aiPlayer.HomePlanet.Name)
		}
	}
}

// prepareAIHomeworld ensures an AI starts with infrastructure and tradable goods
func (a *App) prepareAIHomeworld(player *entities.Player) {
	if player == nil || player.HomePlanet == nil {
		return
	}

	planet := player.HomePlanet
	systemID := 0
	if player.HomeSystem != nil {
		systemID = player.HomeSystem.ID
	}

	if !planetHasTradingPost(planet) {
		addTradingPostToPlanet(planet, player.Name, systemID)
	}

	seedInitialCommodities(planet, player.Name)
}

func planetHasTradingPost(planet *entities.Planet) bool {
	for _, entity := range planet.Buildings {
		if building, ok := entity.(*entities.Building); ok {
			if building.BuildingType == "Trading Post" {
				return true
			}
		}
	}
	return false
}

func addTradingPostToPlanet(planet *entities.Planet, owner string, systemID int) {
	generators := entities.GetGeneratorsByType(entities.EntityTypeBuilding)
	var tradingPostGen entities.EntityGenerator
	for _, gen := range generators {
		if gen.GetSubType() == "Trading Post" {
			tradingPostGen = gen
			break
		}
	}
	if tradingPostGen == nil {
		return
	}

	params := entities.GenerationParams{
		SystemID:      systemID,
		OrbitDistance: 12 + rand.Float64()*6,
		OrbitAngle:    rand.Float64() * 2 * math.Pi,
		SystemSeed:    time.Now().UnixNano(),
	}

	buildingEntity := tradingPostGen.Generate(params)
	if building, ok := buildingEntity.(*entities.Building); ok {
		building.Owner = owner
		building.AttachedTo = fmt.Sprintf("%d", planet.GetID())
		building.AttachmentType = "Planet"
		planet.Buildings = append(planet.Buildings, building)
		planet.RebalanceWorkforce()
	}
}

func seedInitialCommodities(planet *entities.Planet, owner string) {
	if planet == nil {
		return
	}

	for _, resourceEntity := range planet.Resources {
		if resource, ok := resourceEntity.(*entities.Resource); ok {
			if resource.Owner != owner {
				continue
			}
			amount := 400 + rand.Intn(400)
			planet.AddStoredResource(resource.ResourceType, amount)
		}
	}
}
