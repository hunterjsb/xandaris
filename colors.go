package main

import "image/color"

// System Colors
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

// Station Colors (available for use, but stations manage their own colors)
var (
	ColorStationTrading  = color.RGBA{255, 100, 100, 255} // Red
	ColorStationMilitary = color.RGBA{100, 100, 255, 255} // Blue
	ColorStationResearch = color.RGBA{100, 255, 100, 255} // Green
	ColorStationMining   = color.RGBA{255, 255, 100, 255} // Yellow
	ColorStationRefinery = color.RGBA{255, 150, 100, 255} // Orange
	ColorStationShipyard = color.RGBA{150, 100, 255, 255} // Purple
)

// UI Colors
var (
	UIBackground     = color.RGBA{5, 5, 15, 255}
	UIPanelBg        = color.RGBA{20, 20, 40, 230}
	UIPanelBorder    = color.RGBA{100, 100, 150, 255}
	UIHighlight      = color.RGBA{255, 255, 100, 255}
	UITextPrimary    = color.RGBA{255, 255, 255, 255}
	UITextSecondary  = color.RGBA{200, 200, 200, 255}
	UIButtonActive   = color.RGBA{40, 80, 120, 230}
	UIButtonDisabled = color.RGBA{60, 60, 60, 230}
)

// Hyperlane Colors
var (
	HyperlaneNormal = color.RGBA{40, 40, 80, 255}
	HyperlaneActive = color.RGBA{80, 80, 160, 255}
)

// GetSystemColors returns an array of all system colors
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
