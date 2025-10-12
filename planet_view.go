package main

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
)

// PlanetView represents the detailed view of a single planet
type PlanetView struct {
	game         *Game
	system       *System
	planet       *entities.Planet
	clickHandler *ClickHandler
	centerX      float64
	centerY      float64
}

// NewPlanetView creates a new planet view
func NewPlanetView(game *Game) *PlanetView {
	return &PlanetView{
		game:         game,
		clickHandler: NewClickHandler(),
		centerX:      float64(screenWidth) / 2,
		centerY:      float64(screenHeight) / 2,
	}
}

// SetPlanet sets the planet to display
func (pv *PlanetView) SetPlanet(system *System, planet *entities.Planet) {
	pv.system = system
	pv.planet = planet

	// Set planet position to center for click detection
	planet.SetAbsolutePosition(pv.centerX, pv.centerY)

	pv.updateResourcePositions()
	pv.registerClickables()
}

// Update implements View interface
func (pv *PlanetView) Update() error {
	// ESC to return to system view
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		pv.game.viewManager.SwitchTo(ViewTypeSystem)
		return nil
	}

	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		pv.clickHandler.HandleClick(x, y)
	}

	return nil
}

// Draw implements View interface
func (pv *PlanetView) Draw(screen *ebiten.Image) {
	// Fill background
	screen.Fill(UIBackground)

	if pv.planet == nil {
		DrawText(screen, "No planet selected", 10, 10, UITextPrimary)
		return
	}

	// Draw planet at center
	pv.drawPlanet(screen)

	// Draw all resources (no orbital paths for resources)
	pv.drawResources(screen)

	// Highlight selected object
	if selectedObj := pv.clickHandler.GetSelectedObject(); selectedObj != nil {
		x, y := selectedObj.GetPosition()
		DrawHighlightCircle(screen,
			int(x), int(y),
			int(selectedObj.GetClickRadius()),
			UIHighlight)
	}

	// Draw context menu if active
	if pv.clickHandler.HasActiveMenu() {
		pv.clickHandler.GetActiveMenu().Draw(screen)
	}

	// Draw UI info
	title := fmt.Sprintf("Planet View: %s", pv.planet.Name)
	DrawText(screen, title, 10, 10, UITextPrimary)
	DrawText(screen, fmt.Sprintf("Type: %s", pv.planet.PlanetType), 10, 25, UITextSecondary)
	DrawText(screen, fmt.Sprintf("Resources: %d deposits", len(pv.planet.Resources)), 10, 40, UITextSecondary)
	DrawText(screen, "Press ESC to return to system", 10, 55, UITextSecondary)
}

// OnEnter implements View interface
func (pv *PlanetView) OnEnter() {
	if pv.planet != nil {
		pv.updateResourcePositions()
		pv.registerClickables()
	}
}

// OnExit implements View interface
func (pv *PlanetView) OnExit() {
	pv.clickHandler.ClearClickables()
}

// GetType implements View interface
func (pv *PlanetView) GetType() ViewType {
	return ViewTypePlanet
}

// updateResourcePositions positions resources at the planet's surface
func (pv *PlanetView) updateResourcePositions() {
	if pv.planet == nil {
		return
	}

	// Resources are positioned at the planet's surface edge
	planetRadius := float64(pv.planet.Size * 8) // Same scaling as in drawPlanet

	for _, resource := range pv.planet.Resources {
		// Use the orbit angle for positioning around the surface
		orbitAngle := resource.GetOrbitAngle()

		// Position at planet surface edge
		x := pv.centerX + planetRadius*math.Cos(orbitAngle)
		y := pv.centerY + planetRadius*math.Sin(orbitAngle)

		// Update absolute position
		resource.SetAbsolutePosition(x, y)
	}
}

// registerClickables adds all resources as clickable objects
func (pv *PlanetView) registerClickables() {
	pv.clickHandler.ClearClickables()

	if pv.planet == nil {
		return
	}

	// Register resources first so they have priority over the planet
	for _, resource := range pv.planet.Resources {
		if clickable, ok := resource.(Clickable); ok {
			pv.clickHandler.AddClickable(clickable)
		}
	}

	// Register planet itself as clickable (checked last)
	pv.clickHandler.AddClickable(pv.planet)
}

// drawPlanet draws the planet at the center
func (pv *PlanetView) drawPlanet(screen *ebiten.Image) {
	centerX := int(pv.centerX)
	centerY := int(pv.centerY)
	// Scale up the planet for planet view
	radius := pv.planet.Size * 8

	// Create planet image
	planetImg := ebiten.NewImage(radius*2, radius*2)

	// Draw a circle for the planet
	for py := 0; py < radius*2; py++ {
		for px := 0; px < radius*2; px++ {
			dx := float64(px - radius)
			dy := float64(py - radius)
			dist := dx*dx + dy*dy

			if dist <= float64(radius*radius) {
				planetImg.Set(px, py, pv.planet.Color)
			}
		}
	}

	// Draw the planet
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(centerX-radius), float64(centerY-radius))
	screen.DrawImage(planetImg, opts)

	// Draw planet name above
	labelY := centerY - radius - 30
	DrawCenteredText(screen, pv.planet.Name, centerX, labelY)

	// Draw planet type below
	labelY = centerY + radius + 20
	DrawCenteredText(screen, fmt.Sprintf("(%s)", pv.planet.PlanetType), centerX, labelY)
}

// drawResources draws all resource deposits
func (pv *PlanetView) drawResources(screen *ebiten.Image) {
	for _, resource := range pv.planet.Resources {
		if res, ok := resource.(*entities.Resource); ok {
			pv.drawResource(screen, res)
		}
	}
}

// drawResource renders a single resource deposit
func (pv *PlanetView) drawResource(screen *ebiten.Image, resource *entities.Resource) {
	x, y := resource.GetAbsolutePosition()
	centerX := int(x)
	centerY := int(y)
	radius := resource.Size

	// Create resource image
	resourceImg := ebiten.NewImage(radius*2, radius*2)

	// Draw a circle for the resource
	for py := 0; py < radius*2; py++ {
		for px := 0; px < radius*2; px++ {
			dx := float64(px - radius)
			dy := float64(py - radius)
			dist := dx*dx + dy*dy

			if dist <= float64(radius*radius) {
				resourceImg.Set(px, py, resource.Color)
			}
		}
	}

	// Draw the resource
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(centerX-radius), float64(centerY-radius))
	screen.DrawImage(resourceImg, opts)

	// Draw resource type label below
	labelY := centerY + radius + 12
	DrawCenteredText(screen, resource.ResourceType, centerX, labelY)
}
