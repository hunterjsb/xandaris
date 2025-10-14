package ui

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/views"
	"github.com/hunterjsb/xandaris/utils"
)

// ShipyardUI displays the ship construction interface
type ShipyardUI struct {
	ctx         UIContext
	planet       *entities.Planet
	shipyard     *entities.Building
	x            int
	y            int
	width        int
	height       int
	visible      bool
	selectedShip entities.ShipType
	hoveredShip  entities.ShipType
	shipTypes    []entities.ShipType
	scrollOffset int
	errorMessage string
	errorTimer   int
}

// NewShipyardUI creates a new shipyard UI
func NewShipyardUI(ctx UIContext) *ShipyardUI {
	return &ShipyardUI{
		ctx:   ctx,
		x:      1280/2 - 250,
		y:      720/2 - 250,
		width:  500,
		height: 500,
		shipTypes: []entities.ShipType{
			entities.ShipTypeScout,
			entities.ShipTypeColony,
			entities.ShipTypeCargo,
			entities.ShipTypeFrigate,
			entities.ShipTypeDestroyer,
			entities.ShipTypeCruiser,
		},
	}
}

// Show displays the shipyard UI for a specific planet and shipyard
func (sui *ShipyardUI) Show(planet *entities.Planet, shipyard *entities.Building) {
	sui.planet = planet
	sui.shipyard = shipyard
	sui.visible = true
	sui.scrollOffset = 0
	sui.selectedShip = ""
	sui.errorMessage = ""
}

// Hide closes the shipyard UI
func (sui *ShipyardUI) Hide() {
	sui.visible = false
	sui.planet = nil
	sui.shipyard = nil
}

// IsVisible returns whether the UI is visible
func (sui *ShipyardUI) IsVisible() bool {
	return sui.visible
}

// Update handles input for the shipyard UI
func (sui *ShipyardUI) Update() {
	if !sui.visible {
		return
	}

	// Decrement error timer
	if sui.errorTimer > 0 {
		sui.errorTimer--
		if sui.errorTimer == 0 {
			sui.errorMessage = ""
		}
	}

	// Handle escape key to close
	if sui.ctx.GetKeyBindings().IsActionJustPressed(views.ActionEscape) {
		sui.Hide()
		return
	}

	// Handle mouse input
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		sui.handleClick(mx, my)
	}

	// Handle mouse hover
	mx, my := ebiten.CursorPosition()
	sui.handleHover(mx, my)

	// Handle mouse wheel scrolling
	_, dy := ebiten.Wheel()
	if dy != 0 {
		sui.scrollOffset -= int(dy * 20) // Scroll speed
		// Clamp scroll offset
		maxScroll := len(sui.shipTypes)*70 - (sui.height - 150)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if sui.scrollOffset < 0 {
			sui.scrollOffset = 0
		}
		if sui.scrollOffset > maxScroll {
			sui.scrollOffset = maxScroll
		}
	}
}

// handleClick processes mouse clicks
func (sui *ShipyardUI) handleClick(mx, my int) {
	// Close button (top-right corner)
	closeX := sui.x + sui.width - 30
	closeY := sui.y + 10
	if mx >= closeX && mx <= closeX+20 && my >= closeY && my <= closeY+20 {
		sui.Hide()
		return
	}

	// Ship type selection
	startY := sui.y + 80
	itemHeight := 70
	for i, shipType := range sui.shipTypes {
		itemY := startY + i*itemHeight - sui.scrollOffset
		if itemY < sui.y+60 || itemY > sui.y+sui.height-80 {
			continue // Off screen
		}

		if mx >= sui.x+20 && mx <= sui.x+sui.width-20 &&
			my >= itemY && my <= itemY+itemHeight-10 {
			sui.selectedShip = shipType
			return
		}
	}

	// Build button
	buildButtonY := sui.y + sui.height - 60
	if sui.selectedShip != "" &&
		mx >= sui.x+sui.width/2-100 && mx <= sui.x+sui.width/2+100 &&
		my >= buildButtonY && my <= buildButtonY+40 {
		sui.buildShip()
	}
}

// handleHover updates the hovered ship
func (sui *ShipyardUI) handleHover(mx, my int) {
	sui.hoveredShip = ""

	startY := sui.y + 80
	itemHeight := 70
	for i, shipType := range sui.shipTypes {
		itemY := startY + i*itemHeight - sui.scrollOffset
		if itemY < sui.y+60 || itemY > sui.y+sui.height-80 {
			continue
		}

		if mx >= sui.x+20 && mx <= sui.x+sui.width-20 &&
			my >= itemY && my <= itemY+itemHeight-10 {
			sui.hoveredShip = shipType
			return
		}
	}
}

// buildShip attempts to build the selected ship
func (sui *ShipyardUI) buildShip() {
	if sui.selectedShip == "" || sui.planet == nil || sui.ctx.GetState().HumanPlayer == nil {
		return
	}

	// Check if player has enough credits
	cost := entities.GetShipBuildCost(sui.selectedShip)
	if sui.ctx.GetState().HumanPlayer.Credits < cost {
		sui.showError(fmt.Sprintf("Not enough credits! Need %d, have %d", cost, sui.ctx.GetState().HumanPlayer.Credits))
		return
	}

	// Check if player has required resources
	requirements := entities.GetShipResourceRequirements(sui.selectedShip)
	for resourceType, amount := range requirements {
		if !sui.planet.HasStoredResource(resourceType, amount) {
			sui.showError(fmt.Sprintf("Not enough %s! Need %d, have %d",
				resourceType, amount, sui.planet.GetStoredAmount(resourceType)))
			return
		}
	}

	// Deduct credits
	sui.ctx.GetState().HumanPlayer.Credits -= cost

	// Deduct resources
	for resourceType, amount := range requirements {
		sui.planet.RemoveStoredResource(resourceType, amount)
	}

	// Add to construction queue
	// Note: We'll need to add this to the construction system
	sui.addShipToConstructionQueue(sui.selectedShip)

	sui.showError(fmt.Sprintf("%s added to construction queue!", sui.selectedShip))
}

// addShipToConstructionQueue adds a ship to the construction system
func (sui *ShipyardUI) addShipToConstructionQueue(shipType entities.ShipType) {
	if sui.planet == nil || sui.ctx.GetState().HumanPlayer == nil {
		return
	}

	// Get the construction system
	constructionSystem := tickable.GetSystemByName("Construction")
	if constructionSystem == nil {
		fmt.Printf("[ShipyardUI] ERROR: Construction system not found\n")
		return
	}

	cs, ok := constructionSystem.(*tickable.ConstructionSystem)
	if !ok {
		fmt.Printf("[ShipyardUI] ERROR: Failed to cast construction system\n")
		return
	}

	// Create construction item for the ship
	// Use consistent location format: "planet_<ID>"
	location := fmt.Sprintf("planet_%d", sui.planet.GetID())

	item := &tickable.ConstructionItem{
		ID:             fmt.Sprintf("ship_%s_%d", shipType, sui.ctx.GetTickManager().GetCurrentTick()),
		Type:           "Ship",
		Name:           fmt.Sprintf("%s", shipType),
		Location:       location, // Use same format as queue location
		Owner:          sui.ctx.GetState().HumanPlayer.Name,
		Progress:       0,
		TotalTicks:     entities.GetShipBuildTime(shipType),
		RemainingTicks: entities.GetShipBuildTime(shipType),
		Cost:           entities.GetShipBuildCost(shipType),
		Started:        sui.ctx.GetTickManager().GetCurrentTick(),
	}

	// Add to construction queue
	cs.AddToQueue(location, item)

	fmt.Printf("[ShipyardUI] Added %s to construction queue at %s\n", shipType, sui.planet.Name)
}

// showError displays an error message
func (sui *ShipyardUI) showError(message string) {
	sui.errorMessage = message
	sui.errorTimer = 120 // 2 seconds at 60 FPS
}

// Draw renders the shipyard UI
func (sui *ShipyardUI) Draw(screen *ebiten.Image) {
	if !sui.visible || sui.planet == nil {
		return
	}

	// Background panel
	panel := &views.UIPanel{
		X:           sui.x,
		Y:           sui.y,
		Width:       sui.width,
		Height:      sui.height,
		BgColor:     color.RGBA{10, 10, 20, 240},
		BorderColor: color.RGBA{100, 150, 200, 255},
	}
	panel.Draw(screen)

	// Title
	titleText := fmt.Sprintf("Shipyard - %s", sui.planet.Name)
	views.DrawTextCentered(screen, titleText, sui.x+sui.width/2, sui.y+20, color.RGBA{150, 200, 255, 255}, 1.5)

	// Close button
	closeX := sui.x + sui.width - 30
	closeY := sui.y + 10
	views.DrawText(screen, "[X]", closeX, closeY, color.RGBA{255, 100, 100, 255})

	// Player credits
	creditsY := sui.y + 45
	views.DrawText(screen, fmt.Sprintf("Credits: %d", sui.ctx.GetState().HumanPlayer.Credits), sui.x+20, creditsY, utils.TextPrimary)

	// Ship list
	sui.drawShipList(screen)

	// Selected ship details
	if sui.selectedShip != "" {
		sui.drawShipDetails(screen)
	}

	// Build button
	sui.drawBuildButton(screen)

	// Error message
	if sui.errorMessage != "" {
		sui.drawError(screen)
	}
}

// drawShipList draws the list of available ships
func (sui *ShipyardUI) drawShipList(screen *ebiten.Image) {
	startY := sui.y + 80
	itemHeight := 70

	// List background
	listPanel := &views.UIPanel{
		X:           sui.x + 10,
		Y:           sui.y + 70,
		Width:       sui.width - 20,
		Height:      sui.height - 150,
		BgColor:     color.RGBA{5, 5, 10, 200},
		BorderColor: utils.PanelBorder,
	}
	listPanel.Draw(screen)

	visibleCount := 0
	for i, shipType := range sui.shipTypes {
		itemY := startY + i*itemHeight - sui.scrollOffset

		// Skip if off screen
		if itemY < sui.y+60 || itemY > sui.y+sui.height-80 {
			continue
		}
		visibleCount++

		// Item background
		bgColor := color.RGBA{20, 20, 40, 200}
		if shipType == sui.selectedShip {
			bgColor = color.RGBA{40, 60, 100, 220}
		} else if shipType == sui.hoveredShip {
			bgColor = color.RGBA{30, 40, 60, 200}
		}

		itemPanel := &views.UIPanel{
			X:           sui.x + 20,
			Y:           itemY,
			Width:       sui.width - 40,
			Height:      itemHeight - 10,
			BgColor:     bgColor,
			BorderColor: utils.PanelBorder,
		}
		itemPanel.Draw(screen)

		// Ship name
		views.DrawText(screen, string(shipType), sui.x+30, itemY+10, utils.TextPrimary)

		// Cost
		cost := entities.GetShipBuildCost(shipType)
		costColor := utils.TextPrimary
		if sui.ctx.GetState().HumanPlayer.Credits < cost {
			costColor = color.RGBA{255, 100, 100, 255}
		}
		views.DrawText(screen, fmt.Sprintf("Cost: %d credits", cost), sui.x+30, itemY+30, costColor)

		// Build time
		buildTime := entities.GetShipBuildTime(shipType)
		timeStr := fmt.Sprintf("Time: %d ticks (%.1fs)", buildTime, float64(buildTime)/10.0)
		views.DrawText(screen, timeStr, sui.x+30, itemY+50, utils.TextSecondary)
	}

	// Draw scroll indicator if there are more items
	if len(sui.shipTypes) > visibleCount && visibleCount > 0 {
		scrollHintY := sui.y + sui.height - 155
		views.DrawTextCentered(screen, "↓ Scroll for more ↓", sui.x+sui.width/2, scrollHintY, utils.TextSecondary, 0.8)
	}
}

// drawShipDetails draws detailed info about the selected ship
func (sui *ShipyardUI) drawShipDetails(screen *ebiten.Image) {
	detailsY := sui.y + sui.height - 150

	// Resources required
	requirements := entities.GetShipResourceRequirements(sui.selectedShip)
	if len(requirements) > 0 {
		views.DrawText(screen, "Resources Required:", sui.x+20, detailsY, utils.TextPrimary)
		reqY := detailsY + 20

		// Sort resources by name to prevent flickering
		type resourceReq struct {
			name   string
			amount int
		}
		sortedReqs := make([]resourceReq, 0, len(requirements))
		for resourceType, amount := range requirements {
			sortedReqs = append(sortedReqs, resourceReq{resourceType, amount})
		}
		// Sort alphabetically
		for i := 0; i < len(sortedReqs)-1; i++ {
			for j := i + 1; j < len(sortedReqs); j++ {
				if sortedReqs[i].name > sortedReqs[j].name {
					sortedReqs[i], sortedReqs[j] = sortedReqs[j], sortedReqs[i]
				}
			}
		}

		for _, req := range sortedReqs {
			available := sui.planet.GetStoredAmount(req.name)
			reqColor := utils.TextPrimary
			if available < req.amount {
				reqColor = color.RGBA{255, 100, 100, 255}
			}
			reqText := fmt.Sprintf("  %s: %d / %d", req.name, available, req.amount)
			views.DrawText(screen, reqText, sui.x+30, reqY, reqColor)
			reqY += 15
		}
	}
}

// drawBuildButton draws the build button
func (sui *ShipyardUI) drawBuildButton(screen *ebiten.Image) {
	buildButtonY := sui.y + sui.height - 60

	canBuild := sui.selectedShip != "" && sui.canAffordShip(sui.selectedShip)
	buttonColor := color.RGBA{40, 40, 60, 220}
	textColor := utils.TextSecondary

	if canBuild {
		buttonColor = color.RGBA{50, 100, 150, 220}
		textColor = utils.TextPrimary
	}

	buttonPanel := &views.UIPanel{
		X:           sui.x + sui.width/2 - 100,
		Y:           buildButtonY,
		Width:       200,
		Height:      40,
		BgColor:     buttonColor,
		BorderColor: utils.PanelBorder,
	}
	buttonPanel.Draw(screen)

	buttonText := "Build Ship"
	if sui.selectedShip == "" {
		buttonText = "Select a ship"
	} else if !canBuild {
		buttonText = "Cannot afford"
	}

	views.DrawTextCentered(screen, buttonText, sui.x+sui.width/2, buildButtonY+15, textColor, 1.0)
}

// drawError draws an error/notification message
func (sui *ShipyardUI) drawError(screen *ebiten.Image) {
	errorY := sui.y + sui.height - 110
	errorPanel := &views.UIPanel{
		X:           sui.x + 50,
		Y:           errorY,
		Width:       sui.width - 100,
		Height:      30,
		BgColor:     color.RGBA{60, 20, 20, 240},
		BorderColor: color.RGBA{200, 50, 50, 255},
	}
	errorPanel.Draw(screen)

	views.DrawTextCentered(screen, sui.errorMessage, sui.x+sui.width/2, errorY+10, color.RGBA{255, 200, 200, 255}, 0.9)
}

// canAffordShip checks if the player can afford to build a ship
func (sui *ShipyardUI) canAffordShip(shipType entities.ShipType) bool {
	// Check credits
	cost := entities.GetShipBuildCost(shipType)
	if sui.ctx.GetState().HumanPlayer.Credits < cost {
		return false
	}

	// Check resources
	requirements := entities.GetShipResourceRequirements(shipType)
	for resourceType, amount := range requirements {
		if !sui.planet.HasStoredResource(resourceType, amount) {
			return false
		}
	}

	return true
}
