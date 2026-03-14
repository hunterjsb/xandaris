package ui

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/views"
	"github.com/hunterjsb/xandaris/utils"
)

// FleetInfoUI displays detailed information about a selected fleet or ship
type FleetInfoUI struct {
	ctx                  UIContext
	fleet                 *entities.Fleet
	ship                  *entities.Ship // If showing a single ship (not a fleet)
	x                     int
	y                     int
	width                 int
	height                int
	visible               bool
	scrollOffset          int
	showMoveMenu          bool
	showJoinFleetMenu     bool
	showCargoMenu         bool
	nearbyFleets          []*entities.Fleet
	connectedSystems      []int
	currentSystemEntities []entities.Entity
	moveMenuScrollOffset  int
	joinMenuScrollOffset  int
	cargoMenuScrollOffset int
	cargoOrbitPlanet      *entities.Planet // planet the ship is orbiting (for cargo ops)
	cargoStatusMsg        string
	cargoStatusTicks      int
}

// NewFleetInfoUI creates a new fleet info UI
func NewFleetInfoUI(ctx UIContext) *FleetInfoUI {
	return &FleetInfoUI{
		ctx:   ctx,
		x:      1280 - 320,
		y:      80,
		width:  310,
		height: 400,
	}
}

// ShowFleet displays the fleet info for a specific fleet
func (fui *FleetInfoUI) ShowFleet(fleet *entities.Fleet) {
	fui.fleet = fleet
	fui.ship = nil // Clear any ship
	fui.visible = true
	fui.scrollOffset = 0
	fui.showMoveMenu = false
	fui.moveMenuScrollOffset = 0
}

// ShowShip displays info for a single ship (not in a fleet)
func (fui *FleetInfoUI) ShowShip(ship *entities.Ship) {
	fui.ship = ship
	fui.fleet = nil // Clear any fleet
	fui.visible = true
	fui.scrollOffset = 0
	fui.showMoveMenu = false
	fui.showJoinFleetMenu = false
	fui.showCargoMenu = false
	fui.moveMenuScrollOffset = 0
	fui.joinMenuScrollOffset = 0
	fui.cargoMenuScrollOffset = 0
	fui.cargoStatusMsg = ""
}

// Hide closes the fleet info UI
func (fui *FleetInfoUI) Hide() {
	fui.visible = false
	fui.fleet = nil
	fui.ship = nil
	fui.showMoveMenu = false
}

// IsVisible returns whether the UI is visible
func (fui *FleetInfoUI) IsVisible() bool {
	return fui.visible
}

// Update handles input for the fleet info UI
func (fui *FleetInfoUI) Update() {
	if !fui.visible || (fui.fleet == nil && fui.ship == nil) {
		return
	}

	// Handle mouse wheel scrolling
	_, dy := ebiten.Wheel()
	if dy != 0 {
		fui.scrollOffset -= int(dy * 20)
		// Clamp scroll offset
		shipCount := 1
		if fui.fleet != nil {
			shipCount = len(fui.fleet.Ships)
		}
		maxScroll := shipCount*60 - (fui.height - 100)
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

		// Handle button clicks based on whether showing fleet or ship
		if !fui.showMoveMenu && !fui.showJoinFleetMenu && !fui.showCargoMenu {
			buttonY := fui.y + fui.height - 40
			buttonH := 30

			if fui.fleet != nil {
				// Fleet has 2 buttons: Move, Disband
				buttonW := (fui.width - 30) / 2

				// Move button (left)
				if mx >= fui.x+10 && mx <= fui.x+10+buttonW &&
					my >= buttonY && my <= buttonY+buttonH {
					fui.showMoveMenu = true
					fui.initializeMoveMenu()
					return
				}

				// Disband button (right)
				if mx >= fui.x+20+buttonW && mx <= fui.x+20+buttonW+buttonW &&
					my >= buttonY && my <= buttonY+buttonH {
					fui.disbandFleet()
					return
				}

			} else if fui.ship != nil {
				atPlanet := fui.isFleetAtPlanet()
				var numButtons int
				if atPlanet {
					numButtons = 4 // Move, Create, Join, Cargo
				} else {
					numButtons = 3 // Move, Create, Join
				}
				buttonW := (fui.width - 10*(numButtons+1)) / numButtons

				// Move button
				if mx >= fui.x+10 && mx <= fui.x+10+buttonW &&
					my >= buttonY && my <= buttonY+buttonH {
					fui.showMoveMenu = true
					fui.initializeMoveMenu()
					return
				}

				// Create Fleet button
				createX := fui.x + 20 + buttonW
				if mx >= createX && mx <= createX+buttonW &&
					my >= buttonY && my <= buttonY+buttonH {
					fui.createFleetFromShip()
					return
				}

				// Join Fleet button
				joinX := fui.x + 30 + buttonW*2
				if mx >= joinX && mx <= joinX+buttonW &&
					my >= buttonY && my <= buttonY+buttonH {
					fui.showJoinFleetMenu = true
					fui.initializeJoinFleetMenu()
					return
				}

				// Cargo button (only when at planet)
				if atPlanet {
					cargoX := fui.x + 40 + buttonW*3
					if mx >= cargoX && mx <= cargoX+buttonW &&
						my >= buttonY && my <= buttonY+buttonH {
						fui.showCargoMenu = true
						fui.initializeCargoMenu()
						return
					}
				}
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
					// Move fleet/ship to this system (inter-system jump)
					if fui.fleet != nil {
						fleetCommander := fui.ctx.GetFleetCommander()
						success, _ := fleetCommander.MoveFleetToSystem(fui.fleet, systemID)
						if success > 0 {
							fui.showMoveMenu = false
						}
					} else if fui.ship != nil {
						// Move single ship
						helper := tickable.NewShipMovementHelper(fui.ctx.GetSystemsMap(), fui.ctx.GetHyperlanes())
						if helper.StartJourney(fui.ship, systemID) {
							fui.showMoveMenu = false
							fui.Hide() // Close UI since ship is now moving
						}
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

		// Handle join fleet menu clicks
		if fui.showJoinFleetMenu {
			// Back button
			backButtonX := fui.x + 10
			backButtonY := fui.y + fui.height - 40
			backButtonW := 60
			backButtonH := 30
			if mx >= backButtonX && mx <= backButtonX+backButtonW &&
				my >= backButtonY && my <= backButtonY+backButtonH {
				fui.showJoinFleetMenu = false
				return
			}

			// Fleet list clicks
			listStartY := fui.y + 110
			itemHeight := 40
			for i, fleet := range fui.nearbyFleets {
				itemY := listStartY + i*itemHeight - fui.joinMenuScrollOffset
				if itemY < listStartY-itemHeight || itemY > fui.y+fui.height-60 {
					continue
				}
				if mx >= fui.x+10 && mx <= fui.x+fui.width-10 &&
					my >= itemY && my <= itemY+itemHeight-5 {
					// Join this fleet
					fui.joinSelectedFleet(fleet)
					fui.showJoinFleetMenu = false
					return
				}
			}
		}

		// Handle cargo menu clicks
		if fui.showCargoMenu {
			// Back button
			backButtonX := fui.x + 10
			backButtonY := fui.y + fui.height - 40
			backButtonW := 60
			backButtonH := 30
			if mx >= backButtonX && mx <= backButtonX+backButtonW &&
				my >= backButtonY && my <= backButtonY+backButtonH {
				fui.showCargoMenu = false
				return
			}

			// Cargo action buttons (Load/Unload per resource row)
			fui.handleCargoMenuClick(mx, my)
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

	// Handle scroll for join fleet menu
	if fui.showJoinFleetMenu {
		_, dy := ebiten.Wheel()
		if dy != 0 {
			fui.joinMenuScrollOffset -= int(dy * 20)
			maxScroll := len(fui.nearbyFleets)*40 - (fui.height - 160)
			if maxScroll < 0 {
				maxScroll = 0
			}
			if fui.joinMenuScrollOffset < 0 {
				fui.joinMenuScrollOffset = 0
			}
			if fui.joinMenuScrollOffset > maxScroll {
				fui.joinMenuScrollOffset = maxScroll
			}
		}
	}

	// Handle scroll for cargo menu
	if fui.showCargoMenu {
		_, dy := ebiten.Wheel()
		if dy != 0 {
			fui.cargoMenuScrollOffset -= int(dy * 20)
			if fui.cargoMenuScrollOffset < 0 {
				fui.cargoMenuScrollOffset = 0
			}
			maxScroll := fui.getCargoMenuContentHeight() - (fui.height - 200)
			if maxScroll < 0 {
				maxScroll = 0
			}
			if fui.cargoMenuScrollOffset > maxScroll {
				fui.cargoMenuScrollOffset = maxScroll
			}
		}
	}

	// Decay cargo status message
	if fui.cargoStatusTicks > 0 {
		fui.cargoStatusTicks--
		if fui.cargoStatusTicks <= 0 {
			fui.cargoStatusMsg = ""
		}
	}
}

// Draw renders the fleet info panel
func (fui *FleetInfoUI) Draw(screen *ebiten.Image) {
	if !fui.visible || (fui.fleet == nil && fui.ship == nil) {
		return
	}

	// Background panel
	panel := &views.UIPanel{
		X:           fui.x,
		Y:           fui.y,
		Width:       fui.width,
		Height:      fui.height,
		BgColor:     utils.Background,
		BorderColor: utils.PanelBorder,
	}
	panel.Draw(screen)

	// Title
	titleText := "Fleet Details"
	if fui.ship != nil {
		titleText = "Ship Details"
	}
	views.DrawText(screen, titleText, fui.x+10, fui.y+15, utils.SystemLightBlue)

	// Close button
	closeX := fui.x + fui.width - 30
	closeY := fui.y + 10
	views.DrawText(screen, "[X]", closeX, closeY, utils.SystemRed)

	// Summary - different for ship vs fleet
	summaryY := fui.y + 40

	var ships []*entities.Ship
	var owner string
	if fui.fleet != nil {
		ships = fui.fleet.Ships
		owner = fui.fleet.GetOwner()
		views.DrawText(screen, fmt.Sprintf("Ships: %d", fui.fleet.Size()), fui.x+10, summaryY, utils.TextPrimary)
	} else if fui.ship != nil {
		ships = []*entities.Ship{fui.ship}
		owner = fui.ship.Owner
		views.DrawText(screen, fmt.Sprintf("Ship: %s", fui.ship.ShipType), fui.x+10, summaryY, utils.TextPrimary)
	}
	views.DrawText(screen, fmt.Sprintf("Owner: %s", owner), fui.x+10, summaryY+15, utils.TextSecondary)

	// Fuel stats
	fuelY := summaryY + 35
	var avgFuel float64
	var totalFuel, totalMaxFuel int

	for _, ship := range ships {
		avgFuel += ship.GetFuelPercentage()
		totalFuel += ship.CurrentFuel
		totalMaxFuel += ship.MaxFuel
	}
	if len(ships) > 0 {
		avgFuel /= float64(len(ships))
	}

	fuelColor := utils.StationResearch // Green for good fuel
	if avgFuel < 25 {
		fuelColor = utils.SystemRed // Red for low fuel
	} else if avgFuel < 50 {
		fuelColor = utils.SystemOrange // Orange for medium fuel
	}
	views.DrawText(screen, fmt.Sprintf("Avg Fuel: %.0f%%", avgFuel), fui.x+10, fuelY, fuelColor)
	views.DrawText(screen, fmt.Sprintf("Total: %d/%d", totalFuel, totalMaxFuel),
		fui.x+10, fuelY+15, utils.TextSecondary)

	// Separator
	separatorY := fuelY + 35
	views.DrawLine(screen, fui.x+10, separatorY, fui.x+fui.width-10, separatorY, utils.PanelBorder)

	// Show either the ship list, move menu, join fleet menu, or cargo menu
	if fui.showMoveMenu {
		fui.drawMoveMenu(screen)
	} else if fui.showJoinFleetMenu {
		fui.drawJoinFleetMenu(screen)
	} else if fui.showCargoMenu {
		fui.drawCargoMenu(screen)
	} else {
		// Ship list header
		listHeaderY := separatorY + 10
		views.DrawText(screen, "Ships:", fui.x+10, listHeaderY, utils.TextPrimary)

		// Scrollable ship list
		fui.drawShipList(screen, listHeaderY+20)

		// Move button
		fui.drawMoveButton(screen)

		// Scroll indicator for ship list
		shipCount := 1
		if fui.fleet != nil {
			shipCount = len(fui.fleet.Ships)
		}
		if shipCount > 5 {
			scrollHintY := fui.y + fui.height - 50
			views.DrawTextCentered(screen, "Scroll for more", fui.x+fui.width/2, scrollHintY, utils.TextSecondary, 0.7)
		}
	}
}

// drawMoveButton draws the action buttons (Move, Create/Disband/Join Fleet)
func (fui *FleetInfoUI) drawMoveButton(screen *ebiten.Image) {
	buttonY := fui.y + fui.height - 40
	buttonH := 30

	// Check if can move
	var canMove int
	if fui.fleet != nil {
		canMove, _, _ = fui.fleet.GetMovementStatus()
	} else if fui.ship != nil {
		if fui.ship.CanJump() {
			canMove = 1
		}
	}

	moveButtonColor := utils.ButtonActive
	moveButtonText := "Move"
	if canMove == 0 {
		moveButtonColor = utils.ButtonDisabled
		moveButtonText = "No Fuel"
	}

	// For fleets: 2 buttons (Move, Disband)
	if fui.fleet != nil {
		buttonW := (fui.width - 30) / 2

		// Move button (left)
		movePanel := &views.UIPanel{
			X:           fui.x + 10,
			Y:           buttonY,
			Width:       buttonW,
			Height:      buttonH,
			BgColor:     moveButtonColor,
			BorderColor: utils.Highlight,
		}
		movePanel.Draw(screen)
		views.DrawTextCentered(screen, moveButtonText, fui.x+10+buttonW/2, buttonY+10, utils.TextPrimary, 1.0)

		// Disband button (right)
		disbandPanel := &views.UIPanel{
			X:           fui.x + 20 + buttonW,
			Y:           buttonY,
			Width:       buttonW,
			Height:      buttonH,
			BgColor:     utils.SystemRed,
			BorderColor: utils.Highlight,
		}
		disbandPanel.Draw(screen)
		views.DrawTextCentered(screen, "Disband", fui.x+20+buttonW+buttonW/2, buttonY+10, utils.TextPrimary, 1.0)

	} else if fui.ship != nil {
		// For ships: 3 or 4 buttons depending on whether ship is at a planet
		atPlanet := fui.isFleetAtPlanet()
		var numButtons int
		if atPlanet {
			numButtons = 4 // Move, Create, Join, Cargo
		} else {
			numButtons = 3 // Move, Create, Join
		}
		buttonW := (fui.width - 10*(numButtons+1)) / numButtons

		// Move button
		movePanel := &views.UIPanel{
			X:           fui.x + 10,
			Y:           buttonY,
			Width:       buttonW,
			Height:      buttonH,
			BgColor:     moveButtonColor,
			BorderColor: utils.Highlight,
		}
		movePanel.Draw(screen)
		views.DrawTextCentered(screen, moveButtonText, fui.x+10+buttonW/2, buttonY+10, utils.TextPrimary, 0.9)

		// Create Fleet button
		createX := fui.x + 20 + buttonW
		createPanel := &views.UIPanel{
			X:           createX,
			Y:           buttonY,
			Width:       buttonW,
			Height:      buttonH,
			BgColor:     color.RGBA{50, 100, 50, 255},
			BorderColor: utils.Highlight,
		}
		createPanel.Draw(screen)
		views.DrawTextCentered(screen, "Create", createX+buttonW/2, buttonY+10, utils.TextPrimary, 0.9)

		// Join Fleet button
		joinX := fui.x + 30 + buttonW*2
		joinPanel := &views.UIPanel{
			X:           joinX,
			Y:           buttonY,
			Width:       buttonW,
			Height:      buttonH,
			BgColor:     color.RGBA{50, 50, 100, 255},
			BorderColor: utils.Highlight,
		}
		joinPanel.Draw(screen)
		views.DrawTextCentered(screen, "Join", joinX+buttonW/2, buttonY+10, utils.TextPrimary, 0.9)

		// Cargo button (only when at planet)
		if atPlanet {
			cargoX := fui.x + 40 + buttonW*3
			cargoPanel := &views.UIPanel{
				X:           cargoX,
				Y:           buttonY,
				Width:       buttonW,
				Height:      buttonH,
				BgColor:     color.RGBA{100, 80, 30, 255},
				BorderColor: utils.Highlight,
			}
			cargoPanel.Draw(screen)
			views.DrawTextCentered(screen, "Cargo", cargoX+buttonW/2, buttonY+10, utils.TextPrimary, 0.9)
		}
	}
}

// drawMoveMenu draws the destination selection menu
func (fui *FleetInfoUI) drawMoveMenu(screen *ebiten.Image) {
	// Title (fixed, doesn't scroll)
	menuTitleY := fui.y + 100
	views.DrawText(screen, "Select Destination:", fui.x+10, menuTitleY, utils.SystemLightBlue)

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
		views.DrawText(screen, "Jump to System:", fui.x+10, currentY, utils.TextPrimary)
	}
	currentY += headerHeight

	if len(fui.connectedSystems) == 0 {
		if currentY >= visibleTop-15 && currentY <= visibleBottom {
			views.DrawText(screen, "  No adjacent systems", fui.x+20, currentY+5, utils.TextSecondary)
		}
		currentY += 15
	}

	systems := fui.ctx.GetSystemsMap()
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
			views.DrawText(screen, "Move to Star:", fui.x+10, currentY, utils.TextPrimary)
		}
		currentY += headerHeight

		itemY := currentY
		if itemY >= visibleTop-itemHeight && itemY <= visibleBottom {
			// Get star color from current system
			var firstShip *entities.Ship
			if fui.fleet != nil && len(fui.fleet.Ships) > 0 {
				firstShip = fui.fleet.Ships[0]
			} else if fui.ship != nil {
				firstShip = fui.ship
			}

			if firstShip != nil {
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
		views.DrawText(screen, "Move to Planet:", fui.x+10, currentY, utils.TextPrimary)
	}
	currentY += headerHeight

	if len(fui.currentSystemEntities) == 0 {
		if currentY >= visibleTop-15 && currentY <= visibleBottom {
			views.DrawText(screen, "  No planets in system", fui.x+20, currentY+5, utils.TextSecondary)
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

	backPanel := &views.UIPanel{
		X:           backButtonX,
		Y:           backButtonY,
		Width:       backButtonW,
		Height:      backButtonH,
		BgColor:     utils.ButtonActive,
		BorderColor: utils.Highlight,
	}
	backPanel.Draw(screen)
	views.DrawTextCentered(screen, "Back", backButtonX+backButtonW/2, backButtonY+10, utils.TextPrimary, 1.0)

	// Scroll hint
	totalItems := len(fui.connectedSystems) + len(fui.currentSystemEntities)
	if totalItems > 5 {
		scrollHintY := fui.y + fui.height - 50
		views.DrawTextCentered(screen, "Scroll for more", fui.x+fui.width/2, scrollHintY, utils.TextSecondary, 0.7)
	}
}

// drawJoinFleetMenu draws the menu for selecting a fleet to join
func (fui *FleetInfoUI) drawJoinFleetMenu(screen *ebiten.Image) {
	// Title (fixed, doesn't scroll)
	menuTitleY := fui.y + 100
	views.DrawText(screen, "Select Fleet to Join:", fui.x+10, menuTitleY, utils.SystemLightBlue)

	// Check if there are any nearby fleets
	if len(fui.nearbyFleets) == 0 {
		noFleetsY := fui.y + fui.height/2
		views.DrawTextCentered(screen, "No nearby fleets", fui.x+fui.width/2, noFleetsY, utils.TextSecondary, 1.0)
		views.DrawTextCentered(screen, "(within same orbit)", fui.x+fui.width/2, noFleetsY+15, utils.TextSecondary, 0.8)
	} else {
		// List of nearby fleets
		listStartY := fui.y + 110
		itemHeight := 40

		for i, fleet := range fui.nearbyFleets {
			itemY := listStartY + i*itemHeight - fui.joinMenuScrollOffset

			// Don't draw off screen
			if itemY < listStartY-itemHeight || itemY > fui.y+fui.height-60 {
				continue
			}

			// Fleet item
			fleetName := fmt.Sprintf("Fleet %d (%d ships)", fleet.ID, fleet.Size())
			itemColor := fleet.GetColor()

			fui.drawMenuItem(screen, itemY, itemHeight, itemColor, fleetName)

			// Show ship type breakdown
			typeCounts := fleet.GetShipTypeCounts()
			typeY := itemY + 20
			typeText := ""
			for shipType, count := range typeCounts {
				if typeText != "" {
					typeText += ", "
				}
				typeText += fmt.Sprintf("%dx%s", count, shipType)
			}
			views.DrawText(screen, typeText, fui.x+35, typeY, utils.TextSecondary)
		}
	}

	// Back button
	backButtonX := fui.x + 10
	backButtonY := fui.y + fui.height - 40
	backButtonW := 60
	backButtonH := 30

	backPanel := &views.UIPanel{
		X:           backButtonX,
		Y:           backButtonY,
		Width:       backButtonW,
		Height:      backButtonH,
		BgColor:     utils.ButtonActive,
		BorderColor: utils.Highlight,
	}
	backPanel.Draw(screen)
	views.DrawTextCentered(screen, "Back", backButtonX+backButtonW/2, backButtonY+10, utils.TextPrimary, 1.0)

	// Scroll hint
	if len(fui.nearbyFleets) > 5 {
		scrollHintY := fui.y + fui.height - 50
		views.DrawTextCentered(screen, "Scroll for more", fui.x+fui.width/2, scrollHintY, utils.TextSecondary, 0.7)
	}
}

// drawShipList draws the scrollable list of ships
func (fui *FleetInfoUI) drawShipList(screen *ebiten.Image, startY int) {
	itemHeight := 60

	var ships []*entities.Ship
	if fui.fleet != nil {
		ships = fui.fleet.Ships
	} else if fui.ship != nil {
		ships = []*entities.Ship{fui.ship}
	}

	for i, ship := range ships {
		itemY := startY + i*itemHeight - fui.scrollOffset

		// Skip if off screen
		if itemY < startY-itemHeight || itemY > fui.y+fui.height-30 {
			continue
		}

		// Ship item background
		itemPanel := &views.UIPanel{
			X:           fui.x + 10,
			Y:           itemY,
			Width:       fui.width - 20,
			Height:      itemHeight - 5,
			BgColor:     utils.PanelBg,
			BorderColor: utils.PanelBorder,
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
		views.DrawText(screen, ship.Name, nameX, itemY+8, utils.TextPrimary)
		views.DrawText(screen, string(ship.ShipType), nameX, itemY+23, utils.TextSecondary)

		// Ship stats
		statsY := itemY + 38
		fuelPercent := ship.GetFuelPercentage()
		fuelColor := utils.TextPrimary
		if fuelPercent < 25 {
			fuelColor = utils.SystemRed
		} else if fuelPercent < 50 {
			fuelColor = utils.SystemOrange
		}

		views.DrawText(screen, fmt.Sprintf("Fuel: %d/%d", ship.CurrentFuel, ship.MaxFuel), nameX, statsY, fuelColor)

		// Cargo usage
		if ship.MaxCargo > 0 {
			cargoUsed := ship.GetTotalCargo()
			cargoColor := utils.TextSecondary
			if cargoUsed > 0 {
				cargoColor = utils.TextPrimary
			}
			views.DrawText(screen, fmt.Sprintf("Cargo: %d/%d", cargoUsed, ship.MaxCargo), nameX+100, statsY, cargoColor)
		} else {
			healthPercent := ship.GetHealthPercentage()
			healthColor := utils.TextPrimary
			if healthPercent < 50 {
				healthColor = utils.SystemOrange
			}
			if healthPercent < 25 {
				healthColor = utils.SystemRed
			}
			views.DrawText(screen, fmt.Sprintf("HP: %.0f%%", healthPercent), nameX+100, statsY, healthColor)
		}

		// Status indicator
		statusX := fui.x + fui.width - 80
		statusText := string(ship.Status)
		statusColor := utils.TextSecondary
		if ship.Status == entities.ShipStatusMoving {
			statusColor = utils.SystemBlue
		}
		views.DrawText(screen, statusText, statusX, itemY+8, statusColor)
	}
}

// GetFleet returns the currently displayed fleet
func (fui *FleetInfoUI) GetFleet() *entities.Fleet {
	return fui.fleet
}

// drawMenuItem draws a single menu item (helper function)
func (fui *FleetInfoUI) drawMenuItem(screen *ebiten.Image, y int, height int, itemColor color.RGBA, text string) {
	// Item background
	itemPanel := &views.UIPanel{
		X:           fui.x + 10,
		Y:           y,
		Width:       fui.width - 20,
		Height:      height - 5,
		BgColor:     utils.PanelBg,
		BorderColor: utils.PanelBorder,
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
	views.DrawText(screen, text, fui.x+35, y+10, utils.TextPrimary)
}

// isFleetAtPlanet checks if the fleet/ship is currently orbiting a planet
func (fui *FleetInfoUI) isFleetAtPlanet() bool {
	var firstShip *entities.Ship
	if fui.fleet != nil && len(fui.fleet.Ships) > 0 {
		firstShip = fui.fleet.Ships[0]
	} else if fui.ship != nil {
		firstShip = fui.ship
	}

	if firstShip == nil {
		return false
	}

	systems := fui.ctx.GetSystemsMap()
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

// moveFleetToPlanet moves a fleet/ship to orbit a planet in the current system
func (fui *FleetInfoUI) moveFleetToPlanet(entity entities.Entity) {
	planet, ok := entity.(*entities.Planet)
	if !ok {
		return
	}

	var ships []*entities.Ship
	if fui.fleet != nil {
		ships = fui.fleet.Ships
	} else if fui.ship != nil {
		ships = []*entities.Ship{fui.ship}
	}

	// Move all ships to the planet's orbit
	for _, ship := range ships {
		ship.OrbitDistance = planet.GetOrbitDistance()
		ship.OrbitAngle = planet.GetOrbitAngle()
		ship.Status = entities.ShipStatusOrbiting
	}
}

// moveFleetToStar moves a fleet/ship to orbit the star in the current system
func (fui *FleetInfoUI) moveFleetToStar() {
	var ships []*entities.Ship
	if fui.fleet != nil {
		ships = fui.fleet.Ships
	} else if fui.ship != nil {
		ships = []*entities.Ship{fui.ship}
	}

	// Move all ships to a mid-range star orbit
	for _, ship := range ships {
		ship.OrbitDistance = 150.0 // Standard star orbit distance
		ship.OrbitAngle = 0.0
		ship.Status = entities.ShipStatusOrbiting
	}
}

// disbandFleet breaks up the fleet into individual ships
func (fui *FleetInfoUI) disbandFleet() {
	if fui.fleet == nil {
		return
	}

	// Find the owner
	humanPlayer := fui.ctx.GetState().HumanPlayer
	if humanPlayer == nil || fui.fleet.GetOwner() != humanPlayer.Name {
		return
	}

	// Get the fleet management system from app
	fleetMgmt := fui.ctx.GetFleetManagementSystem()

	err := fleetMgmt.DisbandFleet(fui.fleet, humanPlayer)
	if err != nil {
		fmt.Printf("[FleetInfoUI] Error disbanding fleet: %v\n", err)
		return
	}

	// Close the UI since the fleet no longer exists
	fui.Hide()
}

// initializeMoveMenu prepares data for the move menu
func (fui *FleetInfoUI) initializeMoveMenu() {
	var firstShip *entities.Ship
	if fui.fleet != nil && len(fui.fleet.Ships) > 0 {
		firstShip = fui.fleet.Ships[0]
	} else if fui.ship != nil {
		firstShip = fui.ship
	}

	if firstShip != nil {
		helper := tickable.NewShipMovementHelper(fui.ctx.GetSystemsMap(), fui.ctx.GetHyperlanes())
		fui.connectedSystems = helper.GetConnectedSystems(firstShip.CurrentSystem)

		// Get current system entities (planets)
		systems := fui.ctx.GetSystemsMap()
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
}

// initializeJoinFleetMenu prepares data for the join fleet menu
func (fui *FleetInfoUI) initializeJoinFleetMenu() {
	if fui.ship == nil {
		return
	}

	// Find the owner
	humanPlayer := fui.ctx.GetState().HumanPlayer
	if humanPlayer == nil || fui.ship.Owner != humanPlayer.Name {
		return
	}

	// Get the fleet management system and find nearby fleets
	fleetMgmt := fui.ctx.GetFleetManagementSystem()
	fui.nearbyFleets = fleetMgmt.GetNearbyFleets(fui.ship, humanPlayer)
}

// createFleetFromShip promotes a single ship to a fleet
func (fui *FleetInfoUI) createFleetFromShip() {
	if fui.ship == nil {
		return
	}

	// Find the owner
	humanPlayer := fui.ctx.GetState().HumanPlayer
	if humanPlayer == nil || fui.ship.Owner != humanPlayer.Name {
		return
	}

	// Get the fleet management system
	fleetMgmt := fui.ctx.GetFleetManagementSystem()

	newFleet, err := fleetMgmt.CreateFleetFromShip(fui.ship, humanPlayer)
	if err != nil {
		fmt.Printf("[FleetInfoUI] Error creating fleet: %v\n", err)
		return
	}

	// Update the UI to show the real fleet instead of the ship
	fui.ShowFleet(newFleet)
}

// joinSelectedFleet adds the ship to the selected fleet
func (fui *FleetInfoUI) joinSelectedFleet(fleet *entities.Fleet) {
	if fui.ship == nil || fleet == nil {
		return
	}

	// Find the owner
	humanPlayer := fui.ctx.GetState().HumanPlayer
	if humanPlayer == nil || fui.ship.Owner != humanPlayer.Name {
		return
	}

	// Get the fleet management system
	fleetMgmt := fui.ctx.GetFleetManagementSystem()

	err := fleetMgmt.AddShipToFleet(fui.ship, fleet, humanPlayer)
	if err != nil {
		fmt.Printf("[FleetInfoUI] Error joining fleet: %v\n", err)
		return
	}

	// Update the UI to show the fleet the ship just joined
	fui.ShowFleet(fleet)
}

// initializeCargoMenu finds the planet the ship is orbiting for cargo ops
func (fui *FleetInfoUI) initializeCargoMenu() {
	fui.cargoOrbitPlanet = nil
	fui.cargoMenuScrollOffset = 0
	fui.cargoStatusMsg = ""

	ship := fui.ship
	if ship == nil {
		return
	}

	systems := fui.ctx.GetSystemsMap()
	currentSystem := systems[ship.CurrentSystem]
	if currentSystem == nil {
		return
	}

	// Find the planet the ship is orbiting
	for _, entity := range currentSystem.Entities {
		if planet, ok := entity.(*entities.Planet); ok {
			if math.Abs(ship.GetOrbitDistance()-planet.GetOrbitDistance()) < 1.0 {
				fui.cargoOrbitPlanet = planet
				return
			}
		}
	}
}

const cargoLot = 10 // units per load/unload click

// drawCargoMenu draws the cargo management interface
func (fui *FleetInfoUI) drawCargoMenu(screen *ebiten.Image) {
	ship := fui.ship
	if ship == nil {
		return
	}

	// Title
	menuTitleY := fui.y + 100
	if fui.cargoOrbitPlanet != nil {
		views.DrawText(screen, fmt.Sprintf("Cargo at %s", fui.cargoOrbitPlanet.Name), fui.x+10, menuTitleY, utils.SystemLightBlue)
	} else {
		views.DrawText(screen, "Cargo Management", fui.x+10, menuTitleY, utils.SystemLightBlue)
	}

	// Cargo capacity bar
	capacityY := menuTitleY + 18
	totalCargo := ship.GetTotalCargo()
	capacityText := fmt.Sprintf("Hold: %d/%d", totalCargo, ship.MaxCargo)
	views.DrawText(screen, capacityText, fui.x+10, capacityY, utils.TextPrimary)

	// Draw capacity bar
	barX := fui.x + 10
	barY := capacityY + 14
	barW := fui.width - 20
	barH := 6
	barPanel := &views.UIPanel{X: barX, Y: barY, Width: barW, Height: barH, BgColor: utils.PanelBg, BorderColor: utils.PanelBorder}
	barPanel.Draw(screen)
	if ship.MaxCargo > 0 {
		fillW := int(float64(barW) * float64(totalCargo) / float64(ship.MaxCargo))
		if fillW > 0 {
			fillColor := color.RGBA{180, 150, 50, 255}
			if totalCargo >= ship.MaxCargo {
				fillColor = utils.SystemRed
			}
			fillPanel := &views.UIPanel{X: barX + 1, Y: barY + 1, Width: fillW - 2, Height: barH - 2, BgColor: fillColor, BorderColor: fillColor}
			fillPanel.Draw(screen)
		}
	}

	// Status message
	if fui.cargoStatusMsg != "" {
		views.DrawText(screen, fui.cargoStatusMsg, fui.x+10, barY+12, utils.SystemGreen)
	}

	// Content area starts after header
	contentStartY := barY + 28
	itemHeight := 28
	visibleBottom := fui.y + fui.height - 60
	currentY := contentStartY - fui.cargoMenuScrollOffset

	// Section 1: Planet Resources (loadable)
	if fui.cargoOrbitPlanet != nil && fui.cargoOrbitPlanet.Owner == ship.Owner {
		if currentY >= contentStartY-20 && currentY <= visibleBottom {
			views.DrawText(screen, "Planet Stock:", fui.x+10, currentY, utils.TextPrimary)
		}
		currentY += 18

		hasResources := false
		for resType, storage := range fui.cargoOrbitPlanet.StoredResources {
			if storage == nil || storage.Amount <= 0 {
				continue
			}
			hasResources = true
			if currentY >= contentStartY-itemHeight && currentY <= visibleBottom {
				// Resource name and amount
				views.DrawText(screen, fmt.Sprintf("%s: %d", resType, storage.Amount), fui.x+15, currentY+4, utils.TextPrimary)

				// [Load] button
				loadX := fui.x + fui.width - 65
				views.DrawText(screen, "[Load]", loadX, currentY+4, utils.SystemGreen)
			}
			currentY += itemHeight
		}
		if !hasResources {
			if currentY >= contentStartY-15 && currentY <= visibleBottom {
				views.DrawText(screen, "  No resources", fui.x+20, currentY+4, utils.TextSecondary)
			}
			currentY += 18
		}
		currentY += 8
	}

	// Section 2: Ship Cargo (unloadable)
	if currentY >= contentStartY-20 && currentY <= visibleBottom {
		views.DrawText(screen, "Ship Cargo:", fui.x+10, currentY, utils.TextPrimary)
	}
	currentY += 18

	if len(ship.CargoHold) == 0 {
		if currentY >= contentStartY-15 && currentY <= visibleBottom {
			views.DrawText(screen, "  Empty", fui.x+20, currentY+4, utils.TextSecondary)
		}
		currentY += 18
	} else {
		for resType, amount := range ship.CargoHold {
			if amount <= 0 {
				continue
			}
			if currentY >= contentStartY-itemHeight && currentY <= visibleBottom {
				views.DrawText(screen, fmt.Sprintf("%s: %d", resType, amount), fui.x+15, currentY+4, utils.TextPrimary)

				// [Unload] button (only if at owned planet)
				if fui.cargoOrbitPlanet != nil && fui.cargoOrbitPlanet.Owner == ship.Owner {
					unloadX := fui.x + fui.width - 80
					views.DrawText(screen, "[Unload]", unloadX, currentY+4, utils.SystemOrange)
				}
			}
			currentY += itemHeight
		}
	}

	// Back button
	backButtonX := fui.x + 10
	backButtonY := fui.y + fui.height - 40
	backButtonW := 60
	backButtonH := 30

	backPanel := &views.UIPanel{
		X:           backButtonX,
		Y:           backButtonY,
		Width:       backButtonW,
		Height:      backButtonH,
		BgColor:     utils.ButtonActive,
		BorderColor: utils.Highlight,
	}
	backPanel.Draw(screen)
	views.DrawTextCentered(screen, "Back", backButtonX+backButtonW/2, backButtonY+10, utils.TextPrimary, 1.0)
}

// handleCargoMenuClick processes clicks in the cargo menu
func (fui *FleetInfoUI) handleCargoMenuClick(mx, my int) {
	ship := fui.ship
	if ship == nil {
		return
	}

	barY := fui.y + 100 + 18 + 14 // matches draw layout
	contentStartY := barY + 28
	itemHeight := 28
	visibleBottom := fui.y + fui.height - 60
	currentY := contentStartY - fui.cargoMenuScrollOffset

	cargoCommander := fui.ctx.GetCargoCommander()
	if cargoCommander == nil {
		return
	}

	// Section 1: Planet resources — [Load] buttons
	if fui.cargoOrbitPlanet != nil && fui.cargoOrbitPlanet.Owner == ship.Owner {
		currentY += 18 // header

		for resType, storage := range fui.cargoOrbitPlanet.StoredResources {
			if storage == nil || storage.Amount <= 0 {
				continue
			}
			if currentY >= contentStartY-itemHeight && currentY <= visibleBottom {
				loadX := fui.x + fui.width - 65
				loadW := 50
				if mx >= loadX && mx <= loadX+loadW && my >= currentY && my <= currentY+itemHeight {
					loaded, err := cargoCommander.LoadCargo(ship, fui.cargoOrbitPlanet, resType, cargoLot)
					if err != nil {
						fui.cargoStatusMsg = err.Error()
					} else {
						fui.cargoStatusMsg = fmt.Sprintf("Loaded %d %s", loaded, resType)
					}
					fui.cargoStatusTicks = 120
					return
				}
			}
			currentY += itemHeight
		}
		currentY += 8
	}

	// Section 2: Ship cargo — [Unload] buttons
	currentY += 18 // header

	if fui.cargoOrbitPlanet != nil && fui.cargoOrbitPlanet.Owner == ship.Owner {
		for resType, amount := range ship.CargoHold {
			if amount <= 0 {
				continue
			}
			if currentY >= contentStartY-itemHeight && currentY <= visibleBottom {
				unloadX := fui.x + fui.width - 80
				unloadW := 65
				if mx >= unloadX && mx <= unloadX+unloadW && my >= currentY && my <= currentY+itemHeight {
					unloaded, err := cargoCommander.UnloadCargo(ship, fui.cargoOrbitPlanet, resType, cargoLot)
					if err != nil {
						fui.cargoStatusMsg = err.Error()
					} else {
						fui.cargoStatusMsg = fmt.Sprintf("Unloaded %d %s", unloaded, resType)
					}
					fui.cargoStatusTicks = 120
					return
				}
			}
			currentY += itemHeight
		}
	}
}

// getCargoMenuContentHeight calculates total height of cargo menu content
func (fui *FleetInfoUI) getCargoMenuContentHeight() int {
	itemHeight := 28
	total := 0

	// Planet section
	if fui.cargoOrbitPlanet != nil && fui.ship != nil && fui.cargoOrbitPlanet.Owner == fui.ship.Owner {
		total += 18 // header
		count := 0
		for _, storage := range fui.cargoOrbitPlanet.StoredResources {
			if storage != nil && storage.Amount > 0 {
				count++
			}
		}
		if count == 0 {
			total += 18
		} else {
			total += count * itemHeight
		}
		total += 8 // spacing
	}

	// Ship cargo section
	total += 18 // header
	cargoCount := len(fui.ship.CargoHold)
	if cargoCount == 0 {
		total += 18
	} else {
		total += cargoCount * itemHeight
	}

	return total
}
