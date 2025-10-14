package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/systems"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/views"
)

// FleetInfoUI displays detailed information about a selected fleet
type FleetInfoUI struct {
	game                  *Game
	fleet                 *views.Fleet
	x                     int
	y                     int
	width                 int
	height                int
	visible               bool
	scrollOffset          int
	showMoveMenu          bool
	connectedSystems      []int
	currentSystemEntities []entities.Entity
	moveMenuScrollOffset  int
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

// ShowFleet displays the fleet info for a specific fleet
func (fui *FleetInfoUI) ShowFleet(fleet *views.Fleet) {
	fui.fleet = fleet
	fui.visible = true
	fui.scrollOffset = 0
	fui.showMoveMenu = false
	fui.moveMenuScrollOffset = 0
}

// ShowShip displays the fleet info for a single ship (wrapped as a fleet)
func (fui *FleetInfoUI) ShowShip(ship *entities.Ship) {
	// Create a single-ship fleet for display
	fleet := views.NewFleet([]*entities.Ship{ship})
	fui.ShowFleet(fleet)
}

// Hide closes the fleet info UI
func (fui *FleetInfoUI) Hide() {
	fui.visible = false
	fui.fleet = nil
	fui.showMoveMenu = false
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

	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()

		// Close button
		closeX := fui.x + fui.width - 30
		closeY := fui.y + 10
		if mx >= closeX && mx <= closeX+20 && my >= closeY && my <= closeY+20 {
			fui.Hide()
			return
		}

		// Move Fleet button
		if !fui.showMoveMenu {
			moveButtonX := fui.x + 10
			moveButtonY := fui.y + fui.height - 40
			moveButtonW := fui.width - 20
			moveButtonH := 30
			if mx >= moveButtonX && mx <= moveButtonX+moveButtonW &&
				my >= moveButtonY && my <= moveButtonY+moveButtonH {
				// Show move menu
				fui.showMoveMenu = true
				// Get connected systems and current system entities
				if len(fui.fleet.Ships) > 0 {
					firstShip := fui.fleet.Ships[0]
					helper := tickable.NewShipMovementHelper(fui.game.GetSystemsMap(), fui.game.GetHyperlanes())
					fui.connectedSystems = helper.GetConnectedSystems(firstShip.CurrentSystem)

					// Get current system entities (planets)
					systems := fui.game.GetSystemsMap()
					currentSystem := systems[firstShip.CurrentSystem]
					if currentSystem != nil {
						fui.currentSystemEntities = make([]entities.Entity, 0)
						for _, entity := range currentSystem.Entities {
							if _, isPlanet := entity.(*entities.Planet); isPlanet {
								fui.currentSystemEntities = append(fui.currentSystemEntities, entity)
							}
						}
					}
				}
				return
			}
		}

		// Handle move menu destination clicks
		if fui.showMoveMenu {
			// Back button
			backButtonX := fui.x + 10
			backButtonY := fui.y + fui.height - 40
			backButtonW := 60
			backButtonH := 30
			if mx >= backButtonX && mx <= backButtonX+backButtonW &&
				my >= backButtonY && my <= backButtonY+backButtonH {
				fui.showMoveMenu = false
				return
			}

			// Calculate starting positions for both sections
			listStartY := fui.y + 110
			itemHeight := 35

			// Section 1: Adjacent Systems
			systemSectionHeight := len(fui.connectedSystems) * itemHeight

			for i, systemID := range fui.connectedSystems {
				itemY := listStartY + i*itemHeight - fui.moveMenuScrollOffset
				if itemY < listStartY-itemHeight || itemY > fui.y+fui.height-60 {
					continue
				}
				if mx >= fui.x+10 && mx <= fui.x+fui.width-10 &&
					my >= itemY && my <= itemY+itemHeight-5 {
					// Move fleet to this system (inter-system jump)
					fleetManager := systems.NewFleetManager(fui.game)
					success, _ := fleetManager.MoveFleet(fui.fleet, systemID)
					if success > 0 {
						fui.showMoveMenu = false
					}
					return
				}
			}

			// Section 2: Move to Star (if at planet)
			currentOffset := systemSectionHeight + 45 // 45 for section header
			if fui.isFleetAtPlanet() {
				starItemY := listStartY + currentOffset + 20 - fui.moveMenuScrollOffset // 20 for header
				if starItemY >= listStartY-itemHeight && starItemY <= fui.y+fui.height-60 {
					if mx >= fui.x+10 && mx <= fui.x+fui.width-10 &&
						my >= starItemY && my <= starItemY+itemHeight-5 {
						// Move fleet to star
						fui.moveFleetToStar()
						fui.showMoveMenu = false
						return
					}
				}
				currentOffset += itemHeight + 30 // item + header + spacing
			}

			// Section 3: Current System Entities (planets)
			entitySectionStartY := listStartY + currentOffset
			for i, entity := range fui.currentSystemEntities {
				itemY := entitySectionStartY + i*itemHeight - fui.moveMenuScrollOffset
				if itemY < listStartY-itemHeight || itemY > fui.y+fui.height-60 {
					continue
				}
				if mx >= fui.x+10 && mx <= fui.x+fui.width-10 &&
					my >= itemY && my <= itemY+itemHeight-5 {
					// Move fleet to this planet (intra-system movement)
					fui.moveFleetToPlanet(entity)
					fui.showMoveMenu = false
					return
				}
			}
		}
	}

	// Handle scroll for move menu
	if fui.showMoveMenu {
		_, dy := ebiten.Wheel()
		if dy != 0 {
			fui.moveMenuScrollOffset -= int(dy * 20)
			// Calculate total content height (all sections + headers)
			starItems := 0
			if fui.isFleetAtPlanet() {
				starItems = 1
			}
			totalContentHeight := (len(fui.connectedSystems)+starItems+len(fui.currentSystemEntities))*35 + 80 // 80 for section headers
			maxScroll := totalContentHeight - (fui.height - 160)
			if maxScroll < 0 {
				maxScroll = 0
			}
			if fui.moveMenuScrollOffset < 0 {
				fui.moveMenuScrollOffset = 0
			}
			if fui.moveMenuScrollOffset > maxScroll {
				fui.moveMenuScrollOffset = maxScroll
			}
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

	// Show either the ship list OR the move menu (not both)
	if fui.showMoveMenu {
		fui.drawMoveMenu(screen)
	} else {
		// Ship list header
		listHeaderY := separatorY + 10
		DrawText(screen, "Ships:", fui.x+10, listHeaderY, UITextPrimary)

		// Scrollable ship list
		fui.drawShipList(screen, listHeaderY+20)

		// Move button
		fui.drawMoveButton(screen)

		// Scroll indicator for ship list
		if len(fui.fleet.Ships) > 5 {
			scrollHintY := fui.y + fui.height - 50
			DrawTextCentered(screen, "Scroll for more", fui.x+fui.width/2, scrollHintY, UITextSecondary, 0.7)
		}
	}
}

// drawMoveButton draws the "Move Fleet" button
func (fui *FleetInfoUI) drawMoveButton(screen *ebiten.Image) {
	buttonX := fui.x + 10
	buttonY := fui.y + fui.height - 40
	buttonW := fui.width - 20
	buttonH := 30

	// Check if fleet can move (has any ship with fuel)
	canMove, lowFuel, noFuel := systems.NewFleetManager(fui.game).GetFleetMovementStatus(fui.fleet)

	buttonColor := UIButtonActive
	buttonText := "Move Fleet"

	if canMove == 0 {
		buttonColor = UIButtonDisabled
		if noFuel > 0 {
			buttonText = "No Fuel"
		} else {
			buttonText = "Cannot Move"
		}
	} else if lowFuel > 0 || noFuel > 0 {
		buttonText = fmt.Sprintf("Move Fleet (%d/%d ready)", canMove, len(fui.fleet.Ships))
	}

	// Button background
	buttonPanel := &UIPanel{
		X:           buttonX,
		Y:           buttonY,
		Width:       buttonW,
		Height:      buttonH,
		BgColor:     buttonColor,
		BorderColor: UIHighlight,
	}
	buttonPanel.Draw(screen)

	// Button text
	textX := buttonX + buttonW/2
	textY := buttonY + 10
	DrawTextCentered(screen, buttonText, textX, textY, UITextPrimary, 1.0)
}

// drawMoveMenu draws the destination selection menu
func (fui *FleetInfoUI) drawMoveMenu(screen *ebiten.Image) {
	// Title (fixed, doesn't scroll)
	menuTitleY := fui.y + 100
	DrawText(screen, "Select Destination:", fui.x+10, menuTitleY, SystemLightBlue)

	// Create a scrollable content area
	contentStartY := menuTitleY + 10
	itemHeight := 35
	headerHeight := 20

	// Track current Y position for drawing (affected by scroll)
	currentY := contentStartY - fui.moveMenuScrollOffset

	// Define visible area bounds
	visibleTop := contentStartY
	visibleBottom := fui.y + fui.height - 60

	// SECTION 1: Adjacent Systems
	// Draw header
	if currentY >= visibleTop-headerHeight && currentY <= visibleBottom {
		DrawText(screen, "Jump to System:", fui.x+10, currentY, UITextPrimary)
	}
	currentY += headerHeight

	if len(fui.connectedSystems) == 0 {
		if currentY >= visibleTop-15 && currentY <= visibleBottom {
			DrawText(screen, "  No adjacent systems", fui.x+20, currentY+5, UITextSecondary)
		}
		currentY += 15
	}

	systems := fui.game.GetSystemsMap()
	for _, systemID := range fui.connectedSystems {
		itemY := currentY

		// Only draw if visible
		if itemY >= visibleTop-itemHeight && itemY <= visibleBottom {
			system := systems[systemID]
			if system != nil {
				fui.drawMenuItem(screen, itemY, itemHeight, system.Color, "⟫ "+system.Name)
			}
		}
		currentY += itemHeight
	}

	// Add spacing between sections
	currentY += 10

	// SECTION 2: Move to Star (if currently at planet)
	if fui.isFleetAtPlanet() {
		// Draw header
		if currentY >= visibleTop-headerHeight && currentY <= visibleBottom {
			DrawText(screen, "Move to Star:", fui.x+10, currentY, UITextPrimary)
		}
		currentY += headerHeight

		itemY := currentY
		if itemY >= visibleTop-itemHeight && itemY <= visibleBottom {
			// Get star color from current system
			if len(fui.fleet.Ships) > 0 {
				firstShip := fui.fleet.Ships[0]
				currentSystem := systems[firstShip.CurrentSystem]
				if currentSystem != nil {
					starEntities := currentSystem.GetEntitiesByType(entities.EntityTypeStar)
					if len(starEntities) > 0 {
						if star, ok := starEntities[0].(*entities.Star); ok {
							fui.drawMenuItem(screen, itemY, itemHeight, star.Color, "★ "+star.Name)
						}
					}
				}
			}
		}
		currentY += itemHeight
		currentY += 10
	}

	// SECTION 3: Current System Entities (planets)
	// Draw header
	if currentY >= visibleTop-headerHeight && currentY <= visibleBottom {
		DrawText(screen, "Move to Planet:", fui.x+10, currentY, UITextPrimary)
	}
	currentY += headerHeight

	if len(fui.currentSystemEntities) == 0 {
		if currentY >= visibleTop-15 && currentY <= visibleBottom {
			DrawText(screen, "  No planets in system", fui.x+20, currentY+5, UITextSecondary)
		}
		currentY += 15
	}

	for _, entity := range fui.currentSystemEntities {
		itemY := currentY

		// Only draw if visible
		if itemY >= visibleTop-itemHeight && itemY <= visibleBottom {
			planet, ok := entity.(*entities.Planet)
			if ok {
				fui.drawMenuItem(screen, itemY, itemHeight, planet.Color, "◉ "+planet.Name)
			}
		}
		currentY += itemHeight
	}

	// Back button
	backButtonX := fui.x + 10
	backButtonY := fui.y + fui.height - 40
	backButtonW := 60
	backButtonH := 30

	backPanel := &UIPanel{
		X:           backButtonX,
		Y:           backButtonY,
		Width:       backButtonW,
		Height:      backButtonH,
		BgColor:     UIButtonActive,
		BorderColor: UIHighlight,
	}
	backPanel.Draw(screen)
	DrawTextCentered(screen, "Back", backButtonX+backButtonW/2, backButtonY+10, UITextPrimary, 1.0)

	// Scroll hint
	totalItems := len(fui.connectedSystems) + len(fui.currentSystemEntities)
	if totalItems > 5 {
		scrollHintY := fui.y + fui.height - 50
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
func (fui *FleetInfoUI) GetFleet() *views.Fleet {
	return fui.fleet
}

// drawMenuItem draws a single menu item (helper function)
func (fui *FleetInfoUI) drawMenuItem(screen *ebiten.Image, y int, height int, itemColor color.RGBA, text string) {
	// Item background
	itemPanel := &UIPanel{
		X:           fui.x + 10,
		Y:           y,
		Width:       fui.width - 20,
		Height:      height - 5,
		BgColor:     UIPanelBg,
		BorderColor: UIPanelBorder,
	}
	itemPanel.Draw(screen)

	// Color indicator
	colorSize := 6
	colorX := fui.x + 20
	colorY := y + height/2 - colorSize/2
	for py := 0; py < colorSize; py++ {
		for px := 0; px < colorSize; px++ {
			screen.Set(colorX+px, colorY+py, itemColor)
		}
	}

	// Text
	DrawText(screen, text, fui.x+35, y+10, UITextPrimary)
}

// isFleetAtPlanet checks if the fleet is currently orbiting a planet
func (fui *FleetInfoUI) isFleetAtPlanet() bool {
	if fui.fleet == nil || len(fui.fleet.Ships) == 0 {
		return false
	}

	// Check if any ship is at a planet's orbit distance
	firstShip := fui.fleet.Ships[0]
	systems := fui.game.GetSystemsMap()
	currentSystem := systems[firstShip.CurrentSystem]
	if currentSystem == nil {
		return false
	}

	// Check if ship orbit matches any planet orbit
	for _, entity := range currentSystem.Entities {
		if planet, ok := entity.(*entities.Planet); ok {
			if math.Abs(firstShip.GetOrbitDistance()-planet.GetOrbitDistance()) < 1.0 {
				return true
			}
		}
	}
	return false
}

// moveFleetToPlanet moves a fleet to orbit a planet in the current system
func (fui *FleetInfoUI) moveFleetToPlanet(entity entities.Entity) {
	planet, ok := entity.(*entities.Planet)
	if !ok || fui.fleet == nil {
		return
	}

	// Move all ships to the planet's orbit
	for _, ship := range fui.fleet.Ships {
		ship.OrbitDistance = planet.GetOrbitDistance()
		ship.OrbitAngle = planet.GetOrbitAngle()
		ship.Status = entities.ShipStatusOrbiting
	}
}

// moveFleetToStar moves a fleet to orbit the star in the current system
func (fui *FleetInfoUI) moveFleetToStar() {
	if fui.fleet == nil || len(fui.fleet.Ships) == 0 {
		return
	}

	// Move all ships to a mid-range star orbit
	for _, ship := range fui.fleet.Ships {
		ship.OrbitDistance = 150.0 // Standard star orbit distance
		ship.OrbitAngle = 0.0
		ship.Status = entities.ShipStatusOrbiting
	}
}
