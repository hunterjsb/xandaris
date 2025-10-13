package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
)

// GalaxyView represents the galaxy map view
type GalaxyView struct {
	game          *Game
	clickHandler  *ClickHandler
	lastClickX    int
	lastClickY    int
	lastClickTime int64
	fleetManager  *FleetManager
	systemFleets  map[int][]*Fleet // Fleets per system ID
}

// NewGalaxyView creates a new galaxy view
func NewGalaxyView(game *Game) *GalaxyView {
	gv := &GalaxyView{
		game:         game,
		clickHandler: NewClickHandler(),
		fleetManager: NewFleetManager(game),
		systemFleets: make(map[int][]*Fleet),
	}

	// Register all systems as clickable
	for _, system := range game.systems {
		gv.clickHandler.AddClickable(system)
	}

	return gv
}

// Update implements View interface
func (gv *GalaxyView) Update() error {
	// Update fleet aggregation for each system
	gv.updateFleets()

	// Handle escape to return to main menu
	if gv.game.keyBindings.IsActionJustPressed(ActionEscape) {
		gv.game.viewManager.SwitchTo(ViewTypeMainMenu)
		return nil
	}

	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		// Check for double-click with more forgiving tolerance
		currentTime := ebiten.Tick()
		dx := x - gv.lastClickX
		dy := y - gv.lastClickY
		distance := dx*dx + dy*dy // squared distance to avoid sqrt

		// More forgiving double-click: 60 ticks (~1 second) and within 10 pixels
		if distance <= 100 && currentTime-gv.lastClickTime < 60 {
			// Double click detected - check if we clicked on a system
			if selectedObj := gv.clickHandler.GetSelectedObject(); selectedObj != nil {
				if system, ok := selectedObj.(*entities.System); ok {
					// Switch to system view
					gv.game.viewManager.SwitchTo(ViewTypeSystem)
					if systemView, ok := gv.game.viewManager.GetCurrentView().(*SystemView); ok {
						systemView.SetSystem(system)
					}
				}
			}
		}

		gv.lastClickX = x
		gv.lastClickY = y
		gv.lastClickTime = currentTime

		gv.clickHandler.HandleClick(x, y)
	}

	return nil
}

// Draw implements View interface
func (gv *GalaxyView) Draw(screen *ebiten.Image) {
	// Fill background
	screen.Fill(UIBackground)

	// Draw hyperlanes first (so they appear behind systems)
	gv.drawHyperlanes(screen)

	// Draw all systems
	for _, system := range gv.game.systems {
		gv.drawSystem(screen, system)
	}

	// Draw fleets at their system locations
	gv.drawFleets(screen)

	// Highlight selected object
	if selectedObj := gv.clickHandler.GetSelectedObject(); selectedObj != nil {
		x, y := selectedObj.GetPosition()
		DrawHighlightCircle(screen,
			int(x), int(y),
			int(selectedObj.GetClickRadius()),
			UIHighlight)
	}

	// Draw context menu if active
	if gv.clickHandler.HasActiveMenu() {
		gv.clickHandler.GetActiveMenu().Draw(screen)
	}

	// Draw UI info
	DrawText(screen, "Xandaris II - Galaxy Map", 10, 10, UITextPrimary)
	DrawText(screen, "Double-click system to view", 10, 25, UITextSecondary)
	DrawText(screen, "Press ESC to quit", 10, 40, UITextSecondary)

	// Draw player info
	gv.drawPlayerInfo(screen)
}

// OnEnter implements View interface
func (gv *GalaxyView) OnEnter() {
	// Re-register clickables when entering view
	gv.clickHandler.ClearClickables()
	for _, system := range gv.game.systems {
		gv.clickHandler.AddClickable(system)
	}
}

// OnExit implements View interface
func (gv *GalaxyView) OnExit() {
	// Clear selections when leaving view
	gv.clickHandler.ClearClickables()
}

// GetType implements View interface
func (gv *GalaxyView) GetType() ViewType {
	return ViewTypeGalaxy
}

// drawHyperlanes draws connections between systems
func (gv *GalaxyView) drawHyperlanes(screen *ebiten.Image) {
	hyperlaneColor := HyperlaneNormal

	for _, hyperlane := range gv.game.hyperlanes {
		fromSystem := gv.game.systems[hyperlane.From]
		toSystem := gv.game.systems[hyperlane.To]

		// Draw line between systems
		DrawLine(screen,
			int(fromSystem.X), int(fromSystem.Y),
			int(toSystem.X), int(toSystem.Y),
			hyperlaneColor)
	}
}

// drawSystem renders a single system
func (gv *GalaxyView) drawSystem(screen *ebiten.Image, system *entities.System) {
	centerX := int(system.X)
	centerY := int(system.Y)

	// Draw ownership indicator if system has player-owned planets
	if gv.game.humanPlayer != nil && system.HasOwnershipByPlayer(gv.game.humanPlayer.Name) {
		DrawOwnershipRing(screen, centerX, centerY, float64(circleRadius+4), gv.game.humanPlayer.Color)
	}

	// Create a circular image for the system
	circleImg := ebiten.NewImage(circleRadius*2, circleRadius*2)

	// Draw a circle by filling pixels within the radius
	for py := 0; py < circleRadius*2; py++ {
		for px := 0; px < circleRadius*2; px++ {
			// Calculate distance from center
			dx := float64(px - circleRadius)
			dy := float64(py - circleRadius)
			dist := dx*dx + dy*dy

			// If within radius, set pixel to system color
			if dist <= float64(circleRadius*circleRadius) {
				circleImg.Set(px, py, system.Color)
			}
		}
	}

	// Draw the circle centered
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(centerX-circleRadius), float64(centerY-circleRadius))
	screen.DrawImage(circleImg, opts)

	// Draw centered label below the circle
	labelY := centerY + circleRadius + 15
	DrawCenteredText(screen, system.Name, centerX, labelY)
}

// updateFleets aggregates fleets for each system
func (gv *GalaxyView) updateFleets() {
	gv.systemFleets = make(map[int][]*Fleet)

	for _, system := range gv.game.systems {
		// Get fleets orbiting the star in this system (not planets)
		fleets := gv.fleetManager.AggregateFleets(system)
		if len(fleets) > 0 {
			gv.systemFleets[system.ID] = fleets
		}
	}
}

// drawFleets draws fleet indicators at system locations
func (gv *GalaxyView) drawFleets(screen *ebiten.Image) {
	for systemID, fleets := range gv.systemFleets {
		// Find the system
		var system *entities.System
		for _, sys := range gv.game.systems {
			if sys.ID == systemID {
				system = sys
				break
			}
		}
		if system == nil {
			continue
		}

		// Draw fleet indicator near the system
		centerX := int(system.X)
		centerY := int(system.Y)

		// Position fleet indicator above the system
		fleetX := centerX
		fleetY := centerY - circleRadius - 15

		// Count total ships across all fleets
		totalShips := 0
		var ownerColor = UITextPrimary
		for _, fleet := range fleets {
			totalShips += fleet.Size()
			// Use player color if owned by human player
			if gv.game.humanPlayer != nil && fleet.Owner == gv.game.humanPlayer.Name {
				ownerColor = gv.game.humanPlayer.Color
			}
		}

		// Draw ship icon (small triangle)
		size := 4
		for py := 0; py < size*2; py++ {
			for px := 0; px < size*2; px++ {
				dx := float64(px - size)
				dy := float64(py - size)
				if dy > 0 && dx >= -dy/2 && dx <= dy/2 {
					screen.Set(fleetX+px-size, fleetY+py-size, ownerColor)
				}
			}
		}

		// Draw ship count
		if totalShips > 1 {
			countText := fmt.Sprintf("%d", totalShips)
			DrawText(screen, countText, fleetX+6, fleetY-4, ownerColor)
		}
	}

	// Draw ships in transit along hyperlanes
	gv.drawTransitShips(screen)
}

// drawTransitShips draws ships that are in transit between systems
func (gv *GalaxyView) drawTransitShips(screen *ebiten.Image) {
	// Find all moving ships across all systems
	for _, system := range gv.game.systems {
		for _, entity := range system.Entities {
			if ship, ok := entity.(*entities.Ship); ok {
				if ship.Status == entities.ShipStatusMoving && ship.TargetSystem != -1 {
					gv.drawTransitShip(screen, ship)
				}
			}
		}
	}
}

// drawTransitShip draws a single ship in transit
func (gv *GalaxyView) drawTransitShip(screen *ebiten.Image, ship *entities.Ship) {
	// Find source and target systems
	var sourceSystem, targetSystem *entities.System
	for _, sys := range gv.game.systems {
		if sys.ID == ship.CurrentSystem {
			sourceSystem = sys
		}
		if sys.ID == ship.TargetSystem {
			targetSystem = sys
		}
	}

	if sourceSystem == nil || targetSystem == nil {
		return
	}

	// Calculate position along the hyperlane based on travel progress
	progress := ship.TravelProgress
	x := sourceSystem.X + (targetSystem.X-sourceSystem.X)*progress
	y := sourceSystem.Y + (targetSystem.Y-sourceSystem.Y)*progress

	// Determine color
	shipColor := ship.Color
	if gv.game.humanPlayer != nil && ship.Owner == gv.game.humanPlayer.Name {
		shipColor = gv.game.humanPlayer.Color
	}

	// Draw ship as a small triangle
	size := 4
	shipX := int(x)
	shipY := int(y)

	for py := 0; py < size*2; py++ {
		for px := 0; px < size*2; px++ {
			dx := float64(px - size)
			dy := float64(py - size)
			if dy > 0 && dx >= -dy/2 && dx <= dy/2 {
				screen.Set(shipX+px-size, shipY+py-size, shipColor)
			}
		}
	}

	// Draw a subtle travel indicator (pulsing dot)
	// Use tick to create pulsing effect
	pulseSize := 2
	for py := 0; py < pulseSize*2; py++ {
		for px := 0; px < pulseSize*2; px++ {
			dx := float64(px - pulseSize)
			dy := float64(py - pulseSize)
			if dx*dx+dy*dy <= float64(pulseSize*pulseSize) {
				screen.Set(shipX+px-pulseSize, shipY+py-pulseSize+8, SystemBlue)
			}
		}
	}
}

// drawPlayerInfo draws player information panel
func (gv *GalaxyView) drawPlayerInfo(screen *ebiten.Image) {
	if gv.game.humanPlayer == nil {
		return
	}

	player := gv.game.humanPlayer

	// Draw panel in top-right corner
	panelX := screenWidth - 250
	panelY := 10
	panelWidth := 240
	panelHeight := 100

	// Draw panel background
	panel := NewUIPanel(panelX, panelY, panelWidth, panelHeight)
	panel.Draw(screen)

	// Draw player info
	textX := panelX + 10
	textY := panelY + 15

	DrawText(screen, player.Name, textX, textY, player.Color)
	DrawText(screen, fmt.Sprintf("Credits: %d", player.Credits), textX, textY+15, UITextPrimary)
	DrawText(screen, fmt.Sprintf("Planets: %d", len(player.OwnedPlanets)), textX, textY+30, UITextPrimary)
	DrawText(screen, fmt.Sprintf("Population: %d", player.GetTotalPopulation()), textX, textY+45, UITextPrimary)

	if player.HomeSystem != nil {
		DrawText(screen, fmt.Sprintf("Home: %s", player.HomeSystem.Name), textX, textY+60, UITextSecondary)
	}
}
