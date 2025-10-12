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
	game              *Game
	system            *entities.System
	planet            *entities.Planet
	clickHandler      *ClickHandler
	centerX           float64
	centerY           float64
	buildMenu         *BuildMenu
	constructionQueue *ConstructionQueueUI
	resourceStorage   *ResourceStorageUI
	orbitOffset       float64 // For animating orbits
}

// NewPlanetView creates a new planet view
func NewPlanetView(game *Game) *PlanetView {
	return &PlanetView{
		game:              game,
		clickHandler:      NewClickHandler(),
		centerX:           float64(screenWidth) / 2,
		centerY:           float64(screenHeight) / 2,
		buildMenu:         NewBuildMenu(game),
		constructionQueue: NewConstructionQueueUI(game),
		resourceStorage:   NewResourceStorageUI(game),
	}
}

// SetPlanet sets the planet to display
func (pv *PlanetView) SetPlanet(system *entities.System, planet *entities.Planet) {
	pv.system = system
	pv.planet = planet

	// Set planet position to center for click detection
	planet.SetAbsolutePosition(pv.centerX, pv.centerY)

	// Set planet for resource storage UI
	pv.resourceStorage.SetPlanet(planet)

	pv.updateResourcePositions()
	pv.registerClickables()
}

// Update implements View interface
func (pv *PlanetView) Update() error {
	// Update construction queue UI
	pv.constructionQueue.Update()

	// Update resource storage UI
	pv.resourceStorage.Update()

	// Update orbit animation
	if !pv.game.tickManager.IsPaused() {
		pv.orbitOffset += 0.001 * float64(pv.game.tickManager.GetSpeed())
		if pv.orbitOffset > 6.28318 { // 2*PI
			pv.orbitOffset -= 6.28318
		}
	}

	// Update resource/building positions for animation
	pv.updateResourcePositions()

	// Update build menu first (it handles its own input)
	if pv.buildMenu.IsOpen() {
		pv.buildMenu.Update()
		return nil
	}

	// ESC to return to system view
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		pv.game.viewManager.SwitchTo(ViewTypeSystem)
		return nil
	}

	// B key to open build menu on planet
	if inpututil.IsKeyJustPressed(ebiten.KeyB) && pv.planet != nil {
		if pv.game.humanPlayer != nil && pv.planet.Owner == pv.game.humanPlayer.Name {
			pv.buildMenu.Open(pv.planet, screenWidth/2, screenHeight/2)
		}
	}

	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		// Check if clicking on a resource to build
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			// Shift+click on resource opens build menu for that resource
			if selectedObj := pv.clickHandler.GetSelectedObject(); selectedObj != nil {
				if resource, ok := selectedObj.(*entities.Resource); ok {
					if pv.game.humanPlayer != nil && resource.Owner == pv.game.humanPlayer.Name {
						pv.buildMenu.Open(resource, x, y)
						return nil
					}
				}
			}
		}

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

	// Draw all buildings
	pv.drawBuildings(screen)

	// Highlight selected object
	if selectedObj := pv.clickHandler.GetSelectedObject(); selectedObj != nil {
		x, y := selectedObj.GetPosition()
		DrawHighlightCircle(screen,
			int(x), int(y),
			int(selectedObj.GetClickRadius()),
			UIHighlight)
	}

	// Draw context menu if active (but not if build menu is open)
	if pv.clickHandler.HasActiveMenu() && !pv.buildMenu.IsOpen() {
		pv.clickHandler.GetActiveMenu().Draw(screen)
	}

	// Draw UI info
	title := fmt.Sprintf("Planet View: %s", pv.planet.Name)
	DrawText(screen, title, 10, 10, UITextPrimary)
	DrawText(screen, fmt.Sprintf("Type: %s", pv.planet.PlanetType), 10, 25, UITextSecondary)
	DrawText(screen, fmt.Sprintf("Resources: %d deposits", len(pv.planet.Resources)), 10, 40, UITextSecondary)
	DrawText(screen, fmt.Sprintf("Buildings: %d", len(pv.planet.Buildings)), 10, 55, UITextSecondary)

	// Show build hints if player owns this planet
	if pv.game.humanPlayer != nil && pv.planet.Owner == pv.game.humanPlayer.Name {
		DrawText(screen, "[B] Build on planet  [Shift+Click] Build on resource", 10, 70, UITextSecondary)
		DrawText(screen, "Press ESC to return to system", 10, 85, UITextSecondary)
	} else {
		DrawText(screen, "Press ESC to return to system", 10, 70, UITextSecondary)
	}

	// Draw construction queue UI
	pv.constructionQueue.Draw(screen)

	// Draw resource storage UI
	pv.resourceStorage.Draw(screen)

	// Draw build menu on top of everything
	pv.buildMenu.Draw(screen)
}

// OnEnter implements View interface
func (pv *PlanetView) OnEnter() {
	if pv.planet != nil {
		pv.updateResourcePositions()
		pv.registerClickables()
	}
}

// RefreshPlanet refreshes the planet view (called when construction completes)
func (pv *PlanetView) RefreshPlanet() {
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

// updateResourcePositions positions resources and buildings at the planet's surface
func (pv *PlanetView) updateResourcePositions() {
	if pv.planet == nil {
		return
	}

	// Resources and buildings are positioned at the planet's surface edge
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

	// Position buildings slightly further out from resources
	buildingRadius := planetRadius + 15.0 // Buildings are offset from planet surface

	for _, building := range pv.planet.Buildings {
		// Use the orbit angle for positioning around the surface, with animation
		orbitAngle := building.GetOrbitAngle() + pv.orbitOffset

		// Position at building radius
		x := pv.centerX + buildingRadius*math.Cos(orbitAngle)
		y := pv.centerY + buildingRadius*math.Sin(orbitAngle)

		// Update absolute position
		building.SetAbsolutePosition(x, y)
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

	// Register buildings
	for _, building := range pv.planet.Buildings {
		if clickable, ok := building.(Clickable); ok {
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

	// Draw ownership indicator if owned by player
	if resource.Owner != "" && pv.game.humanPlayer != nil && resource.Owner == pv.game.humanPlayer.Name {
		DrawOwnershipRing(screen, centerX, centerY, float64(radius+2), pv.game.humanPlayer.Color)
	}

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

// drawBuildings draws all building entities
func (pv *PlanetView) drawBuildings(screen *ebiten.Image) {
	for _, building := range pv.planet.Buildings {
		if bldg, ok := building.(*entities.Building); ok {
			pv.drawBuilding(screen, bldg)
		}
	}
}

// drawBuilding renders a single building
func (pv *PlanetView) drawBuilding(screen *ebiten.Image, building *entities.Building) {
	x, y := building.GetAbsolutePosition()
	centerX := int(x)
	centerY := int(y)
	size := building.Size

	// Draw ownership indicator if owned by player
	if building.Owner != "" && pv.game.humanPlayer != nil && building.Owner == pv.game.humanPlayer.Name {
		DrawOwnershipRing(screen, centerX, centerY, float64(size+2), pv.game.humanPlayer.Color)
	}

	// Create building image (square for buildings)
	buildingImg := ebiten.NewImage(size*2, size*2)
	buildingImg.Fill(building.Color)

	// Draw the building
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(centerX-size), float64(centerY-size))
	screen.DrawImage(buildingImg, opts)

	// Draw building type label below
	labelY := centerY + size + 12
	DrawCenteredText(screen, building.BuildingType, centerX, labelY)
}
