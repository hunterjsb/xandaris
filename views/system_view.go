package views

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/utils"
)

var (
	systemCircleCache   = utils.NewCircleImageCache()
	systemRectCache     = utils.NewRectImageCache()
	systemTriangleCache = utils.NewTriangleImageCache()
)

// SystemView represents the detailed view of a single system
type SystemView struct {
	ctx              GameContext
	system           *entities.System
	clickHandler     *ClickHandler
	centerX          float64
	centerY          float64
	scale            *ViewScale
	lastClickX       int
	lastClickY       int
	lastClickTime    int64
	orbitOffset      float64 // For animating orbits
	fleetInfoUI      FleetInfoUIInterface
	shipFleetRenderer *ShipFleetRenderer
}

// NewSystemView creates a new system view
func NewSystemView(ctx GameContext, fleetInfoUI FleetInfoUIInterface) *SystemView {
	// Use planet sprite renderer for ships/fleets in system view
	spriteRenderer := planetSpriteRenderer

	return &SystemView{
		ctx:              ctx,
		clickHandler:     NewClickHandler("system"),
		centerX:          float64(ScreenWidth) / 2,
		centerY:          float64(ScreenHeight) / 2,
		scale:            &SystemScale,
		fleetInfoUI:      fleetInfoUI,
		shipFleetRenderer: NewShipFleetRenderer(ctx, spriteRenderer),
	}
}

// SetSystem sets the system to display
func (sv *SystemView) SetSystem(system *entities.System) {
	sv.system = system

	// Calculate auto-scaling based on system size
	maxDistance := GetSystemMaxOrbitDistance(system)
	sv.scale = AutoScale(maxDistance, ScreenWidth, ScreenHeight)

	sv.updateEntityPositions()
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
			// Single click - check larger entities first (planets), then ships/fleets
			// This prevents ships from blocking planet clicks
			handled := sv.clickHandler.HandleClick(x, y)

			// Only check ships/fleets if we didn't click on a planet/star/etc
			if !handled && sv.shipFleetRenderer != nil && sv.fleetInfoUI != nil {
				ships, fleets := sv.shipFleetRenderer.GetShipsAndFleetsInSystem(sv.system)
				clickedShip, clickedFleet := sv.shipFleetRenderer.GetShipOrFleetAtPosition(ships, fleets, x, y, 8.0)

				if clickedFleet != nil {
					sv.fleetInfoUI.ShowFleet(clickedFleet)
					sv.clickHandler.ClearClickables() // Clear context menu
					sv.registerClickables()
				} else if clickedShip != nil {
					sv.fleetInfoUI.ShowShip(clickedShip)
					sv.clickHandler.ClearClickables() // Clear context menu
					sv.registerClickables()
				}
			}
		}

		sv.lastClickX = x
		sv.lastClickY = y
		sv.lastClickTime = currentTime
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
			int(selectedObj.GetClickRadius("system")),
			utils.Highlight)
	}

	// Draw context menu if active (but not if FleetInfoUI is showing)
	if sv.clickHandler.HasActiveMenu() && (sv.fleetInfoUI == nil || !sv.fleetInfoUI.IsVisible()) {
		sv.clickHandler.GetActiveMenu().Draw(screen)
	}

	// Draw fleet info UI if visible
	if sv.fleetInfoUI != nil && sv.fleetInfoUI.IsVisible() {
		sv.fleetInfoUI.Draw(screen)
	}

	// Draw UI info
	// System name + star type
	DrawText(screen, sv.system.Name, 10, 10, utils.Theme.Accent)

	// Compact resource summary for the system
	resSet := make(map[string]bool)
	planetCount := 0
	ownedCount := 0
	for _, e := range sv.system.Entities {
		if p, ok := e.(*entities.Planet); ok {
			planetCount++
			if p.Owner != "" {
				ownedCount++
			}
			for _, resEntity := range p.Resources {
				if res, ok := resEntity.(*entities.Resource); ok {
					resSet[res.ResourceType] = true
				}
			}
		}
	}
	summaryParts := fmt.Sprintf("%d planets", planetCount)
	if ownedCount > 0 {
		summaryParts += fmt.Sprintf("  %d owned", ownedCount)
	}
	if len(resSet) > 0 {
		resList := ""
		for r := range resSet {
			if resList != "" {
				resList += ", "
			}
			resList += r
		}
		summaryParts += "  |  " + resList
	}
	summaryX := 10 + len(sv.system.Name)*6 + 12
	DrawText(screen, summaryParts, summaryX, 10, utils.Theme.TextDim)

	DrawText(screen, "Double-click planet to enter  |  Esc to galaxy", 10, 28, utils.Theme.TextDim)

	if selected := sv.clickHandler.GetSelectedObject(); selected != nil {
		if planet, ok := selected.(*entities.Planet); ok {
			details := formatPlanetDetails(planet)
			panelHeight := 30 + len(details)*15

			// Add storage height
			storageCount := 0
			for _, s := range planet.StoredResources {
				if s != nil {
					storageCount++
				}
			}
			if storageCount > 0 {
				panelHeight += 20 + storageCount*14
			} else if len(planet.Resources) > 0 {
				// Deposits section for unowned planets
				panelHeight += 20 + len(planet.Resources)*14
			}

			infoPanel := NewUIPanel(6, 42, 240, panelHeight)
			infoPanel.BgColor = utils.Theme.PanelBg
			infoPanel.BorderColor = utils.Theme.PanelBorder
			infoPanel.Draw(screen)

			infoY := 52
			DrawText(screen, planet.Name, 14, infoY, utils.Theme.Accent)
			if planet.Owner != "" {
				ownerWidth := len(planet.Name)*6 + 10
				DrawText(screen, planet.Owner, 14+ownerWidth, infoY, utils.Theme.TextDim)
			}
			infoY += 18

			for _, line := range details {
				lineColor := utils.Theme.TextDim
				if strings.HasPrefix(line, "Population") || strings.HasPrefix(line, "Housing") || strings.HasPrefix(line, "Workforce") {
					lineColor = utils.Theme.TextLight
				}
				DrawText(screen, line, 14, infoY, lineColor)
				infoY += 15
			}

			// Storage with fill indicators (sorted for stable display)
			if storageCount > 0 {
				infoY += 6
				DrawText(screen, "Storage", 14, infoY, utils.Theme.TextDim)
				infoY += 15

				// Sort resource types for stable ordering
				sortedTypes := make([]string, 0, len(planet.StoredResources))
				for resType, s := range planet.StoredResources {
					if s != nil {
						sortedTypes = append(sortedTypes, resType)
					}
				}
				sort.Strings(sortedTypes)

				for _, resType := range sortedTypes {
					storage := planet.StoredResources[resType]
					resColor := utils.Theme.TextDim
					fillRatio := float64(0)
					if storage.Capacity > 0 {
						fillRatio = float64(storage.Amount) / float64(storage.Capacity)
					}
					if storage.Amount == 0 {
						resColor = color.RGBA{150, 60, 60, 255}
					} else if fillRatio > 0.8 {
						resColor = color.RGBA{100, 200, 130, 255}
					} else {
						resColor = utils.Theme.TextLight
					}

					label := fmt.Sprintf("  %s: %d", resType, storage.Amount)
					DrawText(screen, label, 14, infoY, resColor)

					// Small fill bar (use cached images to avoid per-frame allocation)
					barX := 170
					barW := 60
					barH := 3
					barBgImg := systemRectCache.GetOrCreate(barW, barH, utils.Theme.BarBg)
					barOpts := &ebiten.DrawImageOptions{}
					barOpts.GeoM.Translate(float64(barX), float64(infoY+4))
					screen.DrawImage(barBgImg, barOpts)
					if fillRatio > 0 {
						fillW := int(float64(barW) * fillRatio)
						if fillW < 1 {
							fillW = 1
						}
						barFillImg := systemRectCache.GetOrCreate(fillW, barH, resColor)
						fillOpts := &ebiten.DrawImageOptions{}
						fillOpts.GeoM.Translate(float64(barX), float64(infoY+4))
						screen.DrawImage(barFillImg, fillOpts)
					}

					infoY += 14
				}
			}

			// Show resource deposits (useful for scouting unowned planets)
			if len(planet.Resources) > 0 && storageCount == 0 {
				infoY += 6
				DrawText(screen, "Deposits", 14, infoY, utils.Theme.TextDim)
				infoY += 15
				for _, resEntity := range planet.Resources {
					if res, ok := resEntity.(*entities.Resource); ok {
						depColor := utils.Theme.TextLight
						if res.Abundance < 20 {
							depColor = utils.SystemOrange
						}
						label := fmt.Sprintf("  %s  %d abundance", res.ResourceType, res.Abundance)
						DrawText(screen, label, 14, infoY, depColor)
						infoY += 14
					}
				}
			}
		} else if provider, ok := selected.(ContextMenuProvider); ok {
			items := provider.GetContextMenuItems()
			infoPanel := NewUIPanel(6, 42, 240, 20+len(items)*15)
			infoPanel.BgColor = utils.Theme.PanelBg
			infoPanel.BorderColor = utils.Theme.PanelBorder
			infoPanel.Draw(screen)

			infoY := 52
			DrawText(screen, provider.GetContextMenuTitle(), 14, infoY, utils.Theme.Accent)
			infoY += 18
			for _, line := range items {
				if strings.TrimSpace(line) == "" {
					continue
				}
				DrawText(screen, line, 14, infoY, utils.Theme.TextDim)
				infoY += 15
			}
		}
	}
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

// HasSystem returns whether a system is currently set for viewing.
func (sv *SystemView) HasSystem() bool {
	return sv.system != nil
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

// registerClickables adds all entities as clickable objects (except ships/fleets)
func (sv *SystemView) registerClickables() {
	sv.clickHandler.ClearClickables()

	if sv.system == nil {
		return
	}

	for _, entity := range sv.system.Entities {
		// Skip ships and fleets - they're handled separately by shipFleetRenderer
		if _, isShip := entity.(*entities.Ship); isShip {
			continue
		}
		if _, isFleet := entity.(*entities.Fleet); isFleet {
			continue
		}

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

	// Get cached star image
	starImg := systemCircleCache.GetOrCreate(radius, star.Color)

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
	if sv.system == nil || sv.shipFleetRenderer == nil {
		return
	}

	// Get ships and fleets from the system
	ships, fleets := sv.shipFleetRenderer.GetShipsAndFleetsInSystem(sv.system)

	// Draw them using centralized renderer
	sv.shipFleetRenderer.DrawShipsAndFleets(screen, ships, fleets, 6)
}

// drawPlanet renders a single planet
func (sv *SystemView) drawPlanet(screen *ebiten.Image, planet *entities.Planet) {
	x, y := planet.GetAbsolutePosition()
	centerX := int(x)
	centerY := int(y)
	// Keep planet size consistent regardless of orbital scale
	radius := planet.Size

	if planet.Owner != "" {
		if ownerColor, ok := sv.getOwnerColor(planet.Owner); ok {
			DrawOwnershipRing(screen, centerX, centerY, float64(radius+3), ownerColor)
		}
	}

	// Get cached planet image
	planetImg := systemCircleCache.GetOrCreate(radius, planet.Color)

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

	// Get cached station image (square/diamond)
	stationImg := systemRectCache.GetOrCreate(size*2, size*2, station.Color)

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

func (sv *SystemView) getOwnerColor(owner string) (color.RGBA, bool) {
	if owner == "" {
		return color.RGBA{}, false
	}

	players := sv.ctx.GetPlayers()
	for _, player := range players {
		if player != nil && player.Name == owner {
			return player.Color, true
		}
	}

	return color.RGBA{}, false
}
