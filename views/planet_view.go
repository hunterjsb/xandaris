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

var (
	workforceButtonWidth  = int(170.0 * utils.UIScale)
	workforceButtonHeight = int(32.0 * utils.UIScale)
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
	ShowFleet(fleet *entities.Fleet)
	ShowShip(ship *entities.Ship)
	Hide()
	IsVisible() bool
	Update()
	Draw(screen *ebiten.Image)
}

// PlanetDataProviderInterface abstracts local vs remote planet data access.
type PlanetDataProviderInterface interface {
	SetPlanetID(id int)
	Update()
	PopulatePlanetEntities(planet *entities.Planet) bool
	ForceRefresh()
	IsRemote() bool
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
	workforceOverlay  *WorkforceOverlay
	spriteRenderer    *rendering.SpriteRenderer
	buildingRenderer  *rendering.BuildingRenderer
	fleetUIManager    *fleetUIManager
	shipFleetRenderer *ShipFleetRenderer
	provider          PlanetDataProviderInterface
}

// NewPlanetView creates a new planet view
func NewPlanetView(ctx GameContext, buildMenu BuildMenuInterface, constructionQueue ConstructionQueueUIInterface, resourceStorage ResourceStorageUIInterface, shipyardUI ShipyardUIInterface, fleetInfoUI FleetInfoUIInterface, provider PlanetDataProviderInterface) *PlanetView {
	spriteRenderer := rendering.NewSpriteRenderer()
	pv := &PlanetView{
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
		shipFleetRenderer: NewShipFleetRenderer(ctx, spriteRenderer),
		provider:          provider,
	}
	pv.fleetUIManager = newFleetUIManager(pv)
	return pv
}

// SetPlanet sets the planet to display
func (pv *PlanetView) SetPlanet(planet *entities.Planet) {
	pv.planet = planet
	if pv.provider != nil && planet != nil {
		pv.provider.SetPlanetID(planet.GetID())
	}
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
	pv.fleetUIManager.updateFleets()
	pv.registerClickables()
}

// Update implements View interface
func (pv *PlanetView) Update() error {
	if pv.planet == nil {
		return nil
	}

	// Update planet data provider (once per frame)
	if pv.provider != nil {
		pv.provider.Update()

		// In remote mode, populate planet entities from server state
		if pv.provider.IsRemote() {
			if pv.provider.PopulatePlanetEntities(pv.planet) {
				pv.registerClickables()
			}
		}
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
	pv.fleetUIManager.updateFleets()

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
	pv.fleetUIManager.updateFleetInfoUI()

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
		if pv.fleetUIManager.closeFleetInfoUI() {
			return nil
		}
		// Close shipyard UI if open
		if pv.shipyardUI != nil && pv.shipyardUI.IsVisible() {
			pv.shipyardUI.Hide()
			return nil
		}
		// Go back to system view if a system is set, otherwise galaxy
		if sysView, ok := vm.GetView(ViewTypeSystem).(interface{ HasSystem() bool }); ok && sysView.HasSystem() {
			vm.SwitchTo(ViewTypeSystem)
		} else {
			vm.SwitchTo(ViewTypeGalaxy)
		}
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
		if pv.fleetUIManager.handleFleetClick(x, y) {
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
	pv.fleetUIManager.drawFleets(screen)

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

	// Draw UI info panel (scaled with utils.UIScale)
	details := formatPlanetDetails(pv.planet)
	cw := utils.CharWidth()
	lineH := int(15.0 * utils.UIScale)
	pad := cw
	margin := 10
	humanPlayer := pv.ctx.GetHumanPlayer()
	isOwned := humanPlayer != nil && pv.planet.Owner == humanPlayer.Name

	// Build building summary (needed for both width calc and drawing)
	buildingCounts := make(map[string]int)
	for _, be := range pv.planet.Buildings {
		if b, ok := be.(*entities.Building); ok {
			buildingCounts[b.BuildingType]++
		}
	}

	// Collect all text lines to auto-size the panel
	var allLines []string
	allLines = append(allLines, pv.planet.Name)
	subLine := ""
	if pv.system != nil {
		subLine = pv.system.Name
	}
	if pv.planet.Owner != "" {
		if subLine != "" {
			subLine += "  "
		}
		subLine += pv.planet.Owner
	}
	allLines = append(allLines, subLine)
	allLines = append(allLines, details...)
	if len(buildingCounts) > 0 {
		allLines = append(allLines, fmt.Sprintf("Buildings (%d):", len(pv.planet.Buildings)))
		for bType, count := range buildingCounts {
			label := "  " + bType
			if count > 1 {
				label = fmt.Sprintf("  %s x%d", bType, count)
			}
			allLines = append(allLines, label)
		}
	} else {
		allLines = append(allLines, "No buildings")
	}
	if len(pv.planet.Resources) > 0 {
		allLines = append(allLines, fmt.Sprintf("%d deposits", len(pv.planet.Resources)))
	}
	if isOwned {
		allLines = append(allLines, "[B]Build [W]Work [H]Help")
	} else {
		allLines = append(allLines, "[M]Market [H]Help")
	}

	// Find widest line for panel width
	maxChars := 20 // minimum width
	for _, line := range allLines {
		if len(line) > maxChars {
			maxChars = len(line)
		}
	}
	panelWidth := (maxChars+4)*cw + pad*2 // content + padding on both sides
	panelHeight := pad + len(allLines)*lineH + pad/2

	infoPanel := &UIPanel{
		X: margin, Y: margin, Width: panelWidth, Height: panelHeight,
		BgColor: utils.Theme.PanelBg, BorderColor: utils.Theme.PanelBorder,
	}
	infoPanel.Draw(screen)

	textX := margin + pad
	textY := margin + pad

	// Draw all lines with appropriate colors
	for i, line := range allLines {
		c := utils.Theme.TextDim
		if i == 0 {
			c = utils.Theme.Accent // planet name
		} else if strings.HasPrefix(line, "Population") || strings.HasPrefix(line, "Workforce") {
			c = utils.Theme.TextLight
		} else if strings.HasPrefix(line, "Tech:") {
			if strings.HasSuffix(line, "DECLINING") {
				c = utils.SystemOrange
			} else {
				c = utils.Theme.Accent
			}
		} else if strings.HasPrefix(line, "  No Electronics") {
			c = utils.SystemRed
		} else if strings.HasPrefix(line, "  Next:") {
			c = utils.Theme.TextDim
		} else if strings.HasPrefix(line, "  Tip:") {
			c = utils.SystemOrange
		} else if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "  [") {
			c = utils.Theme.TextLight // building names (indented)
		}
		DrawText(screen, line, textX, textY, c)
		textY += lineH
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
	pv.fleetUIManager.drawFleetInfoUI(screen)

	// Draw build menu if visible (on top of everything)
	if pv.buildMenu != nil {
		pv.buildMenu.Draw(screen)
	}
}

// OnEnter implements View interface
func (pv *PlanetView) OnEnter() {
	if pv.planet != nil {
		pv.updateResourcePositions()
		pv.fleetUIManager.updateFleets()
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

	// Ships and Fleets orbit further out than buildings and orbit faster
	if pv.system != nil {
		shipRadius := planetRadius + 120.0    // 3x scale (was 40.0)
		shipOrbitSpeed := pv.orbitOffset * 8.0 // Ships orbit 8x faster than surface
		planetOrbit := pv.planet.GetOrbitDistance()

		for _, entity := range pv.system.Entities {
			// Handle individual ships
			if ship, ok := entity.(*entities.Ship); ok {
				// Only show ships that are orbiting THIS specific planet
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

			// Handle fleets
			if fleet, ok := entity.(*entities.Fleet); ok {
				if fleet.LeadShip != nil {
					// Only show fleets that are orbiting THIS specific planet
					fleetOrbit := fleet.LeadShip.GetOrbitDistance()

					// Fleets must be at the EXACT same orbital distance as this planet
					if math.Abs(planetOrbit-fleetOrbit) < 1.0 {
						// Use the fleet's orbit angle relative to planet, with faster animation
						angle := fleet.LeadShip.GetOrbitAngle() - pv.planet.GetOrbitAngle() + shipOrbitSpeed

						// Position at ship orbit radius around this planet
						x := pv.centerX + shipRadius*math.Cos(angle)
						y := pv.centerY + shipRadius*math.Sin(angle)

						// Update absolute position for the fleet
						fleet.SetAbsolutePosition(x, y)
					}
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
	lh := int(15.0 * utils.UIScale)
	labelY := centerY + radius + lh
	DrawCenteredText(screen, resource.ResourceType, centerX, labelY)

	// Show abundance and extraction rate below label
	detailStr := fmt.Sprintf("%d abundance  %.1fx", resource.Abundance, resource.ExtractionRate)
	detailColor := utils.TextSecondary
	if resource.Abundance < 20 {
		detailColor = utils.SystemOrange
	}
	dWidth := len(detailStr) * utils.CharWidth()
	DrawText(screen, detailStr, centerX-dWidth/2, labelY+lh, detailColor)
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

	// Use the building renderer with attachment support
	pv.buildingRenderer.RenderBuildingWithAttachments(screen, building, centerX, centerY)

	// Draw building label with level
	lh := int(15.0 * utils.UIScale)
	labelY := centerY + building.Size + lh
	label := building.BuildingType
	if building.Level > 1 {
		label = fmt.Sprintf("%s L%d", building.BuildingType, building.Level)
	}
	DrawCenteredText(screen, label, centerX, labelY)

	// Show operational status or upgrade cost/tech requirement
	if !building.IsOperational {
		offWidth := len("OFFLINE") * utils.CharWidth()
		DrawText(screen, "OFFLINE", centerX-offWidth/2, labelY+lh, utils.SystemRed)
	} else if building.CanUpgrade() {
		upgradeTechReq := entities.GetUpgradeTechRequirement(building.Level)
		if upgradeTechReq > 0 && pv.planet != nil && pv.planet.TechLevel < upgradeTechReq {
			techStr := fmt.Sprintf("Tech %.1f", upgradeTechReq)
			techWidth := len(techStr) * utils.CharWidth()
			DrawText(screen, techStr, centerX-techWidth/2, labelY+lh, utils.SystemOrange)
		} else {
			costStr := fmt.Sprintf("↑%dcr", building.GetUpgradeCost())
			costWidth := len(costStr) * utils.CharWidth()
			DrawText(screen, costStr, centerX-costWidth/2, labelY+lh, utils.SystemGreen)
		}
	}
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

	var lines []string

	// Population
	pop := utils.FormatInt64WithCommas(planet.Population)
	cap := planet.GetTotalPopulationCapacity()
	if cap > 0 {
		lines = append(lines, fmt.Sprintf("Population: %s / %s", pop, utils.FormatInt64WithCommas(cap)))
	} else if planet.Population > 0 {
		lines = append(lines, fmt.Sprintf("Population: %s", pop))
	}

	// Happiness + productivity (compact)
	if planet.Population > 0 {
		happyPct := planet.Happiness * 100
		prodLabel := ""
		if planet.ProductivityBonus > 1.05 {
			prodLabel = fmt.Sprintf("  +%.0f%% prod", (planet.ProductivityBonus-1.0)*100)
		} else if planet.ProductivityBonus < 0.95 {
			prodLabel = fmt.Sprintf("  %.0f%% prod", (planet.ProductivityBonus-1.0)*100)
		}
		lines = append(lines, fmt.Sprintf("Happiness: %.0f%%%s", happyPct, prodLabel))
	}

	// Power (compact) — show whenever there are buildings that consume power
	if planet.PowerConsumed > 0 || planet.PowerGenerated > 0 {
		lines = append(lines, fmt.Sprintf("Power: %.0f/%.0f MW (%.0f%%)",
			planet.PowerGenerated, planet.PowerConsumed, planet.GetPowerRatio()*100))
	}

	// Tech level with era name
	if planet.TechLevel > 0.01 || planet.Population > 0 {
		era := entities.TechEraName(planet.TechLevel)
		techBonuses := ""
		if planet.TechLevel >= 0.1 {
			techBonuses = fmt.Sprintf("  +%.0f%%build +%.0f%%mine +%.0f%%cap",
				planet.TechLevel*5, planet.TechLevel*3, planet.TechLevel*10)
		}
		// Detect tech decline: no electronics + tech will decay
		declining := planet.TechLevel > 0.1 && planet.GetStoredAmount(entities.ResElectronics) == 0 && planet.Population > 500
		declineLabel := ""
		if declining {
			declineLabel = " DECLINING"
		}
		lines = append(lines, fmt.Sprintf("Tech: %.1f (%s)%s%s", planet.TechLevel, era, techBonuses, declineLabel))
		// Show next unlock hint or progression tip
		if nextName, nextReq := entities.NextTechUnlock(planet.TechLevel); nextName != "" {
			lines = append(lines, fmt.Sprintf("  Next: %s @ %.1f", nextName, nextReq))
		}
		// Hint when stuck at low tech with no electronics
		if planet.TechLevel < 0.5 && planet.GetStoredAmount(entities.ResElectronics) == 0 && planet.Population > 500 {
			lines = append(lines, "  Tip: Buy Electronics at market")
		} else if declining {
			lines = append(lines, "  No Electronics — tech decaying!")
		}
	}

	// Workforce
	if planet.WorkforceTotal > 0 {
		lines = append(lines, fmt.Sprintf("Workforce: %s / %s",
			utils.FormatInt64WithCommas(planet.WorkforceUsed),
			utils.FormatInt64WithCommas(planet.WorkforceTotal)))
	}

	return lines
}

func (pv *PlanetView) drawWorkforceToggleButton(screen *ebiten.Image) {
	rect := pv.workforceButtonRect()
	bgColor := utils.Theme.PanelBg
	borderColor := utils.Theme.PanelBorder
	if pv.workforceOverlay != nil && pv.workforceOverlay.Visible() {
		bgColor = utils.Theme.ButtonActive
		accentBorder := utils.Theme.Accent
		accentBorder.A = 180
		borderColor = accentBorder
	}
	panel := &UIPanel{
		X:           rect.Min.X,
		Y:           rect.Min.Y,
		Width:       rect.Dx(),
		Height:      rect.Dy(),
		BgColor:     bgColor,
		BorderColor: borderColor,
	}
	panel.Draw(screen)

	label := "Workforce [W]"
	textColor := utils.Theme.TextDim
	if pv.workforceOverlay != nil && pv.workforceOverlay.Visible() {
		textColor = utils.Theme.Accent
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
