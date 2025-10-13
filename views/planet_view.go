package views

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
)

// PlanetView represents the detailed view of a single planet
type PlanetView struct {
	ctx          GameContext
	system       *entities.System
	planet       *entities.Planet
	clickHandler *ClickHandler
	centerX      float64
	centerY      float64
	orbitOffset  float64 // For animating orbits
	fleets       []*Fleet
}

// NewPlanetView creates a new planet view
func NewPlanetView(ctx GameContext) *PlanetView {
	return &PlanetView{
		ctx:          ctx,
		clickHandler: NewClickHandler(),
		centerX:      float64(ScreenWidth) / 2,
		centerY:      float64(ScreenHeight) / 2,
	}
}

// SetPlanet sets the planet to display
func (pv *PlanetView) SetPlanet(planet *entities.Planet) {
	pv.planet = planet

	// Find the system that contains this planet
	pv.system = nil
	for _, sys := range pv.ctx.GetSystems() {
		for _, entity := range sys.Entities {
			if p, ok := entity.(*entities.Planet); ok && p == planet {
				pv.system = sys
				break
			}
		}
		if pv.system != nil {
			break
		}
	}

	// Set planet position to center for click detection
	if planet != nil {
		planet.SetAbsolutePosition(pv.centerX, pv.centerY)
	}

	pv.updateResourcePositions()
	pv.updateFleets()
	pv.registerClickables()
}

// Update implements View interface
func (pv *PlanetView) Update() error {
	if pv.planet == nil {
		return nil
	}

	kb := pv.ctx.GetKeyBindings()
	vm := pv.ctx.GetViewManager()
	tm := pv.ctx.GetTickManager()

	// Update orbit animation (very slow for planetary rotation effect)
	if !tm.IsPaused() {
		if speed, ok := tm.GetSpeed().(float64); ok {
			pv.orbitOffset += 0.0001 * speed // 10x slower for rotation feel
			if pv.orbitOffset > 6.28318 {    // 2*PI
				pv.orbitOffset -= 6.28318
			}
		}
	}

	// Update resource/building/ship positions for animation
	pv.updateResourcePositions()

	// Update fleet aggregation
	pv.updateFleets()

	// Handle escape key
	if kb.IsActionJustPressed(ActionEscape) {
		vm.SwitchTo(ViewTypeSystem)
		return nil
	}

	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		// Check if clicking on a fleet
		fm := pv.ctx.GetFleetManager()
		clickedFleet := fm.GetFleetAtPosition(pv.fleets, x, y, 15.0)
		if clickedFleet != nil {
			// TODO: Show fleet info when we port FleetInfoUI
			return nil
		}

		// Check if clicking on a building
		for _, buildingEntity := range pv.planet.Buildings {
			if building, ok := buildingEntity.(*entities.Building); ok {
				bx, by := building.GetAbsolutePosition()
				dx := float64(x) - bx
				dy := float64(y) - by
				distance := dx*dx + dy*dy
				clickRadius := building.GetClickRadius()

				if distance <= clickRadius*clickRadius {
					// Clicked on a building - handle shipyard, etc.
					// TODO: Implement when we port ShipyardUI
					return nil
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

	// Draw all resources
	pv.drawResources(screen)

	// Draw all buildings
	pv.drawBuildings(screen)

	// Draw all fleets orbiting this planet
	pv.drawFleets(screen)

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
	DrawText(screen, fmt.Sprintf("Buildings: %d", len(pv.planet.Buildings)), 10, 55, UITextSecondary)
	DrawText(screen, "Press ESC to return to system", 10, 70, UITextSecondary)
}

// OnEnter implements View interface
func (pv *PlanetView) OnEnter() {
	if pv.planet != nil {
		pv.updateResourcePositions()
		pv.updateFleets()
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

// updateResourcePositions positions resources, buildings, and ships at the planet's surface
func (pv *PlanetView) updateResourcePositions() {
	if pv.planet == nil {
		return
	}

	// Resources and buildings are positioned at the planet's surface edge
	planetRadius := float64(pv.planet.Size * 8) // Same scaling as in drawPlanet

	for _, resource := range pv.planet.Resources {
		// Use the orbit angle for positioning around the surface
		orbitAngle := resource.GetOrbitAngle()
		// Add animation offset
		animatedAngle := orbitAngle + pv.orbitOffset

		// Position at planet surface
		x := pv.centerX + planetRadius*math.Cos(animatedAngle)
		y := pv.centerY + planetRadius*math.Sin(animatedAngle)

		// Update absolute position
		resource.SetAbsolutePosition(x, y)
	}

	// Buildings orbit slightly further out than resources
	buildingRadius := planetRadius + 15.0
	for _, building := range pv.planet.Buildings {
		var orbitAngle float64

		// If this is a mine, position it at the resource node
		if bldg, ok := building.(*entities.Building); ok && bldg.BuildingType == "Mine" && bldg.ResourceNodeID != 0 {
			// Find the associated resource node
			for _, resource := range pv.planet.Resources {
				if resource.GetID() == bldg.ResourceNodeID {
					if res, ok := resource.(*entities.Resource); ok {
						// Use the resource's node position (fixed)
						orbitAngle = res.NodePosition + pv.orbitOffset
					}
					break
				}
			}
		} else {
			// Non-mine buildings use their own orbit angle with animation
			orbitAngle = building.GetOrbitAngle() + pv.orbitOffset
		}

		// Position at building radius
		x := pv.centerX + buildingRadius*math.Cos(orbitAngle)
		y := pv.centerY + buildingRadius*math.Sin(orbitAngle)

		// Update absolute position
		building.SetAbsolutePosition(x, y)
	}

	// Ships orbit further out than buildings and orbit faster
	if pv.system != nil {
		shipRadius := planetRadius + 40.0
		shipOrbitSpeed := pv.orbitOffset * 10.0 // Ships orbit 10x faster than surface
		for _, entity := range pv.system.Entities {
			if ship, ok := entity.(*entities.Ship); ok {
				// Only show ships that are orbiting THIS specific planet
				planetOrbit := pv.planet.GetOrbitDistance()
				shipOrbit := ship.GetOrbitDistance()

				// Ships must be at the EXACT same orbital distance as this planet
				if math.Abs(planetOrbit-shipOrbit) < 1.0 {
					// Use the ship's orbit angle relative to planet, with faster animation
					angle := ship.GetOrbitAngle() - pv.planet.GetOrbitAngle() + shipOrbitSpeed

					// Position at ship orbit radius around this planet
					x := pv.centerX + shipRadius*math.Cos(angle)
					y := pv.centerY + shipRadius*math.Sin(angle)

					// Update absolute position
					ship.SetAbsolutePosition(x, y)
				}
			}
		}
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
	humanPlayer := pv.ctx.GetHumanPlayer()
	if resource.Owner != "" && humanPlayer != nil && resource.Owner == humanPlayer.Name {
		DrawOwnershipRing(screen, centerX, centerY, float64(radius+2), humanPlayer.Color)
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

// updateFleets aggregates ships into fleets at this planet
func (pv *PlanetView) updateFleets() {
	if pv.system == nil || pv.planet == nil {
		return
	}
	// Only aggregate ships that are actually at this planet's orbital distance
	fm := pv.ctx.GetFleetManager()
	pv.fleets = fm.AggregateFleetsAtPlanet(pv.system, pv.planet)
}

// drawFleets draws all fleets orbiting this planet
func (pv *PlanetView) drawFleets(screen *ebiten.Image) {
	for _, fleet := range pv.fleets {
		pv.drawFleet(screen, fleet)
	}
}

// drawFleet draws a fleet of ships
func (pv *PlanetView) drawFleet(screen *ebiten.Image, fleet *Fleet) {
	if fleet == nil || len(fleet.Ships) == 0 {
		return
	}

	// Use lead ship's position
	x, y := fleet.GetPosition()
	centerX := int(x)
	centerY := int(y)
	size := 6

	// Draw ownership indicator if owned by player
	humanPlayer := pv.ctx.GetHumanPlayer()
	if fleet.Owner != "" && humanPlayer != nil && fleet.Owner == humanPlayer.Name {
		DrawOwnershipRing(screen, centerX, centerY, float64(size+3), humanPlayer.Color)
	}

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
			BgColor:     UIPanelBg,
			BorderColor: UIPanelBorder,
		}
		badgePanel.Draw(screen)

		DrawText(screen, badge, badgeX, badgeY, UIHighlight)
	}

	// Draw fleet info
	if fleet.Size() == 1 {
		DrawText(screen, fleet.Ships[0].Name, centerX-30, centerY+size+5, UITextSecondary)
	} else {
		fleetText := fmt.Sprintf("Fleet (%d ships)", fleet.Size())
		DrawText(screen, fleetText, centerX-40, centerY+size+5, UITextSecondary)
	}
}

// drawBuilding renders a single building
func (pv *PlanetView) drawBuilding(screen *ebiten.Image, building *entities.Building) {
	x, y := building.GetAbsolutePosition()
	centerX := int(x)
	centerY := int(y)
	size := building.Size

	// Draw ownership indicator if owned by player
	humanPlayer := pv.ctx.GetHumanPlayer()
	if building.Owner != "" && humanPlayer != nil && building.Owner == humanPlayer.Name {
		DrawOwnershipRing(screen, centerX, centerY, float64(size+2), humanPlayer.Color)
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
