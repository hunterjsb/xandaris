package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
)

// FleetInfoUI displays detailed information about a selected fleet
type FleetInfoUI struct {
	game         *Game
	fleet        *Fleet
	x            int
	y            int
	width        int
	height       int
	visible      bool
	scrollOffset int
}

// NewFleetInfoUI creates a new fleet info UI
func NewFleetInfoUI(game *Game) *FleetInfoUI {
	return &FleetInfoUI{
		game:   game,
		x:      screenWidth - 320,
		y:      80,
		width:  310,
		height: 400,
	}
}

// Show displays the fleet info for a specific fleet
func (fui *FleetInfoUI) Show(fleet *Fleet) {
	fui.fleet = fleet
	fui.visible = true
	fui.scrollOffset = 0
}

// Hide closes the fleet info UI
func (fui *FleetInfoUI) Hide() {
	fui.visible = false
	fui.fleet = nil
}

// IsVisible returns whether the UI is visible
func (fui *FleetInfoUI) IsVisible() bool {
	return fui.visible
}

// Update handles input for the fleet info UI
func (fui *FleetInfoUI) Update() {
	if !fui.visible || fui.fleet == nil {
		return
	}

	// Handle mouse wheel scrolling
	_, dy := ebiten.Wheel()
	if dy != 0 {
		fui.scrollOffset -= int(dy * 20)
		// Clamp scroll offset
		maxScroll := len(fui.fleet.Ships)*60 - (fui.height - 100)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if fui.scrollOffset < 0 {
			fui.scrollOffset = 0
		}
		if fui.scrollOffset > maxScroll {
			fui.scrollOffset = maxScroll
		}
	}

	// Handle close button click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		closeX := fui.x + fui.width - 30
		closeY := fui.y + 10
		if mx >= closeX && mx <= closeX+20 && my >= closeY && my <= closeY+20 {
			fui.Hide()
		}
	}
}

// Draw renders the fleet info panel
func (fui *FleetInfoUI) Draw(screen *ebiten.Image) {
	if !fui.visible || fui.fleet == nil {
		return
	}

	// Background panel
	panel := &UIPanel{
		X:           fui.x,
		Y:           fui.y,
		Width:       fui.width,
		Height:      fui.height,
		BgColor:     UIBackground,
		BorderColor: UIPanelBorder,
	}
	panel.Draw(screen)

	// Title
	titleText := fmt.Sprintf("Fleet Details")
	DrawText(screen, titleText, fui.x+10, fui.y+15, SystemLightBlue)

	// Close button
	closeX := fui.x + fui.width - 30
	closeY := fui.y + 10
	DrawText(screen, "[X]", closeX, closeY, SystemRed)

	// Fleet summary
	summaryY := fui.y + 40
	DrawText(screen, fmt.Sprintf("Ships: %d", fui.fleet.Size()), fui.x+10, summaryY, UITextPrimary)
	DrawText(screen, fmt.Sprintf("Owner: %s", fui.fleet.Owner), fui.x+10, summaryY+15, UITextSecondary)

	// Fuel stats
	fuelY := summaryY + 35
	avgFuel := fui.fleet.GetAverageFuelPercent()
	fuelColor := ColorStationResearch // Green for good fuel
	if avgFuel < 25 {
		fuelColor = SystemRed // Red for low fuel
	} else if avgFuel < 50 {
		fuelColor = SystemOrange // Orange for medium fuel
	}
	DrawText(screen, fmt.Sprintf("Avg Fuel: %.0f%%", avgFuel), fui.x+10, fuelY, fuelColor)
	DrawText(screen, fmt.Sprintf("Total: %d/%d", fui.fleet.GetTotalFuel(), fui.fleet.GetTotalMaxFuel()),
		fui.x+10, fuelY+15, UITextSecondary)

	// Separator
	separatorY := fuelY + 35
	DrawLine(screen, fui.x+10, separatorY, fui.x+fui.width-10, separatorY, UIPanelBorder)

	// Ship list header
	listHeaderY := separatorY + 10
	DrawText(screen, "Ships:", fui.x+10, listHeaderY, UITextPrimary)

	// Scrollable ship list
	fui.drawShipList(screen, listHeaderY+20)

	// Scroll indicator
	if len(fui.fleet.Ships) > 5 {
		scrollHintY := fui.y + fui.height - 20
		DrawTextCentered(screen, "Scroll for more", fui.x+fui.width/2, scrollHintY, UITextSecondary, 0.7)
	}
}

// drawShipList draws the scrollable list of ships
func (fui *FleetInfoUI) drawShipList(screen *ebiten.Image, startY int) {
	itemHeight := 60

	for i, ship := range fui.fleet.Ships {
		itemY := startY + i*itemHeight - fui.scrollOffset

		// Skip if off screen
		if itemY < startY-itemHeight || itemY > fui.y+fui.height-30 {
			continue
		}

		// Ship item background
		itemPanel := &UIPanel{
			X:           fui.x + 10,
			Y:           itemY,
			Width:       fui.width - 20,
			Height:      itemHeight - 5,
			BgColor:     UIPanelBg,
			BorderColor: UIPanelBorder,
		}
		itemPanel.Draw(screen)

		// Ship icon (small triangle)
		iconX := fui.x + 20
		iconY := itemY + 10
		iconSize := 4
		for py := 0; py < iconSize*2; py++ {
			for px := 0; px < iconSize*2; px++ {
				dx := float64(px - iconSize)
				dy := float64(py - iconSize)
				if dy > 0 && dx >= -dy/2 && dx <= dy/2 {
					screen.Set(iconX+px, iconY+py, ship.Color)
				}
			}
		}

		// Ship name and type
		nameX := fui.x + 35
		DrawText(screen, ship.Name, nameX, itemY+8, UITextPrimary)
		DrawText(screen, string(ship.ShipType), nameX, itemY+23, UITextSecondary)

		// Ship stats
		statsY := itemY + 38
		fuelPercent := ship.GetFuelPercentage()
		fuelColor := UITextPrimary
		if fuelPercent < 25 {
			fuelColor = SystemRed
		} else if fuelPercent < 50 {
			fuelColor = SystemOrange
		}

		DrawText(screen, fmt.Sprintf("Fuel: %.0f%%", fuelPercent), nameX, statsY, fuelColor)
		healthPercent := ship.GetHealthPercentage()
		healthColor := UITextPrimary
		if healthPercent < 50 {
			healthColor = SystemOrange
		}
		if healthPercent < 25 {
			healthColor = SystemRed
		}
		DrawText(screen, fmt.Sprintf("HP: %.0f%%", healthPercent), nameX+90, statsY, healthColor)

		// Status indicator
		statusX := fui.x + fui.width - 80
		statusText := string(ship.Status)
		statusColor := UITextSecondary
		if ship.Status == entities.ShipStatusMoving {
			statusColor = SystemBlue
		}
		DrawText(screen, statusText, statusX, itemY+8, statusColor)
	}
}

// GetFleet returns the currently displayed fleet
func (fui *FleetInfoUI) GetFleet() *Fleet {
	return fui.fleet
}
