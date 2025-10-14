package main

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

// UI Colors - re-exported from ui package
var (
	UIBackground     = ui.Background
	UIPanelBg        = ui.PanelBg
	UIPanelBorder    = ui.PanelBorder
	UIHighlight      = ui.Highlight
	UITextPrimary    = ui.TextPrimary
	UITextSecondary  = ui.TextSecondary
	UIButtonActive   = ui.ButtonActive
	UIButtonDisabled = ui.ButtonDisabled
)

// Hyperlane Colors - re-exported from ui package
var (
	HyperlaneNormal = ui.HyperlaneNormal
	HyperlaneActive = ui.HyperlaneActive
)

// GetSystemColors returns an array of all system colors
func GetSystemColors() []color.RGBA {
	return ui.GetSystemColors()
}

// GetPlanetColors returns an array of planet colors indexed by type
func GetPlanetColors() []color.RGBA {
	return ui.GetPlanetColors()
}
