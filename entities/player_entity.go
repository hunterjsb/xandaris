package entities

import (
	"image/color"
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
	HomeSystem *System // *System
	HomePlanet *Planet

	// Owned entities
	OwnedPlanets  []*Planet
	OwnedStations []*Station
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
		OwnedPlanets:  make([]*Planet, 0),
		OwnedStations: make([]*Station, 0),
		OwnedFleets:   make([]interface{}, 0),
	}
}

// AddOwnedPlanet adds a planet to the player's ownership
func (p *Player) AddOwnedPlanet(planet *Planet) {
	p.OwnedPlanets = append(p.OwnedPlanets, planet)
}

// AddOwnedStation adds a station to the player's ownership
func (p *Player) AddOwnedStation(station *Station) {
	p.OwnedStations = append(p.OwnedStations, station)
}

// RemoveOwnedPlanet removes a planet from the player's ownership
func (p *Player) RemoveOwnedPlanet(planet *Planet) {
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
func (p *Player) OwnsPlanet(planet *Planet) bool {
	for _, owned := range p.OwnedPlanets {
		if owned == planet {
			return true
		}
	}
	return false
}
