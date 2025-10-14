package views

// This file now re-exports colors from the ui package for backward compatibility
// Consider importing "github.com/hunterjsb/xandaris/ui" directly instead

import (
	"image/color"

	"github.com/hunterjsb/xandaris/ui"
)

// System Colors - re-exported from ui package
var (
	SystemBlue      = ui.SystemBlue
	SystemPurple    = ui.SystemPurple
	SystemGreen     = ui.SystemGreen
	SystemOrange    = ui.SystemOrange
	SystemYellow    = ui.SystemYellow
	SystemRed       = ui.SystemRed
	SystemLightBlue = ui.SystemLightBlue
	SystemPink      = ui.SystemPink
)

// Planet Colors - re-exported from ui package
var (
	PlanetTerrestrial = ui.PlanetTerrestrial
	PlanetGasGiant    = ui.PlanetGasGiant
	PlanetIce         = ui.PlanetIce
	PlanetDesert      = ui.PlanetDesert
	PlanetOcean       = ui.PlanetOcean
	PlanetLava        = ui.PlanetLava
)

// Station Colors - re-exported from ui package
var (
	ColorStationTrading  = ui.StationTrading
	ColorStationMilitary = ui.StationMilitary
	ColorStationResearch = ui.StationResearch
	ColorStationMining   = ui.StationMining
	ColorStationRefinery = ui.StationRefinery
	ColorStationShipyard = ui.StationShipyard
)

// Hyperlane Colors - re-exported from ui package
var (
	HyperlaneNormal = ui.HyperlaneNormal
	HyperlaneActive = ui.HyperlaneActive
)

// UI Colors - re-exported from ui package
var (
	UIBackground     = ui.Background
	UIHighlight      = ui.Highlight
	UIButtonActive   = ui.ButtonActive
	UIButtonDisabled = ui.ButtonDisabled
	UIBackgroundDark = ui.BackgroundDark
	UIPanelBg        = ui.PanelBg
	UIPanelBorder    = ui.PanelBorder
	UITextPrimary    = ui.TextPrimary
	UITextSecondary  = ui.TextSecondary
)

// GetSystemColors returns an array of all system colors
func GetSystemColors() []color.RGBA {
	return ui.GetSystemColors()
}

// GetPlanetColors returns an array of planet colors indexed by type
func GetPlanetColors() []color.RGBA {
	return ui.GetPlanetColors()
}

// GetPlanetTypeColor returns the color for a planet type string
func GetPlanetTypeColor(planetType string) color.RGBA {
	return ui.GetPlanetTypeColor(planetType)
}

// GetStationTypeColor returns the color for a station type string
func GetStationTypeColor(stationType string) color.RGBA {
	return ui.GetStationTypeColor(stationType)
}
