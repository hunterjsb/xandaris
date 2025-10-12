package main

import (
	"image/color"
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

// PlayerType represents the type of player
type PlayerType string

const (
	PlayerTypeHuman PlayerType = "Human"
	PlayerTypeAI    PlayerType = "AI"
)

// Player represents a player in the game (human or AI)
type Player struct {
	ID         int
	Name       string
	Color      color.RGBA
	Type       PlayerType
	Credits    int
	HomeSystem *System
	HomePlanet *entities.Planet

	// Owned entities
	OwnedPlanets  []*entities.Planet
	OwnedStations []*entities.Station
	OwnedFleets   []interface{} // For future fleet system
}

// NewPlayer creates a new player
func NewPlayer(id int, name string, playerColor color.RGBA, playerType PlayerType) *Player {
	return &Player{
		ID:            id,
		Name:          name,
		Color:         playerColor,
		Type:          playerType,
		Credits:       1000, // Starting credits
		OwnedPlanets:  make([]*entities.Planet, 0),
		OwnedStations: make([]*entities.Station, 0),
		OwnedFleets:   make([]interface{}, 0),
	}
}

// AddOwnedPlanet adds a planet to the player's ownership
func (p *Player) AddOwnedPlanet(planet *entities.Planet) {
	p.OwnedPlanets = append(p.OwnedPlanets, planet)
}

// AddOwnedStation adds a station to the player's ownership
func (p *Player) AddOwnedStation(station *entities.Station) {
	p.OwnedStations = append(p.OwnedStations, station)
}

// RemoveOwnedPlanet removes a planet from the player's ownership
func (p *Player) RemoveOwnedPlanet(planet *entities.Planet) {
	for i, owned := range p.OwnedPlanets {
		if owned == planet {
			p.OwnedPlanets = append(p.OwnedPlanets[:i], p.OwnedPlanets[i+1:]...)
			break
		}
	}
}

// GetTotalPopulation returns the player's total population across all planets
func (p *Player) GetTotalPopulation() int64 {
	total := int64(0)
	for _, planet := range p.OwnedPlanets {
		total += planet.Population
	}
	return total
}

// IsHuman returns whether this is a human player
func (p *Player) IsHuman() bool {
	return p.Type == PlayerTypeHuman
}

// IsAI returns whether this is an AI player
func (p *Player) IsAI() bool {
	return p.Type == PlayerTypeAI
}

// OwnsPlanet checks if the player owns a specific planet
func (p *Player) OwnsPlanet(planet *entities.Planet) bool {
	for _, owned := range p.OwnedPlanets {
		if owned == planet {
			return true
		}
	}
	return false
}

// PlayerController interface for future AI implementation
type PlayerController interface {
	GetPlayer() *Player
	TakeTurn() // For turn-based gameplay
	Update()   // For real-time gameplay
}

// HumanController controls a human player
type HumanController struct {
	player *Player
}

// NewHumanController creates a new human player controller
func NewHumanController(player *Player) *HumanController {
	return &HumanController{
		player: player,
	}
}

// GetPlayer returns the player
func (h *HumanController) GetPlayer() *Player {
	return h.player
}

// TakeTurn is called when it's the player's turn (turn-based)
func (h *HumanController) TakeTurn() {
	// Human player actions are handled through UI
}

// Update is called every frame (real-time)
func (h *HumanController) Update() {
	// Human player actions are handled through UI
}

// InitializePlayer sets up a new player with a starting planet
func (g *Game) InitializePlayer(player *Player) {
	// Find systems with habitable terrestrial planets
	validSystems := make([]*System, 0)

	for _, system := range g.systems {
		// Check if system has habitable terrestrial planets
		for _, entity := range system.Entities {
			if planet, ok := entity.(*entities.Planet); ok {
				if planet.PlanetType == "Terrestrial" && planet.IsHabitable() {
					validSystems = append(validSystems, system)
					break
				}
			}
		}
	}

	// Fallback: use any system with a planet
	if len(validSystems) == 0 {
		for _, system := range g.systems {
			if system.HasEntityType(entities.EntityTypePlanet) {
				validSystems = append(validSystems, system)
			}
		}
	}

	if len(validSystems) == 0 {
		return // No valid systems found
	}

	// Pick a random system
	homeSystem := validSystems[rand.Intn(len(validSystems))]
	player.HomeSystem = homeSystem

	// Find the best terrestrial planet in that system
	var bestPlanet *entities.Planet
	bestHabitability := 0

	for _, entity := range homeSystem.Entities {
		if planet, ok := entity.(*entities.Planet); ok {
			if planet.PlanetType == "Terrestrial" && planet.Habitability > bestHabitability {
				bestPlanet = planet
				bestHabitability = planet.Habitability
			}
		}
	}

	// If no terrestrial planet, pick any habitable planet
	if bestPlanet == nil {
		for _, entity := range homeSystem.Entities {
			if planet, ok := entity.(*entities.Planet); ok {
				if planet.IsHabitable() {
					bestPlanet = planet
					break
				}
			}
		}
	}

	// Last resort: pick any planet
	if bestPlanet == nil {
		for _, entity := range homeSystem.Entities {
			if planet, ok := entity.(*entities.Planet); ok {
				bestPlanet = planet
				break
			}
		}
	}

	if bestPlanet != nil {
		// Set up the home planet
		player.HomePlanet = bestPlanet
		bestPlanet.Population = 100000000 // 100 million starting population
		bestPlanet.Owner = player.Name    // Mark planet as owned by player

		// Mark all resources on the home planet as owned by player
		for _, resource := range bestPlanet.Resources {
			if res, ok := resource.(*entities.Resource); ok {
				res.Owner = player.Name
			}
		}

		player.AddOwnedPlanet(bestPlanet)
	}
}
