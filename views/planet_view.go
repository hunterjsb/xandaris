package views

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/rendering"
	"github.com/hunterjsb/xandaris/utils"
)

var (
	planetCircleCache      = utils.NewCircleImageCache()
	planetRectCache        = utils.NewRectImageCache()
	planetTriangleCache    = utils.NewTriangleImageCache()
	planetSpriteRenderer   *rendering.SpriteRenderer
	planetBuildingRenderer *rendering.BuildingRenderer
)

func init() {
	planetSpriteRenderer = rendering.NewSpriteRenderer()
	planetBuildingRenderer = rendering.NewBuildingRenderer(planetSpriteRenderer)
}

const (
	workforceButtonWidth  = 170
	workforceButtonHeight = 32
)

// BuildMenuInterface defines the interface for build menu operations
type BuildMenuInterface interface {
	Open(attachedTo entities.Entity, x, y int)
	Close()
	IsOpen() bool
	Update()
	Draw(screen *ebiten.Image)
}

// ConstructionQueueUIInterface defines the interface for construction queue UI
type ConstructionQueueUIInterface interface {
	Update()
	Draw(screen *ebiten.Image)
}

// ResourceStorageUIInterface defines the interface for resource storage UI
type ResourceStorageUIInterface interface {
	SetPlanet(planet *entities.Planet)
	Update()
	Draw(screen *ebiten.Image)
}

// ShipyardUIInterface defines the interface for shipyard UI
type ShipyardUIInterface interface {
	Show(planet *entities.Planet, building *entities.Building)
	Hide()
	IsVisible() bool
	Update()
	Draw(screen *ebiten.Image)
}

// FleetInfoUIInterface defines the interface for the fleet info UI
type FleetInfoUIInterface interface {
	ShowFleet(fleet *Fleet)
	ShowShip(ship *entities.Ship)
	Hide()
	IsVisible() bool
	Update()
	Draw(screen *ebiten.Image)
}

// PlanetView represents the detailed view of a single planet
type PlanetView struct {
	ctx               GameContext
	system            *entities.System
	planet            *entities.Planet
	clickHandler      *ClickHandler
	buildMenu         BuildMenuInterface
	constructionQueue ConstructionQueueUIInterface
	resourceStorage   ResourceStorageUIInterface
	shipyardUI        ShipyardUIInterface
	fleetInfoUI       FleetInfoUIInterface
	centerX           float64
	centerY           float64
	orbitOffset       float64 // For animating orbits
	fleets            []*Fleet
	workforceOverlay  *WorkforceOverlay
	spriteRenderer    *rendering.SpriteRenderer
	buildingRenderer  *rendering.BuildingRenderer
}

// NewPlanetView creates a new planet view
func NewPlanetView(ctx GameContext, buildMenu BuildMenuInterface, constructionQueue ConstructionQueueUIInterface, resourceStorage ResourceStorageUIInterface, shipyardUI ShipyardUIInterface, fleetInfoUI FleetInfoUIInterface) *PlanetView {
	spriteRenderer := rendering.NewSpriteRenderer()
	return &PlanetView{
		ctx:               ctx,
		clickHandler:      NewClickHandler("planet"),
		buildMenu:         buildMenu,
		constructionQueue: constructionQueue,
		resourceStorage:   resourceStorage,
		shipyardUI:        shipyardUI,
		fleetInfoUI:       fleetInfoUI,
		centerX:           float64(ScreenWidth) / 2,
		centerY:           float64(ScreenHeight) / 2,
		workforceOverlay:  NewWorkforceOverlay(),
		spriteRenderer:    spriteRenderer,
		buildingRenderer:  rendering.NewBuildingRenderer(spriteRenderer),
	}
}

// SetPlanet sets the planet to display
func (pv *PlanetView) SetPlanet(planet *entities.Planet) {
	pv.planet = planet
	if pv.workforceOverlay != nil {
		pv.workforceOverlay.Hide()
		pv.workforceOverlay.SetPlanet(planet)
	}

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

	// Set planet for resource storage UI
	if pv.resourceStorage != nil && planet != nil {
		pv.resourceStorage.SetPlanet(planet)
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

	// Update sprite animations based on game tick rate
	if pv.spriteRenderer != nil {
		tm := pv.ctx.GetTickManager()
		// Scale animation speed with tick rate (1 = normal, 2 = 2x faster, etc)
		speedMultiplier := int(tm.GetSpeedFloat())
		if speedMultiplier < 1 {
			speedMultiplier = 1
		}
		for i := 0; i < speedMultiplier; i++ {
			pv.spriteRenderer.Update()
		}
	}

	kb := pv.ctx.GetKeyBindings()
	vm := pv.ctx.GetViewManager()
	tm := pv.ctx.GetTickManager()

	if kb.IsActionJustPressed(ActionToggleWorkforceView) && pv.planet != nil {
		if pv.workforceOverlay != nil {
			pv.workforceOverlay.Toggle()
		}
	}

	// Update orbit animation (very slow for planetary rotation effect)
	if !tm.IsPaused() {
		speedVal := tm.GetSpeedFloat()
		pv.orbitOffset += 0.0001 * speedVal // 10x slower for rotation feel
		if pv.orbitOffset > 6.28318 {       // 2*PI
			pv.orbitOffset -= 6.28318
		}
	}

	// Update resource/building/ship positions for animation
	pv.updateResourcePositions()

	// Update fleet aggregation
	pv.updateFleets()

	// Update construction queue UI
	if pv.constructionQueue != nil {
		pv.constructionQueue.Update()
	}

	// Update resource storage UI
	if pv.resourceStorage != nil {
		pv.resourceStorage.Update()
	}

	// Update shipyard UI
	if pv.shipyardUI != nil {
		pv.shipyardUI.Update()
	}

	// Update fleet info UI
	if pv.fleetInfoUI != nil && pv.fleetInfoUI.IsVisible() {
		pv.fleetInfoUI.Update()
	}

	// Update build menu first (it handles its own input)
	if pv.buildMenu != nil && pv.buildMenu.IsOpen() {
		pv.buildMenu.Update()
		return nil
	}

	// Handle escape key
	if kb.IsActionJustPressed(ActionEscape) {
		if pv.workforceOverlay != nil && pv.workforceOverlay.Visible() {
			pv.workforceOverlay.Hide()
			return nil
		}

		// Close fleet info UI if open
		if pv.fleetInfoUI != nil && pv.fleetInfoUI.IsVisible() {
			pv.fleetInfoUI.Hide()
			return nil
		}
		// Close shipyard UI if open
		if pv.shipyardUI != nil && pv.shipyardUI.IsVisible() {
			pv.shipyardUI.Hide()
			return nil
		}
		vm.SwitchTo(ViewTypeSystem)
		return nil
	}

	// Open build menu on planet
	if kb.IsActionJustPressed(ActionOpenBuildMenu) && pv.planet != nil {
		humanPlayer := pv.ctx.GetHumanPlayer()
		if humanPlayer != nil && pv.planet.Owner == humanPlayer.Name && pv.buildMenu != nil {
			pv.buildMenu.Open(pv.planet, ScreenWidth/2, ScreenHeight/2)
		}
	}

	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		if rectContains(x, y, pv.workforceButtonRect()) && pv.planet != nil {
			if pv.workforceOverlay != nil {
				pv.workforceOverlay.Toggle()
			}
			return nil
		}

		if pv.workforceOverlay != nil && pv.workforceOverlay.Visible() {
			if pv.workforceOverlay.HandleClick(x, y) {
				return nil
			}
		}

		// Check if clicking on a fleet
		fm := pv.ctx.GetFleetManager()
		clickedFleet := fm.GetFleetAtPosition(pv.fleets, x, y, 15.0)
		if clickedFleet != nil {
			if pv.fleetInfoUI != nil {
				pv.fleetInfoUI.ShowFleet(clickedFleet)
			}
			return nil
		}

		// Check if clicking on a building
		for _, buildingEntity := range pv.planet.Buildings {
			if building, ok := buildingEntity.(*entities.Building); ok {
				bx, by := building.GetAbsolutePosition()
				dx := float64(x) - bx
				dy := float64(y) - by
				distance := dx*dx + dy*dy
				clickRadius := building.GetClickRadius("planet")

				if distance <= clickRadius*clickRadius {
					// Clicked on a building
					if building.BuildingType == "Shipyard" && building.IsOperational {
						humanPlayer := pv.ctx.GetHumanPlayer()
						if humanPlayer != nil && building.Owner == humanPlayer.Name && pv.shipyardUI != nil {
							pv.shipyardUI.Show(pv.planet, building)
							return nil
						}
					}
					return nil
				}
			}
		}

		// Check if shift+clicking on a resource to build
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			// Shift+click on resource opens build menu for that resource
			if selectedObj := pv.clickHandler.GetSelectedObject(); selectedObj != nil {
				if resource, ok := selectedObj.(*entities.Resource); ok {
					humanPlayer := pv.ctx.GetHumanPlayer()
					if humanPlayer != nil && resource.Owner == humanPlayer.Name && pv.buildMenu != nil {
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
	screen.Fill(utils.Background)

	if pv.planet == nil {
		DrawText(screen, "No planet selected", 10, 10, utils.TextPrimary)
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
			int(selectedObj.GetClickRadius("planet")),
			utils.Highlight)
	}

	// Draw context menu if active (but not if build menu is open)
	if pv.clickHandler.HasActiveMenu() && (pv.buildMenu == nil || !pv.buildMenu.IsOpen()) {
		pv.clickHandler.GetActiveMenu().Draw(screen)
	}

	// Draw UI info
	title := fmt.Sprintf("Planet View: %s", pv.planet.Name)
	DrawText(screen, title, 10, 10, utils.TextPrimary)

	infoY := 25
	for _, line := range formatPlanetDetails(pv.planet) {
		color := utils.TextSecondary
		if strings.HasPrefix(line, "Population") || strings.HasPrefix(line, "Housing") || strings.HasPrefix(line, "Workforce") {
			color = utils.TextPrimary
		}
		DrawText(screen, line, 10, infoY, color)
		infoY += 15
	}

	infoY += 10

	// Show build hints if player owns this planet
	humanPlayer := pv.ctx.GetHumanPlayer()
	if humanPlayer != nil && pv.planet.Owner == humanPlayer.Name {
		DrawText(screen, "[B] Build on planet  [Shift+Click] Build on resource", 10, infoY, utils.TextSecondary)
		DrawText(screen, "Press ESC to return to system", 10, infoY+15, utils.TextSecondary)
	} else {
		DrawText(screen, "Press ESC to return to system", 10, infoY, utils.TextSecondary)
	}

	pv.drawWorkforceToggleButton(screen)

	if pv.workforceOverlay != nil && pv.workforceOverlay.Visible() {
		pv.workforceOverlay.Draw(screen)
		return
	}

	// Draw construction queue UI
	if pv.constructionQueue != nil {
		pv.constructionQueue.Draw(screen)
	}

	// Draw resource storage UI
	if pv.resourceStorage != nil {
		pv.resourceStorage.Draw(screen)
	}

	// Draw shipyard UI if visible
	if pv.shipyardUI != nil {
		pv.shipyardUI.Draw(screen)
	}

	// Draw fleet info UI if visible
	if pv.fleetInfoUI != nil && pv.fleetInfoUI.IsVisible() {
		pv.fleetInfoUI.Draw(screen)
	}

	// Draw build menu if visible (on top of everything)
	if pv.buildMenu != nil {
		pv.buildMenu.Draw(screen)
	}
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
	planetRadius := float64(pv.planet.Size * 24) // Same scaling as in drawPlanet (3x scale)

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
	buildingRadius := planetRadius + 60.0 // 3x scale (was 20.0)

	// Count non-mine buildings to distribute them evenly
	nonMineBuildings := make([]entities.Entity, 0)
	for _, building := range pv.planet.Buildings {
		if bldg, ok := building.(*entities.Building); ok {
			if bldg.BuildingType != "Mine" {
				nonMineBuildings = append(nonMineBuildings, building)
			}
		}
	}

	nonMineIndex := 0
	for _, building := range pv.planet.Buildings {
		var orbitAngle float64

		// If this is a mine, position it at the resource node
		if bldg, ok := building.(*entities.Building); ok && bldg.BuildingType == "Mine" && bldg.ResourceNodeID != 0 {
			// Find the associated resource node
			for _, resource := range pv.planet.Resources {
				if resource.GetID() == bldg.ResourceNodeID {
					if res, ok := resource.(*entities.Resource); ok {
						// Use the resource's node position
						orbitAngle = res.NodePosition + pv.orbitOffset
					}
					break
				}
			}
		} else {
			// Non-mine buildings (shipyards, refineries) are distributed evenly around the planet
			if len(nonMineBuildings) > 0 {
				angleStep := (2.0 * math.Pi) / float64(len(nonMineBuildings))
				orbitAngle = float64(nonMineIndex)*angleStep + pv.orbitOffset
				nonMineIndex++
			} else {
				orbitAngle = building.GetOrbitAngle() + pv.orbitOffset
			}
		}

		// Position at building radius
		x := pv.centerX + buildingRadius*math.Cos(orbitAngle)
		y := pv.centerY + buildingRadius*math.Sin(orbitAngle)

		// Update absolute position
		building.SetAbsolutePosition(x, y)
	}

	// Ships orbit further out than buildings and orbit faster
	if pv.system != nil {
		shipRadius := planetRadius + 120.0     // 3x scale (was 40.0)
		shipOrbitSpeed := pv.orbitOffset * 8.0 // Ships orbit 8x faster than surface
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

	// Register planet first, so it's checked last and doesn't steal clicks
	pv.clickHandler.AddClickable(pv.planet)

	// Register resources and buildings, which will have click priority over the planet
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
}

// drawPlanet draws the planet at the center
func (pv *PlanetView) drawPlanet(screen *ebiten.Image) {
	centerX := int(pv.centerX)
	centerY := int(pv.centerY)
	// Scale up the planet for planet view
	radius := pv.planet.Size * 24

	// Try to render planet with sprite, fallback to cached circle
	sprite, err := pv.spriteRenderer.GetAssetLoader().LoadPlanetSprite(pv.planet.PlanetType)
	if err == nil && sprite != nil {
		// Render animated sprite
		frame := sprite.GetFrame(pv.spriteRenderer.GetAnimationTick())
		if frame != nil {
			opts := &ebiten.DrawImageOptions{}

			// Scale to match desired radius
			bounds := frame.Bounds()
			frameWidth := float64(bounds.Dx())
			frameHeight := float64(bounds.Dy())
			scale := float64(radius*2) / frameWidth

			// Center and scale
			opts.GeoM.Translate(-frameWidth/2, -frameHeight/2)
			opts.GeoM.Scale(scale, scale)
			opts.GeoM.Translate(float64(centerX), float64(centerY))

			screen.DrawImage(frame, opts)
		} else {
			// Fallback to circle if frame is nil
			planetImg := planetCircleCache.GetOrCreate(radius, pv.planet.Color)
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(centerX-radius), float64(centerY-radius))
			screen.DrawImage(planetImg, opts)
		}
	} else {
		// Fallback to circle if sprite not found
		planetImg := planetCircleCache.GetOrCreate(radius, pv.planet.Color)
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(centerX-radius), float64(centerY-radius))
		screen.DrawImage(planetImg, opts)
	}

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

	if resource.Owner != "" {
		if ownerColor, ok := pv.getOwnerColor(resource.Owner); ok {
			DrawOwnershipRing(screen, centerX, centerY, float64(radius+6), ownerColor)
		}
	}

	// Try to load resource sprite
	sprite, err := pv.spriteRenderer.GetAssetLoader().LoadResourceSprite(resource.ResourceType)
	if err == nil && sprite != nil {
		// Render sprite
		opts := &ebiten.DrawImageOptions{}

		// Scale to match desired size
		bounds := sprite.Bounds()
		spriteWidth := float64(bounds.Dx())
		spriteHeight := float64(bounds.Dy())
		scale := float64(radius*2) / spriteWidth

		// Center and scale
		opts.GeoM.Translate(-spriteWidth/2, -spriteHeight/2)
		opts.GeoM.Scale(scale, scale)
		opts.GeoM.Translate(float64(centerX), float64(centerY))

		screen.DrawImage(sprite, opts)
	} else {
		// Fallback to cached circle
		resourceImg := planetCircleCache.GetOrCreate(radius, resource.Color)
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(centerX-radius), float64(centerY-radius))
		screen.DrawImage(resourceImg, opts)
	}

	// Render any attached buildings
	attachedBuildings := resource.GetAttachedBuildings()
	for i, building := range attachedBuildings {
		// Position buildings around the resource in a circle
		angle := (float64(i) / float64(len(attachedBuildings))) * 2 * math.Pi
		buildingRadius := float64(radius + 45) // Increase spacing for 3x scale
		buildingX := centerX + int(buildingRadius*math.Cos(angle))
		buildingY := centerY + int(buildingRadius*math.Sin(angle))

		pv.buildingRenderer.RenderBuilding(screen, building, buildingX, buildingY)

		// Draw connection line
		DrawLine(screen, centerX, centerY, buildingX, buildingY, building.Color)
	}

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

	if fleet.Owner != "" {
		if ownerColor, ok := pv.getOwnerColor(fleet.Owner); ok {
			DrawOwnershipRing(screen, centerX, centerY, float64(size+9), ownerColor)
		}
	}

	// Render fleet with sprite renderer
	pv.spriteRenderer.RenderFleet(screen, centerX, centerY, size, fleet.LeadShip.Color)

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
		fleetText := fmt.Sprintf("Fleet (%d ships)", fleet.Size())
		DrawText(screen, fleetText, centerX-40, centerY+size+5, utils.TextSecondary)
	}
}

// drawBuilding renders a single building
func (pv *PlanetView) drawBuilding(screen *ebiten.Image, building *entities.Building) {
	x, y := building.GetAbsolutePosition()
	centerX := int(x)
	centerY := int(y)

	// Use the building renderer with attachment support
	pv.buildingRenderer.RenderBuildingWithAttachments(screen, building, centerX, centerY)

	// Draw building type label below
	labelY := centerY + building.Size + 12
	DrawCenteredText(screen, building.BuildingType, centerX, labelY)
}

func (pv *PlanetView) getOwnerColor(owner string) (color.RGBA, bool) {
	if owner == "" {
		return color.RGBA{}, false
	}

	for _, player := range pv.ctx.GetPlayers() {
		if player != nil && player.Name == owner {
			return player.Color, true
		}
	}

	return color.RGBA{}, false
}

func formatPlanetDetails(planet *entities.Planet) []string {
	if planet == nil {
		return nil
	}

	owner := planet.Owner
	if owner == "" {
		owner = "Unclaimed"
	}

	lines := []string{
		fmt.Sprintf("Owner: %s", owner),
		fmt.Sprintf("Type: %s", planet.PlanetType),
		fmt.Sprintf("Atmosphere: %s", planet.Atmosphere),
		fmt.Sprintf("Temperature: %d°C", planet.Temperature),
		fmt.Sprintf("Habitability: %d%%", planet.Habitability),
	}

	populationStr := utils.FormatInt64WithCommas(planet.Population)
	capacity := planet.GetTotalPopulationCapacity()
	if capacity > 0 {
		capacityStr := utils.FormatInt64WithCommas(capacity)
		lines = append(lines, fmt.Sprintf("Population: %s / %s", populationStr, capacityStr))

		baseHousing := planet.GetBaseHousingCapacity()
		otherHousing := capacity - baseHousing
		if otherHousing < 0 {
			otherHousing = 0
		}

		lines = append(lines, fmt.Sprintf(
			"Housing: %s (Base %s | Buildings %s)",
			capacityStr,
			utils.FormatInt64WithCommas(baseHousing),
			utils.FormatInt64WithCommas(otherHousing),
		))
	} else {
		lines = append(lines, fmt.Sprintf("Population: %s (no housing)", populationStr))
	}

	if planet.WorkforceTotal > 0 {
		lines = append(lines, fmt.Sprintf(
			"Workforce: %s / %s",
			utils.FormatInt64WithCommas(planet.WorkforceUsed),
			utils.FormatInt64WithCommas(planet.WorkforceTotal),
		))
	}

	lines = append(lines, fmt.Sprintf("Resources: %d deposits", len(planet.Resources)))
	lines = append(lines, fmt.Sprintf("Buildings: %d", len(planet.Buildings)))

	return lines
}

func (pv *PlanetView) drawWorkforceToggleButton(screen *ebiten.Image) {
	rect := pv.workforceButtonRect()
	panel := &UIPanel{
		X:           rect.Min.X,
		Y:           rect.Min.Y,
		Width:       rect.Dx(),
		Height:      rect.Dy(),
		BgColor:     utils.PanelBg,
		BorderColor: utils.PanelBorder,
	}
	if pv.workforceOverlay != nil && pv.workforceOverlay.Visible() {
		panel.BgColor = utils.ButtonActive
	}
	panel.Draw(screen)

	label := "Workforce [W]"
	textColor := utils.TextSecondary
	if pv.workforceOverlay != nil && pv.workforceOverlay.Visible() {
		textColor = utils.TextPrimary
	}
	DrawTextCenteredInRect(screen, label, rect, textColor)
}

func (pv *PlanetView) workforceButtonRect() image.Rectangle {
	x := ScreenWidth - workforceButtonWidth - 20
	y := 10
	return image.Rect(x, y, x+workforceButtonWidth, y+workforceButtonHeight)
}

func rectContains(x, y int, rect image.Rectangle) bool {
	return x >= rect.Min.X && x < rect.Max.X && y >= rect.Min.Y && y < rect.Max.Y
}

type barSegment struct {
	Label string
	Value float64
	Color color.RGBA
}

func drawStackedBar(screen *ebiten.Image, x, y, width, height int, segments []barSegment) {
	if width <= 0 || height <= 0 {
		return
	}

	barImg := planetRectCache.GetOrCreate(width, height, utils.BackgroundDark)

	total := 0.0
	for _, seg := range segments {
		if seg.Value > 0 {
			total += seg.Value
		}
	}

	if total > 0 {
		offset := 0
		for idx, seg := range segments {
			if seg.Value <= 0 {
				continue
			}
			ratio := seg.Value / total
			segWidth := int(ratio * float64(width))
			if segWidth <= 0 && seg.Value > 0 {
				if idx == len(segments)-1 {
					segWidth = width - offset
				} else {
					segWidth = 1
				}
			}
			if offset+segWidth > width {
				segWidth = width - offset
			}
			if segWidth <= 0 {
				continue
			}

			segImg := planetRectCache.GetOrCreate(segWidth, height, seg.Color)
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(offset), 0)
			barImg.DrawImage(segImg, opts)
			offset += segWidth
			if offset >= width {
				break
			}
		}
	}

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(barImg, opts)
	DrawRectOutline(screen, x, y, width, height, utils.PanelBorder)
}
func drawLegend(screen *ebiten.Image, x, y int, segments []barSegment) int {
	currentY := y
	for _, seg := range segments {
		if seg.Value <= 0 {
			continue
		}
		drawColorSwatch(screen, x, currentY, seg.Color)
		label := fmt.Sprintf("%s (%s)", seg.Label, utils.FormatInt64WithCommas(int64(seg.Value+0.5)))
		DrawText(screen, label, x+18, currentY+12, utils.TextSecondary)
		currentY += 20
	}
	return currentY
}
