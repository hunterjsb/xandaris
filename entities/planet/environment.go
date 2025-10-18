package planet

import (
	"math/rand"

	"github.com/hunterjsb/xandaris/entities"
)

type atmosphereWeight struct {
	Type   string
	Weight float64
}

// Weighted atmosphere distributions by planet subtype
var planetAtmosphereDistributions = map[string][]atmosphereWeight{
	"Terrestrial": {
		{Type: entities.AtmosphereBreathable, Weight: 0.45},
		{Type: entities.AtmosphereThin, Weight: 0.35},
		{Type: entities.AtmosphereToxic, Weight: 0.20},
	},
	"Ocean": {
		{Type: entities.AtmosphereBreathable, Weight: 0.60},
		{Type: entities.AtmosphereThin, Weight: 0.25},
		{Type: entities.AtmosphereToxic, Weight: 0.15},
	},
	"Desert": {
		{Type: entities.AtmosphereThin, Weight: 0.55},
		{Type: entities.AtmosphereBreathable, Weight: 0.15},
		{Type: entities.AtmosphereToxic, Weight: 0.30},
	},
	"Ice": {
		{Type: entities.AtmosphereThin, Weight: 0.70},
		{Type: entities.AtmosphereNone, Weight: 0.30},
	},
	"Barren": {
		{Type: entities.AtmosphereThin, Weight: 0.40},
		{Type: entities.AtmosphereNone, Weight: 0.40},
		{Type: entities.AtmosphereToxic, Weight: 0.20},
	},
	"Gas Giant": {
		{Type: entities.AtmosphereDense, Weight: 1.0},
	},
	"Lava": {
		{Type: entities.AtmosphereCorrosive, Weight: 1.0},
	},
}

// Habitability modifiers applied per atmosphere class
var atmosphereHabitabilityModifiers = map[string]int{
	entities.AtmosphereBreathable: 25,
	entities.AtmosphereThin:       5,
	entities.AtmosphereDense:      -10,
	entities.AtmosphereToxic:      -25,
	entities.AtmosphereCorrosive:  -40,
	entities.AtmosphereNone:       -30,
}

// Habitability baseline modifiers for specific planet subtypes
var planetTypeHabitabilityModifiers = map[string]int{
	"Terrestrial": 5,
	"Ocean":       15,
	"Desert":      -10,
	"Ice":         -15,
}

type temperatureProfile struct {
	IdealMin     int
	IdealMax     int
	HabitableMin int
	HabitableMax int
}

// Preferred temperature ranges by planet subtype
var planetTemperatureProfiles = map[string]temperatureProfile{
	"default":     {IdealMin: -10, IdealMax: 30, HabitableMin: -50, HabitableMax: 60},
	"Terrestrial": {IdealMin: -10, IdealMax: 30, HabitableMin: -45, HabitableMax: 60},
	"Ocean":       {IdealMin: 0, IdealMax: 35, HabitableMin: -25, HabitableMax: 45},
	"Desert":      {IdealMin: 25, IdealMax: 55, HabitableMin: -5, HabitableMax: 80},
	"Ice":         {IdealMin: -70, IdealMax: -30, HabitableMin: -110, HabitableMax: 0},
}

// randomAtmosphereForType picks a weighted atmosphere for the provided planet subtype
func randomAtmosphereForType(planetType string) string {
	distribution, ok := planetAtmosphereDistributions[planetType]
	if !ok || len(distribution) == 0 {
		return entities.AtmosphereNone
	}

	totalWeight := 0.0
	for _, entry := range distribution {
		totalWeight += entry.Weight
	}

	if totalWeight <= 0 {
		return distribution[len(distribution)-1].Type
	}

	choice := rand.Float64() * totalWeight
	accumulated := 0.0
	for _, entry := range distribution {
		accumulated += entry.Weight
		if choice <= accumulated {
			return entry.Type
		}
	}

	return distribution[len(distribution)-1].Type
}

func temperatureScore(temp int, profile temperatureProfile) int {
	if temp >= profile.IdealMin && temp <= profile.IdealMax {
		return 30
	}
	if temp >= profile.HabitableMin && temp <= profile.HabitableMax {
		return 10
	}
	return -35
}
