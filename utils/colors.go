package utils

import "image/color"

var (
	PlayerGreen  = color.RGBA{180, 230, 130, 255}
	PlayerBlue   = color.RGBA{130, 180, 255, 255}
	PlayerRed    = color.RGBA{255, 160, 160, 255}
	PlayerOrange = color.RGBA{255, 190, 120, 255}
	PlayerPurple = color.RGBA{190, 150, 255, 255}
	PlayerTeal   = color.RGBA{140, 220, 210, 255}
)

// System Colors - used for star system visualization
var (
	SystemBlue      = color.RGBA{100, 100, 200, 255}
	SystemPurple    = color.RGBA{200, 100, 150, 255}
	SystemGreen     = color.RGBA{150, 200, 100, 255}
	SystemOrange    = color.RGBA{200, 150, 100, 255}
	SystemYellow    = color.RGBA{200, 200, 100, 255}
	SystemRed       = color.RGBA{200, 100, 100, 255}
	SystemLightBlue = color.RGBA{150, 150, 200, 255}
	SystemPink      = color.RGBA{180, 120, 180, 255}
)

// Planet Colors by Type
var (
	PlanetTerrestrial = color.RGBA{100, 150, 100, 255} // Green
	PlanetGasGiant    = color.RGBA{200, 180, 150, 255} // Tan
	PlanetIce         = color.RGBA{150, 200, 255, 255} // Light Blue
	PlanetDesert      = color.RGBA{200, 180, 100, 255} // Yellow
	PlanetOcean       = color.RGBA{50, 100, 200, 255}  // Blue
	PlanetLava        = color.RGBA{255, 100, 50, 255}  // Red
)

// Station Colors
var (
	StationTrading  = color.RGBA{255, 100, 100, 255} // Red
	StationMilitary = color.RGBA{100, 100, 255, 255} // Blue
	StationResearch = color.RGBA{100, 255, 100, 255} // Green
	StationMining   = color.RGBA{255, 255, 100, 255} // Yellow
	StationRefinery = color.RGBA{255, 150, 100, 255} // Orange
	StationShipyard = color.RGBA{150, 100, 255, 255} // Purple
)

// Hyperlane Colors
var (
	HyperlaneNormal = color.RGBA{40, 40, 80, 255}
	HyperlaneActive = color.RGBA{80, 80, 160, 255}
)

// UI Theme Colors
var (
	Background     = color.RGBA{5, 5, 15, 255}
	BackgroundDark = color.RGBA{15, 15, 25, 255}
	PanelBg        = color.RGBA{20, 20, 40, 230}
	PanelBorder    = color.RGBA{100, 100, 150, 255}
	Highlight      = color.RGBA{255, 255, 100, 255}
	TextPrimary    = color.RGBA{255, 255, 255, 255}
	TextSecondary  = color.RGBA{200, 200, 200, 255}
	ButtonActive   = color.RGBA{40, 80, 120, 230}
	ButtonDisabled = color.RGBA{60, 60, 60, 230}
)

// GetSystemColors returns an array of all system colors for random selection
func GetSystemColors() []color.RGBA {
	return []color.RGBA{
		SystemBlue,
		SystemPurple,
		SystemGreen,
		SystemOrange,
		SystemYellow,
		SystemRed,
		SystemLightBlue,
		SystemPink,
	}
}

// GetPlanetColors returns an array of planet colors indexed by type
func GetPlanetColors() []color.RGBA {
	return []color.RGBA{
		PlanetTerrestrial, // Index 0: Terrestrial
		PlanetGasGiant,    // Index 1: Gas Giant
		PlanetIce,         // Index 2: Ice World
		PlanetDesert,      // Index 3: Desert
		PlanetOcean,       // Index 4: Ocean
		PlanetLava,        // Index 5: Lava
	}
}

// GetPlanetTypeColor returns the color for a planet type string
func GetPlanetTypeColor(planetType string) color.RGBA {
	switch planetType {
	case "Terrestrial":
		return PlanetTerrestrial
	case "Gas Giant":
		return PlanetGasGiant
	case "Ice World":
		return PlanetIce
	case "Desert":
		return PlanetDesert
	case "Ocean":
		return PlanetOcean
	case "Lava":
		return PlanetLava
	default:
		return PlanetTerrestrial // Default fallback
	}
}

// GetStationTypeColor returns the color for a station type string
func GetStationTypeColor(stationType string) color.RGBA {
	// Remove " Station" suffix if present
	if len(stationType) > 8 && stationType[len(stationType)-8:] == " Station" {
		stationType = stationType[:len(stationType)-8]
	}

	switch stationType {
	case "Trading":
		return StationTrading
	case "Military":
		return StationMilitary
	case "Research":
		return StationResearch
	case "Mining":
		return StationMining
	case "Refinery":
		return StationRefinery
	case "Shipyard":
		return StationShipyard
	default:
		return StationTrading // Default fallback
	}
}

// GetAIPlayerColors returns a palette of colors for AI factions
func GetAIPlayerColors() []color.RGBA {
	return []color.RGBA{
		PlayerBlue,
		PlayerRed,
		PlayerOrange,
		PlayerPurple,
		PlayerTeal,
	}
}
