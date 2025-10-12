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
}

// NewGalaxyView creates a new galaxy view
func NewGalaxyView(game *Game) *GalaxyView {
	gv := &GalaxyView{
		game:         game,
		clickHandler: NewClickHandler(),
	}

	// Register all systems as clickable
	for _, system := range game.systems {
		gv.clickHandler.AddClickable(system)
	}

	return gv
}

// Update implements View interface
func (gv *GalaxyView) Update() error {
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
