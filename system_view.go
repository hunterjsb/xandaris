package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// SystemView represents the detailed view of a single system
type SystemView struct {
	game         *Game
	system       *System
	clickHandler *ClickHandler
	centerX      float64
	centerY      float64
}

// NewSystemView creates a new system view
func NewSystemView(game *Game) *SystemView {
	return &SystemView{
		game:         game,
		clickHandler: NewClickHandler(),
		centerX:      float64(screenWidth) / 2,
		centerY:      float64(screenHeight) / 2,
	}
}

// SetSystem sets the system to display
func (sv *SystemView) SetSystem(system *System) {
	sv.system = system
	sv.updateEntityPositions()
	sv.registerClickables()
}

// Update implements View interface
func (sv *SystemView) Update() error {
	// ESC to return to galaxy view
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		sv.game.viewManager.SwitchTo(ViewTypeGalaxy)
		return nil
	}

	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		sv.clickHandler.HandleClick(x, y)
	}

	return nil
}

// Draw implements View interface
func (sv *SystemView) Draw(screen *ebiten.Image) {
	// Fill background
	screen.Fill(UIBackground)

	if sv.system == nil {
		ebitenutil.DebugPrint(screen, "No system selected")
		return
	}

	// Draw system center (star)
	sv.drawSystemCenter(screen)

	// Draw orbital paths
	sv.drawOrbitalPaths(screen)

	// Draw all entities (planets and stations)
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
	title := fmt.Sprintf("System View: %s\nPress ESC to return to galaxy", sv.system.Name)
	ebitenutil.DebugPrint(screen, title)
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

		// Calculate position based on orbit
		x := sv.centerX + orbitDistance*math.Cos(orbitAngle)
		y := sv.centerY + orbitDistance*math.Sin(orbitAngle)

		// Update absolute position based on entity type
		switch e := entity.(type) {
		case *Planet:
			e.AbsoluteX = x
			e.AbsoluteY = y
		case *SpaceStation:
			e.AbsoluteX = x
			e.AbsoluteY = y
		}
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

// drawSystemCenter draws the star at the center of the system
func (sv *SystemView) drawSystemCenter(screen *ebiten.Image) {
	starRadius := 15
	starColor := color.RGBA{255, 255, 200, 255}

	// Create star image
	starImg := ebiten.NewImage(starRadius*2, starRadius*2)

	// Draw a circle for the star
	for py := 0; py < starRadius*2; py++ {
		for px := 0; px < starRadius*2; px++ {
			dx := float64(px - starRadius)
			dy := float64(py - starRadius)
			dist := dx*dx + dy*dy

			if dist <= float64(starRadius*starRadius) {
				starImg.Set(px, py, starColor)
			}
		}
	}

	// Draw the star
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(sv.centerX-float64(starRadius), sv.centerY-float64(starRadius))
	screen.DrawImage(starImg, opts)

	// Draw system name above star
	DrawCenteredText(screen, sv.system.Name, int(sv.centerX), int(sv.centerY)-starRadius-20)
}

// drawOrbitalPaths draws the orbital rings
func (sv *SystemView) drawOrbitalPaths(screen *ebiten.Image) {
	orbitColor := color.RGBA{40, 40, 60, 100}

	// Get unique orbital distances
	orbits := make(map[float64]bool)
	for _, entity := range sv.system.Entities {
		orbits[entity.GetOrbitDistance()] = true
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

// drawEntities draws all planets and stations
func (sv *SystemView) drawEntities(screen *ebiten.Image) {
	// Draw planets
	for _, entity := range sv.system.GetEntitiesByType("Planet") {
		if planet, ok := entity.(*Planet); ok {
			sv.drawPlanet(screen, planet)
		}
	}

	// Draw stations
	for _, entity := range sv.system.GetEntitiesByType("Station") {
		if station, ok := entity.(*SpaceStation); ok {
			sv.drawStation(screen, station)
		}
	}
}

// drawPlanet renders a single planet
func (sv *SystemView) drawPlanet(screen *ebiten.Image, planet *Planet) {
	centerX := int(planet.AbsoluteX)
	centerY := int(planet.AbsoluteY)
	radius := planet.Size

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
func (sv *SystemView) drawStation(screen *ebiten.Image, station *SpaceStation) {
	centerX := int(station.AbsoluteX)
	centerY := int(station.AbsoluteY)
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
