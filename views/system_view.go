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

// SystemView represents the detailed view of a single system
type SystemView struct {
	ctx           GameContext
	system        *entities.System
	clickHandler  *ClickHandler
	centerX       float64
	centerY       float64
	scale         *ViewScale
	lastClickX    int
	lastClickY    int
	lastClickTime int64
	orbitOffset   float64 // For animating orbits
	fleets        []*Fleet
	fleetInfoUI   FleetInfoUIInterface
}

// NewSystemView creates a new system view
func NewSystemView(ctx GameContext, fleetInfoUI FleetInfoUIInterface) *SystemView {
	return &SystemView{
		ctx:          ctx,
		clickHandler: NewClickHandler(),
		centerX:      float64(ScreenWidth) / 2,
		centerY:      float64(ScreenHeight) / 2,
		scale:        &SystemScale,
		fleetInfoUI:  fleetInfoUI,
	}
}

// SetSystem sets the system to display
func (sv *SystemView) SetSystem(system *entities.System) {
	sv.system = system

	// Calculate auto-scaling based on system size
	maxDistance := GetSystemMaxOrbitDistance(system)
	sv.scale = AutoScale(maxDistance, ScreenWidth, ScreenHeight)

	sv.updateEntityPositions()
	sv.updateFleets()
	sv.registerClickables()
}

// Update implements View interface
func (sv *SystemView) Update() error {
	if sv.system == nil {
		return nil
	}

	kb := sv.ctx.GetKeyBindings()
	vm := sv.ctx.GetViewManager()
	tm := sv.ctx.GetTickManager()

	// Update orbit animation
	if !tm.IsPaused() {
		speedVal := tm.GetSpeedFloat()
		sv.orbitOffset += 0.0005 * speedVal
		if sv.orbitOffset > 6.28318 { // 2*PI
			sv.orbitOffset -= 6.28318
		}
	}

	// Update entity positions for animation
	sv.updateEntityPositions()

	// Update fleet aggregation
	sv.updateFleets()

	// Update fleet info UI if it exists
	if sv.fleetInfoUI != nil && sv.fleetInfoUI.IsVisible() {
		sv.fleetInfoUI.Update()
	}

	// Escape handling - close fleet info UI first, then return to galaxy view
	if kb.IsActionJustPressed(ActionEscape) {
		if sv.fleetInfoUI != nil && sv.fleetInfoUI.IsVisible() {
			sv.fleetInfoUI.Hide()
			return nil
		}
		vm.SwitchTo(ViewTypeGalaxy)
		return nil
	}

	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		// Check for double-click with more forgiving tolerance
		currentTime := ebiten.Tick()
		dx := x - sv.lastClickX
		dy := y - sv.lastClickY
		distance := dx*dx + dy*dy // squared distance to avoid sqrt

		// More forgiving double-click: 60 ticks (~1 second) and within 10 pixels
		if distance <= 100 && currentTime-sv.lastClickTime < 60 {
			// Double click detected - check if we clicked on a planet
			if selectedObj := sv.clickHandler.GetSelectedObject(); selectedObj != nil {
				if planet, ok := selectedObj.(*entities.Planet); ok {
					// Switch to planet view
					vm.SwitchTo(ViewTypePlanet)
					if planetView, ok := vm.GetView(ViewTypePlanet).(*PlanetView); ok {
						planetView.SetPlanet(planet)
					}
				}
			}
		} else {
			// Single click - check if clicking on a fleet
			clickedFleet := sv.getFleetAtPosition(x, y)
			if clickedFleet != nil && sv.fleetInfoUI != nil {
				sv.fleetInfoUI.ShowFleet(clickedFleet)
			}
		}

		sv.lastClickX = x
		sv.lastClickY = y
		sv.lastClickTime = currentTime

		sv.clickHandler.HandleClick(x, y)
	}

	return nil
}

// Draw implements View interface
func (sv *SystemView) Draw(screen *ebiten.Image) {
	// Fill background
	screen.Fill(utils.Background)

	if sv.system == nil {
		DrawText(screen, "No system selected", 10, 10, utils.TextPrimary)
		return
	}

	// Draw orbital paths
	sv.drawOrbitalPaths(screen)

	// Draw all entities (star, planets and stations)
	sv.drawEntities(screen)

	// Highlight selected object
	if selectedObj := sv.clickHandler.GetSelectedObject(); selectedObj != nil {
		x, y := selectedObj.GetPosition()
		DrawHighlightCircle(screen,
			int(x), int(y),
			int(selectedObj.GetClickRadius()),
			utils.Highlight)
	}

	// Draw context menu if active
	if sv.clickHandler.HasActiveMenu() {
		sv.clickHandler.GetActiveMenu().Draw(screen)
	}

	// Draw fleet info UI if visible
	if sv.fleetInfoUI != nil && sv.fleetInfoUI.IsVisible() {
		sv.fleetInfoUI.Draw(screen)
	}

	// Draw UI info
	title := fmt.Sprintf("System View: %s", sv.system.Name)
	DrawText(screen, title, 10, 10, utils.TextPrimary)
	DrawText(screen, "Press ESC to return to galaxy", 10, 25, utils.TextSecondary)
}

// updateFleets aggregates ships into fleets
func (sv *SystemView) updateFleets() {
	if sv.system == nil {
		return
	}
	fm := sv.ctx.GetFleetManager()
	sv.fleets = fm.AggregateFleets(sv.system)
}

// OnEnter implements View interface
func (sv *SystemView) OnEnter() {
	if sv.system != nil {
		sv.updateEntityPositions()
		sv.registerClickables()
	}
}

// OnExit implements View interface
func (sv *SystemView) OnExit() {
	sv.clickHandler.ClearClickables()
}

// GetType implements View interface
func (sv *SystemView) GetType() ViewType {
	return ViewTypeSystem
}

// updateEntityPositions calculates absolute positions for all entities based on their orbits
func (sv *SystemView) updateEntityPositions() {
	if sv.system == nil {
		return
	}

	for _, entity := range sv.system.Entities {
		orbitDistance := entity.GetOrbitDistance()
		orbitAngle := entity.GetOrbitAngle()

		// Add animation offset to orbit angle
		animatedAngle := orbitAngle + sv.orbitOffset

		// Scale the orbital distance
		scaledDistance := sv.scale.ScaleOrbitDistance(orbitDistance)

		// Calculate position based on scaled orbit with animation
		x := sv.centerX + scaledDistance*math.Cos(animatedAngle)
		y := sv.centerY + scaledDistance*math.Sin(animatedAngle)

		// Update absolute position using the SetAbsolutePosition method
		entity.SetAbsolutePosition(x, y)
	}
}

// registerClickables adds all entities as clickable objects
func (sv *SystemView) registerClickables() {
	sv.clickHandler.ClearClickables()

	if sv.system == nil {
		return
	}

	for _, entity := range sv.system.Entities {
		if clickable, ok := entity.(Clickable); ok {
			sv.clickHandler.AddClickable(clickable)
		}
	}
}

// drawStar renders a star entity
func (sv *SystemView) drawStar(screen *ebiten.Image, star *entities.Star) {
	centerX := int(sv.centerX)
	centerY := int(sv.centerY)
	// Scale the star radius based on the view scale
	radius := sv.scale.ScaleSize(float64(star.Radius))

	// Create star image
	starImg := ebiten.NewImage(radius*2, radius*2)

	// Draw a circle for the star
	for py := 0; py < radius*2; py++ {
		for px := 0; px < radius*2; px++ {
			dx := float64(px - radius)
			dy := float64(py - radius)
			dist := dx*dx + dy*dy

			if dist <= float64(radius*radius) {
				starImg.Set(px, py, star.Color)
			}
		}
	}

	// Draw the star
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(centerX-radius), float64(centerY-radius))
	screen.DrawImage(starImg, opts)

	// Draw star name above star (adjust for scaled star sizes)
	labelOffset := int(float64(radius) * 0.6)
	DrawCenteredText(screen, star.Name, centerX, centerY-radius-labelOffset)

	// Draw star type below star (adjust for scaled star sizes)
	DrawCenteredText(screen, fmt.Sprintf("(%s)", star.StarType), centerX, centerY+radius+labelOffset)
}

// drawOrbitalPaths draws the orbital rings
func (sv *SystemView) drawOrbitalPaths(screen *ebiten.Image) {
	orbitColor := color.RGBA{40, 40, 60, 100}

	// Get unique orbital distances (scaled)
	orbits := make(map[float64]bool)
	for _, entity := range sv.system.Entities {
		scaledDistance := sv.scale.ScaleOrbitDistance(entity.GetOrbitDistance())
		orbits[scaledDistance] = true
	}

	// Draw orbital rings
	for orbitDistance := range orbits {
		sv.drawOrbitRing(screen, orbitDistance, orbitColor)
	}
}

// drawOrbitRing draws a single orbital ring
func (sv *SystemView) drawOrbitRing(screen *ebiten.Image, radius float64, c color.RGBA) {
	segments := 100
	for i := 0; i < segments; i++ {
		angle1 := float64(i) * 2 * math.Pi / float64(segments)
		angle2 := float64(i+1) * 2 * math.Pi / float64(segments)

		x1 := int(sv.centerX + radius*math.Cos(angle1))
		y1 := int(sv.centerY + radius*math.Sin(angle1))
		x2 := int(sv.centerX + radius*math.Cos(angle2))
		y2 := int(sv.centerY + radius*math.Sin(angle2))

		DrawLine(screen, x1, y1, x2, y2, c)
	}
}

// drawEntities draws all stars, planets and stations
func (sv *SystemView) drawEntities(screen *ebiten.Image) {
	// Draw star first (in the center)
	for _, entity := range sv.system.GetEntitiesByType(entities.EntityTypeStar) {
		if star, ok := entity.(*entities.Star); ok {
			sv.drawStar(screen, star)
		}
	}

	// Draw planets
	for _, entity := range sv.system.GetEntitiesByType(entities.EntityTypePlanet) {
		if planet, ok := entity.(*entities.Planet); ok {
			sv.drawPlanet(screen, planet)
		}
	}

	// Draw stations
	for _, entity := range sv.system.GetEntitiesByType(entities.EntityTypeStation) {
		if station, ok := entity.(*entities.Station); ok {
			sv.drawStation(screen, station)
		}
	}

	// Draw fleets
	sv.drawFleets(screen)
}

// drawFleets draws all fleets in the system
func (sv *SystemView) drawFleets(screen *ebiten.Image) {
	for _, fleet := range sv.fleets {
		sv.drawFleet(screen, fleet)
	}
}

// drawFleet draws a fleet of ships
func (sv *SystemView) drawFleet(screen *ebiten.Image, fleet *Fleet) {
	if fleet == nil || len(fleet.Ships) == 0 {
		return
	}

	// Use lead ship's position
	x, y := fleet.GetPosition()
	centerX := int(x)
	centerY := int(y)
	size := 6

	// Draw ship as a triangle
	shipImg := ebiten.NewImage(size*2, size*2)
	for py := 0; py < size*2; py++ {
		for px := 0; px < size*2; px++ {
			dx := float64(px - size)
			dy := float64(py - size)
			if dy > 0 && math.Abs(dx) < float64(size)-dy/2 {
				shipImg.Set(px, py, fleet.LeadShip.Color)
			}
		}
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(centerX-size), float64(centerY-size))
	screen.DrawImage(shipImg, op)

	// If multiple ships, draw count badge
	if fleet.Size() > 1 {
		badge := fmt.Sprintf("%d", fleet.Size())
		badgeX := centerX + size - 2
		badgeY := centerY - size - 8

		// Badge background
		badgePanel := &UIPanel{
			X:           badgeX - 2,
			Y:           badgeY - 2,
			Width:       12,
			Height:      12,
			BgColor:     utils.PanelBg,
			BorderColor: utils.PanelBorder,
		}
		badgePanel.Draw(screen)

		DrawText(screen, badge, badgeX, badgeY, utils.Highlight)
	}

	// Draw fleet info
	if fleet.Size() == 1 {
		DrawText(screen, fleet.Ships[0].Name, centerX-30, centerY+size+5, utils.TextSecondary)
	} else {
		typeCounts := fleet.GetShipTypeCounts()
		fleetText := fmt.Sprintf("Fleet (%d ships)", fleet.Size())
		DrawText(screen, fleetText, centerX-40, centerY+size+5, utils.TextSecondary)

		// Show ship type breakdown
		offsetY := 18
		for shipType, count := range typeCounts {
			typeText := fmt.Sprintf("%dx %s", count, shipType)
			DrawText(screen, typeText, centerX-35, centerY+size+5+offsetY, utils.TextSecondary)
			offsetY += 12
		}
	}

	// Draw fuel indicator
	fuelPercent := fleet.GetAverageFuelPercent()
	fuelColor := utils.StationResearch // Green for good fuel
	if fuelPercent < 25 {
		fuelColor = utils.SystemRed
	} else if fuelPercent < 50 {
		fuelColor = utils.SystemOrange
	}
	fuelText := fmt.Sprintf("Fuel: %.0f%%", fuelPercent)
	DrawText(screen, fuelText, centerX-25, centerY+size+5+12, fuelColor)
}

// drawPlanet renders a single planet
func (sv *SystemView) drawPlanet(screen *ebiten.Image, planet *entities.Planet) {
	x, y := planet.GetAbsolutePosition()
	centerX := int(x)
	centerY := int(y)
	// Keep planet size consistent regardless of orbital scale
	radius := planet.Size

	// Draw ownership indicator if owned by player
	humanPlayer := sv.ctx.GetHumanPlayer()
	if planet.Owner != "" && humanPlayer != nil && planet.Owner == humanPlayer.Name {
		DrawOwnershipRing(screen, centerX, centerY, float64(radius+3), humanPlayer.Color)
	}

	// Create planet image
	planetImg := ebiten.NewImage(radius*2, radius*2)

	// Draw a circle for the planet
	for py := 0; py < radius*2; py++ {
		for px := 0; px < radius*2; px++ {
			dx := float64(px - radius)
			dy := float64(py - radius)
			dist := dx*dx + dy*dy

			if dist <= float64(radius*radius) {
				planetImg.Set(px, py, planet.Color)
			}
		}
	}

	// Draw the planet
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(centerX-radius), float64(centerY-radius))
	screen.DrawImage(planetImg, opts)

	// Draw planet name below
	labelY := centerY + radius + 12
	DrawCenteredText(screen, planet.Name, centerX, labelY)

	// Draw rings if planet has them
	if planet.HasRings {
		sv.drawPlanetRings(screen, centerX, centerY, radius)
	}
}

// drawPlanetRings draws rings around a planet
func (sv *SystemView) drawPlanetRings(screen *ebiten.Image, centerX, centerY, planetRadius int) {
	ringColor := color.RGBA{150, 150, 150, 150}
	ringRadius := float64(planetRadius) * 1.5

	segments := 40
	for i := 0; i < segments; i++ {
		angle1 := float64(i) * 2 * math.Pi / float64(segments)
		angle2 := float64(i+1) * 2 * math.Pi / float64(segments)

		x1 := centerX + int(ringRadius*math.Cos(angle1))
		y1 := centerY + int(ringRadius*math.Sin(angle1)*0.3) // Ellipse effect
		x2 := centerX + int(ringRadius*math.Cos(angle2))
		y2 := centerY + int(ringRadius*math.Sin(angle2)*0.3)

		DrawLine(screen, x1, y1, x2, y2, ringColor)
	}
}

// drawStation renders a single space station
func (sv *SystemView) drawStation(screen *ebiten.Image, station *entities.Station) {
	x, y := station.GetAbsolutePosition()
	centerX := int(x)
	centerY := int(y)
	// Keep station size consistent regardless of orbital scale
	size := 8

	// Draw station as a square/diamond
	stationImg := ebiten.NewImage(size*2, size*2)
	stationImg.Fill(station.Color)

	// Draw the station
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(centerX-size), float64(centerY-size))
	// Rotate 45 degrees to make it a diamond
	opts.GeoM.Translate(-float64(centerX), -float64(centerY))
	opts.GeoM.Rotate(math.Pi / 4)
	opts.GeoM.Translate(float64(centerX), float64(centerY))
	screen.DrawImage(stationImg, opts)

	// Draw station name below
	labelY := centerY + size + 12
	DrawCenteredText(screen, station.Name, centerX, labelY)
}

// getFleetAtPosition returns the fleet at the given screen position, or nil if none
func (sv *SystemView) getFleetAtPosition(x, y int) *Fleet {
	clickRadius := 15.0 // Click radius for fleets

	for _, fleet := range sv.fleets {
		fx, fy := fleet.GetPosition()
		dx := float64(x) - fx
		dy := float64(y) - fy
		distance := math.Sqrt(dx*dx + dy*dy)

		if distance <= clickRadius {
			return fleet
		}
	}

	return nil
}
