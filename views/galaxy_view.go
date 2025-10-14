package views

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/utils"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
)

const (
	circleRadius = 8
)

// GalaxyView represents the galaxy map view
type GalaxyView struct {
	ctx           GameContext
	clickHandler  *ClickHandler
	lastClickX    int
	lastClickY    int
	lastClickTime int64
	systemFleets  map[int][]*Fleet // Fleets per system ID
	orbitOffset   float64
}

// NewGalaxyView creates a new galaxy view
func NewGalaxyView(ctx GameContext) *GalaxyView {
	gv := &GalaxyView{
		ctx:          ctx,
		clickHandler: NewClickHandler(),
		systemFleets: make(map[int][]*Fleet),
	}

	// Register all systems as clickable
	for _, system := range ctx.GetSystems() {
		gv.clickHandler.AddClickable(system)
	}

	return gv
}

// Update implements View interface
func (gv *GalaxyView) Update() error {
	kb := gv.ctx.GetKeyBindings()
	vm := gv.ctx.GetViewManager()

	// Update orbit animation
	tm := gv.ctx.GetTickManager()
	if tm != nil && !tm.IsPaused() {
		speedVal := tm.GetSpeedFloat()
		gv.orbitOffset += 0.005 * speedVal
		if gv.orbitOffset > 6.28318 { // 2*PI
			gv.orbitOffset -= 6.28318
		}
	}

	// Update fleet aggregation for each system
	gv.updateFleets()

	// Handle escape to return to main menu
	if kb.IsActionJustPressed(ActionEscape) {
		vm.SwitchTo(ViewTypeMainMenu)
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
					vm.SwitchTo(ViewTypeSystem)
					if systemView, ok := vm.GetView(ViewTypeSystem).(*SystemView); ok {
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
	screen.Fill(utils.Background)

	// Draw hyperlanes first (so they appear behind systems)
	gv.drawHyperlanes(screen)

	// Draw all systems
	for _, system := range gv.ctx.GetSystems() {
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
			utils.Highlight)
	}

	// Draw context menu if active
	if gv.clickHandler.HasActiveMenu() {
		gv.clickHandler.GetActiveMenu().Draw(screen)
	}

	// Draw UI info
	DrawText(screen, "Xandaris II - Galaxy Map", 10, 10, utils.TextPrimary)
	DrawText(screen, "Double-click system to view", 10, 25, utils.TextSecondary)
	DrawText(screen, "Press ESC to quit", 10, 40, utils.TextSecondary)

	// Draw player info
	gv.drawPlayerInfo(screen)
}

// OnEnter implements View interface
func (gv *GalaxyView) OnEnter() {
	// Re-register clickables when entering view
	gv.clickHandler.ClearClickables()
	for _, system := range gv.ctx.GetSystems() {
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
	hyperlaneColor := utils.HyperlaneNormal

	for _, hyperlane := range gv.ctx.GetHyperlanes() {
		systems := gv.ctx.GetSystems()
		fromSystem := systems[hyperlane.From]
		toSystem := systems[hyperlane.To]

		// Draw line between systems
		DrawLine(screen,
			int(fromSystem.X), int(fromSystem.Y),
			int(toSystem.X), int(toSystem.Y),
			hyperlaneColor)
	}
}

// drawSystem renders a single system
func (gv *GalaxyView) drawSystem(screen *ebiten.Image, system *entities.System) {
	humanPlayer := gv.ctx.GetHumanPlayer()
	centerX := int(system.X)
	centerY := int(system.Y)

	// Draw ownership indicator if system has player-owned planets
	var ownershipColor color.RGBA
	if humanPlayer != nil && system.HasOwnershipByPlayer(humanPlayer.Name) {
		ownershipColor = humanPlayer.Color
		// DrawOwnershipRing(screen, centerX, centerY, float64(circleRadius+4), humanPlayer.Color)
	}

	// Get the star and planets from the system
	star := system.GetEntitiesByType(entities.EntityTypeStar)[0].(*entities.Star)
	planets := system.GetEntitiesByType(entities.EntityTypePlanet)

	// Draw a small circle for the star
	starRadius := 4
	starImg := ebiten.NewImage(starRadius*2, starRadius*2)
	for py := 0; py < starRadius*2; py++ {
		for px := 0; px < starRadius*2; px++ {
			dx := float64(px - starRadius)
			dy := float64(py - starRadius)
			dist := dx*dx + dy*dy
			if dist <= float64(starRadius*starRadius) {
				starImg.Set(px, py, star.Color)
			}
		}
	}
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(centerX-starRadius), float64(centerY-starRadius))
	screen.DrawImage(starImg, opts)

	// Draw the correct number of orbits
	orbitColor := color.RGBA{R: 100, G: 100, B: 100, A: 100}
	if ownershipColor.A != 0 {
		orbitColor = ownershipColor
	}

	for i, planet := range planets {
		orbitRadius := float64(starRadius + (i+1)*4)
		segments := 20
		for j := 0; j < segments; j++ {
			angle1 := float64(j) * 2 * math.Pi / float64(segments)
			angle2 := float64(j+1) * 2 * math.Pi / float64(segments)

			x1 := centerX + int(orbitRadius*math.Cos(angle1))
			y1 := centerY + int(orbitRadius*math.Sin(angle1))
			x2 := centerX + int(orbitRadius*math.Cos(angle2))
			y2 := centerY + int(orbitRadius*math.Sin(angle2))

			DrawLine(screen, x1, y1, x2, y2, orbitColor)
		}

		// Draw the planet
		planetRadius := 1
		planetAngle := planet.GetOrbitAngle() + gv.orbitOffset // Animate planet orbit
		planetX := centerX + int(orbitRadius*math.Cos(planetAngle))
		planetY := centerY + int(orbitRadius*math.Sin(planetAngle))
		planetImg := ebiten.NewImage(planetRadius*2, planetRadius*2)
		for py := 0; py < planetRadius*2; py++ {
			for px := 0; px < planetRadius*2; px++ {
				dx := float64(px - planetRadius)
				dy := float64(py - planetRadius)
				dist := dx*dx + dy*dy
				if dist <= float64(planetRadius*planetRadius) {
					planetImg.Set(px, py, planet.GetColor())
				}
			}
		}
		planetOpts := &ebiten.DrawImageOptions{}
		planetOpts.GeoM.Translate(float64(planetX-planetRadius), float64(planetY-planetRadius))
		screen.DrawImage(planetImg, planetOpts)
	}

	// Draw centered label below the circle
	labelY := centerY + circleRadius + 15
	DrawCenteredText(screen, system.Name, centerX, labelY)
}

// updateFleets aggregates fleets for each system
func (gv *GalaxyView) updateFleets() {
	fm := gv.ctx.GetFleetManager()
	gv.systemFleets = make(map[int][]*Fleet)

	for _, system := range gv.ctx.GetSystems() {
		// Get fleets orbiting the star in this system (not planets)
		fleets := fm.AggregateFleets(system)
		if len(fleets) > 0 {
			gv.systemFleets[system.ID] = fleets
		}
	}
}

// drawFleets draws fleet indicators at system locations
func (gv *GalaxyView) drawFleets(screen *ebiten.Image) {
	humanPlayer := gv.ctx.GetHumanPlayer()

	for systemID, fleets := range gv.systemFleets {
		// Find the system
		var system *entities.System
		for _, sys := range gv.ctx.GetSystems() {
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
		ownerColor := utils.TextPrimary
		for _, fleet := range fleets {
			totalShips += fleet.Size()
			// Use player color if owned by human player
			if humanPlayer != nil && fleet.Owner == humanPlayer.Name {
				ownerColor = humanPlayer.Color
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
	humanPlayer := gv.ctx.GetHumanPlayer()

	// Find all moving ships across all systems
	for _, system := range gv.ctx.GetSystems() {
		for _, entity := range system.Entities {
			if ship, ok := entity.(*entities.Ship); ok {
				if ship.Status == entities.ShipStatusMoving && ship.TargetSystem != -1 {
					gv.drawTransitShip(screen, ship, humanPlayer)
				}
			}
		}
	}
}

// drawTransitShip draws a single ship in transit
func (gv *GalaxyView) drawTransitShip(screen *ebiten.Image, ship *entities.Ship, humanPlayer *entities.Player) {
	// Find source and target systems
	var sourceSystem, targetSystem *entities.System
	for _, sys := range gv.ctx.GetSystems() {
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
	if humanPlayer != nil && ship.Owner == humanPlayer.Name {
		shipColor = humanPlayer.Color
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
	pulseSize := 2
	for py := 0; py < pulseSize*2; py++ {
		for px := 0; px < pulseSize*2; px++ {
			dx := float64(px - pulseSize)
			dy := float64(py - pulseSize)
			if dx*dx+dy*dy <= float64(pulseSize*pulseSize) {
				screen.Set(shipX+px-pulseSize, shipY+py-pulseSize+8, utils.SystemBlue)
			}
		}
	}
}

// drawPlayerInfo draws player information panel
func (gv *GalaxyView) drawPlayerInfo(screen *ebiten.Image) {
	humanPlayer := gv.ctx.GetHumanPlayer()
	if humanPlayer == nil {
		return
	}

	// Draw panel in top-right corner
	panelX := ScreenWidth - 250
	panelY := 10
	panelWidth := 240
	panelHeight := 100

	// Draw panel background
	panel := NewUIPanel(panelX, panelY, panelWidth, panelHeight)
	panel.Draw(screen)

	// Draw player info
	textX := panelX + 10
	textY := panelY + 15

	DrawText(screen, humanPlayer.Name, textX, textY, humanPlayer.Color)
	DrawText(screen, fmt.Sprintf("Credits: %d", humanPlayer.Credits), textX, textY+15, utils.TextPrimary)
	DrawText(screen, fmt.Sprintf("Planets: %d", len(humanPlayer.OwnedPlanets)), textX, textY+30, utils.TextPrimary)
	DrawText(screen, fmt.Sprintf("Population: %d", humanPlayer.GetTotalPopulation()), textX, textY+45, utils.TextPrimary)

	if humanPlayer.HomeSystem != nil {
		DrawText(screen, fmt.Sprintf("Home: %s", humanPlayer.HomeSystem.Name), textX, textY+60, utils.TextSecondary)
	}
}
