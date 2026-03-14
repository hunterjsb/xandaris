package views

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/utils"
)

var (
	galaxyCircleCache = utils.NewCircleImageCache()
)

const (
	circleRadius = 8
)

// GalaxyView represents the galaxy map view
type GalaxyView struct {
	ctx                     GameContext
	clickHandler            *ClickHandler
	lastClickX              int
	lastClickY              int
	lastClickTime           int64
	systemFleets            map[int][]*entities.Fleet // Fleets per system ID
	orbitOffset             float64
	playerPanelRect         image.Rectangle
	playerDirectoryHintRect image.Rectangle
	playerPanelToggleRect   image.Rectangle
	playerPanelCollapsed    bool
}

// NewGalaxyView creates a new galaxy view
func NewGalaxyView(ctx GameContext) *GalaxyView {
	gv := &GalaxyView{
		ctx:                     ctx,
		clickHandler:            NewClickHandler("galaxy"),
		systemFleets:            make(map[int][]*entities.Fleet),
		playerPanelRect:         image.Rectangle{},
		playerDirectoryHintRect: image.Rectangle{},
		playerPanelToggleRect:   image.Rectangle{},
		playerPanelCollapsed:    false,
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

	// Quick-select home system
	if kb.IsActionJustPressed(ActionFocusHome) {
		gv.focusHomeSystem()
	}

	// Handle escape to return to main menu
	if kb.IsActionJustPressed(ActionEscape) {
		vm.SwitchTo(ViewTypeMainMenu)
		return nil
	}

	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		if gv.playerPanelToggleRect.Dx() > 0 && gv.playerPanelToggleRect.Dy() > 0 {
			if x >= gv.playerPanelToggleRect.Min.X && x <= gv.playerPanelToggleRect.Max.X &&
				y >= gv.playerPanelToggleRect.Min.Y && y <= gv.playerPanelToggleRect.Max.Y {
				gv.playerPanelCollapsed = !gv.playerPanelCollapsed
				return nil
			}
		}

		if gv.playerDirectoryHintRect.Dx() > 0 && gv.playerDirectoryHintRect.Dy() > 0 {
			if x >= gv.playerDirectoryHintRect.Min.X && x <= gv.playerDirectoryHintRect.Max.X &&
				y >= gv.playerDirectoryHintRect.Min.Y && y <= gv.playerDirectoryHintRect.Max.Y {
				if directory, ok := vm.GetView(ViewTypePlayers).(*PlayerDirectoryView); ok {
					directory.SetReturnView(ViewTypeGalaxy)
				}
				vm.SwitchTo(ViewTypePlayers)
				return nil
			}
		}

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
			int(selectedObj.GetClickRadius("galaxy")),
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

	// Draw hints below header
	gv.drawHints(screen)

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
	centerX := int(system.X)
	centerY := int(system.Y)

	// Get the star and planets from the system
	star := system.GetEntitiesByType(entities.EntityTypeStar)[0].(*entities.Star)
	planets := system.GetEntitiesByType(entities.EntityTypePlanet)

	// Draw a small circle for the star
	starRadius := 4
	starImg := galaxyCircleCache.GetOrCreate(starRadius, star.Color)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(centerX-starRadius), float64(centerY-starRadius))
	screen.DrawImage(starImg, opts)

	// Draw the correct number of orbits
	for i, planet := range planets {
		orbitColor := color.RGBA{R: 100, G: 100, B: 100, A: 100}
		if p, ok := planet.(*entities.Planet); ok {
			if ownerColor, ok := gv.getOwnerColor(p.Owner); ok {
				orbitColor = ownerColor
			}
		}

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
		planetImg := galaxyCircleCache.GetOrCreate(planetRadius, planet.GetColor())
		planetOpts := &ebiten.DrawImageOptions{}
		planetOpts.GeoM.Translate(float64(planetX-planetRadius), float64(planetY-planetRadius))
		screen.DrawImage(planetImg, planetOpts)
	}

	// Draw centered label below the circle
	labelY := centerY + circleRadius + 15
	DrawCenteredText(screen, system.Name, centerX, labelY)

	// Show compact owner + stock info for inhabited systems
	for _, entity := range system.Entities {
		if p, ok := entity.(*entities.Planet); ok && p.Owner != "" {
			totalStock := 0
			for _, s := range p.StoredResources {
				if s != nil {
					totalStock += s.Amount
				}
			}
			shortOwner := p.Owner
			if len(shortOwner) > 8 {
				shortOwner = shortOwner[:8]
			}
			info := fmt.Sprintf("%s %d", shortOwner, totalStock)
			infoColor := utils.TextSecondary
			if ownerColor, ok := gv.getOwnerColor(p.Owner); ok {
				infoColor = ownerColor
				infoColor.A = 180
			}
			infoWidth := len(info) * 6
			DrawText(screen, info, centerX-infoWidth/2, labelY+12, infoColor)
			break // only show first owned planet
		}
	}
}

func (gv *GalaxyView) getOwnerColor(owner string) (color.RGBA, bool) {
	if owner == "" {
		return color.RGBA{}, false
	}

	for _, player := range gv.ctx.GetPlayers() {
		if player != nil && player.Name == owner {
			return player.Color, true
		}
	}

	return color.RGBA{}, false
}

// FocusSystem highlights the given system in the galaxy view
func (gv *GalaxyView) FocusSystem(system *entities.System) {
	if system == nil {
		return
	}
	gv.clickHandler.Select(system)
}

func (gv *GalaxyView) focusHomeSystem() {
	player := gv.ctx.GetHumanPlayer()
	if player == nil || player.HomeSystem == nil {
		return
	}

	gv.FocusSystem(player.HomeSystem)
}

// updateFleets collects fleets for each system
func (gv *GalaxyView) updateFleets() {
	gv.systemFleets = make(map[int][]*entities.Fleet)

	for _, system := range gv.ctx.GetSystems() {
		// Get actual fleet entities in this system
		var fleets []*entities.Fleet
		for _, entity := range system.Entities {
			if fleet, ok := entity.(*entities.Fleet); ok {
				fleets = append(fleets, fleet)
			}
		}
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
			if humanPlayer != nil && fleet.GetOwner() == humanPlayer.Name {
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

	gv.playerPanelRect = image.Rectangle{}
	gv.playerDirectoryHintRect = image.Rectangle{}
	gv.playerPanelToggleRect = image.Rectangle{}

	players := gv.ctx.GetPlayers()
	aiSummaries := make([]struct {
		name         string
		color        color.RGBA
		planets      int
		tradingPosts int
		totalStorage int
	}, 0)

	for _, player := range players {
		if player == nil || player == humanPlayer || !player.IsAI() {
			continue
		}

		tradingPosts := countTradingPosts(player.OwnedPlanets)
		if tradingPosts == 0 {
			continue
		}

		aiSummaries = append(aiSummaries, struct {
			name         string
			color        color.RGBA
			planets      int
			tradingPosts int
			totalStorage int
		}{
			name:         player.Name,
			color:        player.Color,
			planets:      len(player.OwnedPlanets),
			tradingPosts: tradingPosts,
			totalStorage: totalStoredResources(player.OwnedPlanets),
		})
	}

	baseHeight := 170
	extraHeight := len(aiSummaries) * 26
	panelHeight := baseHeight + extraHeight
	if gv.playerPanelCollapsed {
		panelHeight = 74
	}
	if panelHeight > ScreenHeight-20 {
		panelHeight = ScreenHeight - 20
	}
	if panelHeight < baseHeight && !gv.playerPanelCollapsed {
		panelHeight = baseHeight
	}

	// Draw panel in top-right corner
	panelWidth := 290
	panelX := ScreenWidth - panelWidth - 10
	panelY := 10

	gv.playerPanelRect = image.Rect(panelX, panelY, panelX+panelWidth, panelY+panelHeight)

	// Draw panel background
	panel := NewUIPanel(panelX, panelY, panelWidth, panelHeight)
	panel.Draw(screen)

	// Draw player info
	textX := panelX + 12
	textY := panelY + 16

	toggleLabel := "−"
	if gv.playerPanelCollapsed {
		toggleLabel = "+"
	}
	gv.playerPanelToggleRect = image.Rect(panelX+panelWidth-28, panelY+14, panelX+panelWidth-8, panelY+34)
	DrawText(screen, fmt.Sprintf("[%s]", toggleLabel), panelX+panelWidth-28, panelY+18, utils.Highlight)

	DrawText(screen, humanPlayer.Name, textX, textY, humanPlayer.Color)
	DrawText(screen, fmt.Sprintf("Credits: %d", humanPlayer.Credits), textX, textY+18, utils.TextPrimary)

	if gv.playerPanelCollapsed {
		DrawText(screen, fmt.Sprintf("Planets: %d", len(humanPlayer.OwnedPlanets)), textX, textY+36, utils.TextPrimary)
		collapsedHint := "Click [+] to expand  |  Press [P] for Directory"
		hintY := panelY + panelHeight - 18
		DrawText(screen, collapsedHint, textX, hintY, utils.TextSecondary)
		hintWidth := len(collapsedHint) * 6
		gv.playerDirectoryHintRect = image.Rect(textX-2, hintY-12, textX+hintWidth+2, hintY+4)
		return
	}

	DrawText(screen, fmt.Sprintf("Planets: %d", len(humanPlayer.OwnedPlanets)), textX, textY+36, utils.TextPrimary)
	DrawText(screen, fmt.Sprintf("Population: %d", humanPlayer.GetTotalPopulation()), textX, textY+54, utils.TextPrimary)

	// Ships + construction count on same line
	shipCount := len(humanPlayer.OwnedShips) + len(humanPlayer.OwnedFleets)
	shipLine := fmt.Sprintf("Ships: %d", shipCount)
	if cs := tickable.GetSystemByName("Construction"); cs != nil {
		if csys, ok := cs.(*tickable.ConstructionSystem); ok {
			queueItems := csys.GetConstructionsByOwner(humanPlayer.Name)
			if len(queueItems) > 0 {
				shipLine += fmt.Sprintf("  Building: %d", len(queueItems))
			}
		}
	}
	DrawText(screen, shipLine, textX, textY+72, utils.TextPrimary)

	if humanPlayer.HomeSystem != nil {
		DrawText(screen, fmt.Sprintf("Home: %s", humanPlayer.HomeSystem.Name), textX, textY+90, utils.TextSecondary)
	}

	// Separator + footer hint
	separatorY := textY + 106
	DrawLine(screen, panelX+8, separatorY, panelX+panelWidth-8, separatorY, utils.PanelBorder)

	// Footer: Player Directory hint
	footerY := panelY + panelHeight - 18
	DrawText(screen, "Press [P] for Player Directory", textX, footerY, utils.TextSecondary)
	footerWidth := len("Press [P] for Player Directory") * 6
	gv.playerDirectoryHintRect = image.Rect(textX-2, footerY-12, textX+footerWidth+2, footerY+4)

	if len(aiSummaries) > 0 {
		listY := separatorY + 10
		DrawText(screen, "NPC Traders:", textX, listY, utils.TextSecondary)
		listY += 15
		maxListY := footerY - 8

		for _, summary := range aiSummaries {
			if listY+18 > maxListY {
				DrawText(screen, fmt.Sprintf("+%d more", len(aiSummaries)-len(aiSummaries)), textX, maxListY-4, utils.TextSecondary)
				break
			}
			DrawText(screen, summary.name, textX, listY, summary.color)
			info := fmt.Sprintf("%dp %dTP %d stock", summary.planets, summary.tradingPosts, summary.totalStorage)
			DrawText(screen, info, textX+4, listY+12, utils.TextSecondary)
			listY += 26
		}
	}

}

// drawHints renders actionable suggestions in the bottom-center of the screen.
func (gv *GalaxyView) drawHints(screen *ebiten.Image) {
	humanPlayer := gv.ctx.GetHumanPlayer()
	if humanPlayer == nil {
		return
	}

	var hints []string

	// Check mines
	totalMines := 0
	hasShipyard := false
	hasRefinery := false
	hasFuel := false
	hasOil := false
	for _, planet := range humanPlayer.OwnedPlanets {
		if planet == nil {
			continue
		}
		if planet.GetStoredAmount("Fuel") > 0 {
			hasFuel = true
		}
		if planet.GetStoredAmount("Oil") > 20 {
			hasOil = true
		}
		for _, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok {
				if b.BuildingType == "Mine" {
					totalMines++
				}
				if b.BuildingType == "Shipyard" {
					hasShipyard = true
				}
				if b.BuildingType == "Refinery" {
					hasRefinery = true
				}
			}
		}
	}

	if totalMines == 0 {
		// Check if under construction
		constructing := false
		if cs := tickable.GetSystemByName("Construction"); cs != nil {
			if csys, ok := cs.(*tickable.ConstructionSystem); ok {
				for _, item := range csys.GetConstructionsByOwner(humanPlayer.Name) {
					if item.Name == "Mine" {
						constructing = true
						break
					}
				}
			}
		}
		if constructing {
			hints = append(hints, "Mines under construction — production will begin soon")
		} else {
			hints = append(hints, "Build mines on resource deposits to start producing")
		}
	}

	// Market-driven suggestions using price ratios
	market := gv.ctx.GetMarket()
	if market != nil && totalMines > 0 {
		snap := market.GetSnapshot()
		for name, rm := range snap.Resources {
			if rm.BasePrice > 0 && rm.CurrentPrice/rm.BasePrice > 2.0 {
				hints = append(hints, fmt.Sprintf("%s price at %.0fx base — build more mines", name, rm.CurrentPrice/rm.BasePrice))
				break // only show one price hint
			}
		}
	}

	if !hasShipyard && humanPlayer.Credits > 2000 {
		hints = append(hints, "Build a Shipyard to construct ships")
	}
	if !hasRefinery && hasOil && !hasFuel {
		hints = append(hints, "Build a Refinery to convert Oil into Fuel")
	}
	if humanPlayer.Credits > 50000 {
		hints = append(hints, "Excess credits — invest in upgrades or buildings")
	}

	// Resource depletion warning
	for _, planet := range humanPlayer.OwnedPlanets {
		if planet == nil {
			continue
		}
		for _, resEntity := range planet.Resources {
			if res, ok := resEntity.(*entities.Resource); ok && res.Abundance > 0 && res.Abundance < 15 {
				hints = append(hints, fmt.Sprintf("%s deposit on %s nearly depleted (a:%d)", res.ResourceType, planet.Name, res.Abundance))
				break
			}
		}
	}

	// Low fuel ship
	for _, ship := range humanPlayer.OwnedShips {
		if ship != nil && ship.CurrentFuel < ship.MaxFuel/4 {
			hints = append(hints, fmt.Sprintf("%s is low on fuel", ship.Name))
			break
		}
	}

	if len(hints) == 0 {
		return
	}

	// Show up to 2 hints in the bottom-center
	screenH := screen.Bounds().Dy()
	hintY := screenH - 80
	maxHints := 2
	if len(hints) < maxHints {
		maxHints = len(hints)
	}

	for i := 0; i < maxHints; i++ {
		text := hints[i]
		textW := len(text) * 6
		x := (screen.Bounds().Dx() - textW) / 2
		DrawText(screen, text, x, hintY, utils.SystemYellow)
		hintY += 14
	}
}

func countTradingPosts(planets []*entities.Planet) int {
	count := 0
	for _, planet := range planets {
		for _, buildingEntity := range planet.Buildings {
			if building, ok := buildingEntity.(*entities.Building); ok && building.BuildingType == "Trading Post" {
				count++
			}
		}
	}
	return count
}

func totalStoredResources(planets []*entities.Planet) int {
	total := 0
	for _, planet := range planets {
		for _, storage := range planet.StoredResources {
			if storage != nil {
				total += storage.Amount
			}
		}
	}
	return total
}
