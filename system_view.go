package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
)

// SystemView represents the detailed view of a single system
type SystemView struct {
	game              *Game
	system            *System
	clickHandler      *ClickHandler
	centerX           float64
	centerY           float64
	scale             *ViewScale
	lastClickX        int
	lastClickY        int
	lastClickTime     int64
	constructionQueue *ConstructionQueueUI
	orbitOffset       float64 // For animating orbits
}

// NewSystemView creates a new system view
func NewSystemView(game *Game) *SystemView {
	return &SystemView{
		game:              game,
		clickHandler:      NewClickHandler(),
		centerX:           float64(screenWidth) / 2,
		centerY:           float64(screenHeight) / 2,
		scale:             &SystemScale,
		constructionQueue: NewConstructionQueueUI(game),
	}
}

// SetSystem sets the system to display
func (sv *SystemView) SetSystem(system *System) {
	sv.system = system

	// Calculate auto-scaling based on system size
	maxDistance := GetSystemMaxOrbitDistance(system)
	sv.scale = AutoScale(maxDistance, screenWidth, screenHeight)

	sv.updateEntityPositions()
	sv.registerClickables()
}

// Update implements View interface
func (sv *SystemView) Update() error {
	// Update construction queue UI
	sv.constructionQueue.Update()

	// Update orbit animation
	if !sv.game.tickManager.IsPaused() {
		sv.orbitOffset += 0.0005 * float64(sv.game.tickManager.GetSpeed())
		if sv.orbitOffset > 6.28318 { // 2*PI
			sv.orbitOffset -= 6.28318
		}
	}

	// Update entity positions for animation
	sv.updateEntityPositions()

	// ESC to return to galaxy view
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		sv.game.viewManager.SwitchTo(ViewTypeGalaxy)
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
					sv.game.viewManager.SwitchTo(ViewTypePlanet)
					if planetView, ok := sv.game.viewManager.GetCurrentView().(*PlanetView); ok {
						planetView.SetPlanet(sv.system, planet)
					}
				}
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
	screen.Fill(UIBackground)

	if sv.system == nil {
		DrawText(screen, "No system selected", 10, 10, UITextPrimary)
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
			UIHighlight)
	}

	// Draw context menu if active
	if sv.clickHandler.HasActiveMenu() {
		sv.clickHandler.GetActiveMenu().Draw(screen)
	}

	// Draw UI info
	title := fmt.Sprintf("System View: %s", sv.system.Name)
	DrawText(screen, title, 10, 10, UITextPrimary)
	DrawText(screen, "Press ESC to return to galaxy", 10, 25, UITextSecondary)

	// Draw construction queue UI
	sv.constructionQueue.Draw(screen)
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
}

// drawPlanet renders a single planet
func (sv *SystemView) drawPlanet(screen *ebiten.Image, planet *entities.Planet) {
	x, y := planet.GetAbsolutePosition()
	centerX := int(x)
	centerY := int(y)
	// Keep planet size consistent regardless of orbital scale
	radius := planet.Size

	// Draw ownership indicator if owned by player
	if planet.Owner != "" && sv.game.humanPlayer != nil && planet.Owner == sv.game.humanPlayer.Name {
		DrawOwnershipRing(screen, centerX, centerY, float64(radius+3), sv.game.humanPlayer.Color)
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
