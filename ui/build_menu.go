package ui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/game"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/utils"
	"github.com/hunterjsb/xandaris/views"
)

var (
	buildMenuRectCache = utils.NewRectImageCache()
)

var (
	buildMenuItemHeight  = int(80.0 * utils.UIScale)
	buildMenuItemPadding = 5
	scrollStepPixels     = 40
)

// BuildMenuItem represents a single building option in the menu
type BuildMenuItem struct {
	BuildingType   string
	Name           string
	Description    string
	Cost           int
	TechRequired   float64 // minimum tech level to build
	AttachmentType string  // "Planet" or "Resource"
	Color          color.RGBA
	Bounds         struct {
		X, Y, Width, Height int
	}
}

// BuildMenu displays available buildings and handles construction
type BuildMenu struct {
	ctx               UIContext
	provider          *PlanetDataProvider
	isOpen            bool
	x                 int
	y                 int
	width             int
	height            int
	items             []*BuildMenuItem
	attachedTo        entities.Entity // The planet or resource we're building on
	attachmentID      string
	attachmentType    string
	selectedIndex     int
	scrollOffset      int
	notification      string
	notificationTimer int
}

// NewBuildMenu creates a new build menu
func NewBuildMenu(ctx UIContext, provider *PlanetDataProvider) *BuildMenu {
	return &BuildMenu{
		ctx:           ctx,
		provider:      provider,
		isOpen:        false,
		width:         int(400.0 * utils.UIScale),
		height:        int(550.0 * utils.UIScale),
		items:         make([]*BuildMenuItem, 0),
		selectedIndex: -1,
		scrollOffset:  0,
	}
}

// Open opens the build menu for a specific planet or resource
func (bm *BuildMenu) Open(attachedTo entities.Entity, x, y int) {
	bm.attachedTo = attachedTo
	bm.attachmentID = fmt.Sprintf("%d", attachedTo.GetID())
	bm.isOpen = true
	bm.selectedIndex = -1
	bm.scrollOffset = 0

	// Position menu near click location but keep on screen
	bm.x = x - bm.width/2
	bm.y = y - bm.height/2

	if bm.x < 10 {
		bm.x = 10
	}
	if bm.y < 10 {
		bm.y = 10
	}
	if bm.x+bm.width > views.ScreenWidth-10 {
		bm.x = views.ScreenWidth - bm.width - 10
	}
	if bm.y+bm.height > views.ScreenHeight-10 {
		bm.y = views.ScreenHeight - bm.height - 10
	}

	// Determine what we're building on
	if _, ok := attachedTo.(*entities.Planet); ok {
		bm.attachmentType = "Planet"
		bm.loadPlanetBuildings()
	} else if _, ok := attachedTo.(*entities.Resource); ok {
		bm.attachmentType = "Resource"
		bm.loadResourceBuildings()
	}
}

// Close closes the build menu
func (bm *BuildMenu) Close() {
	bm.isOpen = false
	bm.attachedTo = nil
	bm.items = make([]*BuildMenuItem, 0)
}

// IsOpen returns whether the menu is open
func (bm *BuildMenu) IsOpen() bool {
	return bm.isOpen
}

// loadPlanetBuildings populates menu with buildings that can be built on planets
func (bm *BuildMenu) loadPlanetBuildings() {
	bm.items = make([]*BuildMenuItem, 0)

	// Habitat
	bm.items = append(bm.items, &BuildMenuItem{
		BuildingType:   "Habitat",
		Name:           "Habitat Module",
		Description:    "Provides housing for 10M population",
		Cost:           800,
		TechRequired:   entities.GetTechRequirement("Habitat"),
		AttachmentType: "Planet",
		Color:          color.RGBA{100, 180, 220, 255},
	})

	// Generator (available early — key for power bootstrap)
	bm.items = append(bm.items, &BuildMenuItem{
		BuildingType:   "Generator",
		Name:           "Fuel Generator",
		Description:    "Burns Fuel → 50 MW power for buildings + life support",
		Cost:           1000,
		TechRequired:   entities.GetTechRequirement("Generator"),
		AttachmentType: "Planet",
		Color:          color.RGBA{255, 180, 50, 255},
	})

	// Trading Post
	bm.items = append(bm.items, &BuildMenuItem{
		BuildingType:   "Trading Post",
		Name:           "Trading Post",
		Description:    "Opens interstellar commerce and grants access to the market.",
		Cost:           1200,
		TechRequired:   entities.GetTechRequirement("Trading Post"),
		AttachmentType: "Planet",
		Color:          color.RGBA{210, 175, 95, 255},
	})

	// Refinery (Tech 0.5)
	bm.items = append(bm.items, &BuildMenuItem{
		BuildingType:   "Refinery",
		Name:           "Oil Refinery",
		Description:    "Converts Oil into Fuel (10 Oil/s → 5 Fuel/s)",
		Cost:           1500,
		TechRequired:   entities.GetTechRequirement("Refinery"),
		AttachmentType: "Planet",
		Color:          utils.StationRefinery,
	})

	// Factory (Tech 1.0)
	bm.items = append(bm.items, &BuildMenuItem{
		BuildingType:   "Factory",
		Name:           "Electronics Factory",
		Description:    "Converts Rare Metals + Iron into Electronics",
		Cost:           2000,
		TechRequired:   entities.GetTechRequirement("Factory"),
		AttachmentType: "Planet",
		Color:          color.RGBA{180, 130, 255, 255},
	})

	// Shipyard (Tech 1.0)
	bm.items = append(bm.items, &BuildMenuItem{
		BuildingType:   "Shipyard",
		Name:           "Orbital Shipyard",
		Description:    "Enables ship construction (+100% speed)",
		Cost:           2000,
		TechRequired:   entities.GetTechRequirement("Shipyard"),
		AttachmentType: "Planet",
		Color:          color.RGBA{150, 160, 180, 255},
	})

	// Fusion Reactor (Tech 2.0)
	bm.items = append(bm.items, &BuildMenuItem{
		BuildingType:   "Fusion Reactor",
		Name:           "Fusion Reactor",
		Description:    "Helium-3 fusion → 200 MW clean power",
		Cost:           3000,
		TechRequired:   entities.GetTechRequirement("Fusion Reactor"),
		AttachmentType: "Planet",
		Color:          color.RGBA{100, 220, 255, 255},
	})
}

// loadResourceBuildings populates menu with buildings that can be built on resources
func (bm *BuildMenu) loadResourceBuildings() {
	bm.items = make([]*BuildMenuItem, 0)

	// Mine
	description := "Increases resource extraction (+50%)"

	// Check if this resource node already has a mine or is building one
	if resource, ok := bm.attachedTo.(*entities.Resource); ok {
		resourceIDStr := fmt.Sprintf("%d", resource.GetID())

		// Check construction queue first
		if bm.provider.HasMineQueued(resourceIDStr) {
			description = "Mine already being built (1 per node max)"
		}

		// Check completed buildings
		planet := bm.findParentPlanet(resource)
		if planet != nil && bm.resourceHasMine(planet, resource.GetID()) {
			description = "This node already has a mine (1 per node max)"
		}
	}

	bm.items = append(bm.items, &BuildMenuItem{
		BuildingType:   "Mine",
		Name:           "Mining Complex",
		Description:    description,
		Cost:           500,
		AttachmentType: "Resource",
		Color:          color.RGBA{120, 110, 90, 255},
	})

	// Future: Add more resource buildings here
	// - Advanced Extractor
	// - Refinery
	// - Storage Depot
}

// Update handles input for the build menu
func (bm *BuildMenu) Update() {
	if !bm.isOpen {
		return
	}

	if len(bm.items) > 0 {
		_, wheelY := ebiten.Wheel()
		if wheelY != 0 {
			_, _, maxScroll := bm.computeScrollMetrics()
			if maxScroll > 0 {
				bm.scrollOffset -= int(wheelY * float64(scrollStepPixels))
				if bm.scrollOffset < 0 {
					bm.scrollOffset = 0
				}
				if bm.scrollOffset > maxScroll {
					bm.scrollOffset = maxScroll
				}
			} else {
				bm.scrollOffset = 0
			}
		}
	}

	if bm.notificationTimer > 0 {
		bm.notificationTimer--
	}

	// Close on Escape or right-click
	if bm.ctx.GetKeyBindings().IsActionJustPressed(views.ActionEscape) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		bm.Close()
		return
	}

	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()

		// Check if click is inside menu
		if mx >= bm.x && mx < bm.x+bm.width && my >= bm.y && my < bm.y+bm.height {
			bm.handleClick(mx, my)
		} else {
			// Click outside closes menu
			bm.Close()
		}
	}

	// Handle mouse hover
	mx, my := ebiten.CursorPosition()
	bm.updateHover(mx, my)
}

// handleClick processes a click within the menu
func (bm *BuildMenu) handleClick(mx, my int) {
	// Check if clicking on an item
	for i, item := range bm.items {
		if mx >= item.Bounds.X && mx < item.Bounds.X+item.Bounds.Width &&
			my >= item.Bounds.Y && my < item.Bounds.Y+item.Bounds.Height {
			bm.startConstruction(i)
			return
		}
	}
}

// updateHover updates the selected index based on mouse position
func (bm *BuildMenu) updateHover(mx, my int) {
	bm.selectedIndex = -1

	if !bm.isOpen {
		return
	}

	for i, item := range bm.items {
		if mx >= item.Bounds.X && mx < item.Bounds.X+item.Bounds.Width &&
			my >= item.Bounds.Y && my < item.Bounds.Y+item.Bounds.Height {
			bm.selectedIndex = i
			return
		}
	}
}

// findParentPlanet finds the planet that contains a given resource
func (bm *BuildMenu) findParentPlanet(resource *entities.Resource) *entities.Planet {
	// Search through all systems to find the planet with this resource
	for _, system := range bm.ctx.GetState().Systems {
		for _, entity := range system.Entities {
			if planet, ok := entity.(*entities.Planet); ok {
				for _, res := range planet.Resources {
					if res.GetID() == resource.GetID() {
						return planet
					}
				}
			}
		}
	}
	return nil
}

// resourceHasMine checks if a resource node already has a mine built on it
func (bm *BuildMenu) resourceHasMine(planet *entities.Planet, resourceID int) bool {
	for _, building := range planet.Buildings {
		if bldg, ok := building.(*entities.Building); ok {
			if bldg.BuildingType == "Mine" && bldg.ResourceNodeID == resourceID {
				return true
			}
		}
	}
	return false
}

// getPlanetTechLevel returns the tech level of the planet being built on.
func (bm *BuildMenu) getPlanetTechLevel() float64 {
	if planet, ok := bm.attachedTo.(*entities.Planet); ok {
		return planet.TechLevel
	}
	if resource, ok := bm.attachedTo.(*entities.Resource); ok {
		planet := bm.findParentPlanet(resource)
		if planet != nil {
			return planet.TechLevel
		}
	}
	return 0
}

// startConstruction queues a building for construction
func (bm *BuildMenu) startConstruction(itemIndex int) {
	if itemIndex < 0 || itemIndex >= len(bm.items) {
		return
	}

	item := bm.items[itemIndex]

	// Check tech requirement
	if item.TechRequired > 0 {
		techLevel := bm.getPlanetTechLevel()
		if techLevel < item.TechRequired {
			bm.notification = fmt.Sprintf("Requires Tech %.1f (have %.1f)", item.TechRequired, techLevel)
			bm.notificationTimer = 120
			return
		}
	}

	// Check if building a mine on a resource node that already has a mine or is being built
	if item.BuildingType == "Mine" && bm.attachmentType == "Resource" {
		if resource, ok := bm.attachedTo.(*entities.Resource); ok {
			resourceIDStr := fmt.Sprintf("%d", resource.GetID())

			// Check if there's already a mine in the construction queue for this resource
			if bm.provider.HasMineQueued(resourceIDStr) {
				bm.notification = "Mine already being built on this node"
				bm.notificationTimer = 120 // 2 seconds at 60fps
				return
			}

			// Find the parent planet to check existing completed mines
			planet := bm.findParentPlanet(resource)
			if planet != nil && bm.resourceHasMine(planet, resource.GetID()) {
				bm.notification = "This resource node already has a mine"
				bm.notificationTimer = 120 // 2 seconds at 60fps
				return
			}
		}
	}

	// Check if player has enough credits
	if bm.ctx.GetState().HumanPlayer.Credits < item.Cost {
		bm.notification = "Insufficient funds"
		bm.notificationTimer = 120 // 2 seconds at 60fps
		return
	}

	// In remote mode, send a build command instead of manipulating construction directly
	if bm.provider.IsRemote() {
		planetID := 0
		resourceID := 0
		if planet, ok := bm.attachedTo.(*entities.Planet); ok {
			planetID = planet.GetID()
		} else if resource, ok := bm.attachedTo.(*entities.Resource); ok {
			resourceID = resource.GetID()
			planet := bm.findParentPlanet(resource)
			if planet != nil {
				planetID = planet.GetID()
			}
		}
		bm.ctx.GetCommandChannel() <- game.GameCommand{
			Type: game.CmdBuild,
			Data: game.BuildCommandData{
				PlanetID:     planetID,
				BuildingType: item.BuildingType,
				ResourceID:   resourceID,
			},
		}
		bm.provider.ForceRefresh()
		bm.Close()
		return
	}

	// Deduct cost
	bm.ctx.GetState().HumanPlayer.Credits -= item.Cost

	// Create construction item
	constructionItem := &tickable.ConstructionItem{
		ID:             fmt.Sprintf("%s_%d_%d", bm.attachmentID, bm.ctx.GetTickManager().GetCurrentTick(), itemIndex),
		Type:           "Building",
		Name:           item.Name,
		Location:       bm.attachmentID,
		Owner:          bm.ctx.GetState().HumanPlayer.Name,
		Progress:       0,
		TotalTicks:     600, // 60 seconds at 1x speed
		RemainingTicks: 600,
		Cost:           item.Cost,
		Started:        bm.ctx.GetTickManager().GetCurrentTick(),
	}

	// Add to construction queue
	constructionSystem := tickable.GetSystemByName("Construction")
	if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
		cs.AddToQueue(bm.attachmentID, constructionItem)
	}

	// Close menu
	bm.Close()
}

// Draw renders the build menu
func (bm *BuildMenu) Draw(screen *ebiten.Image) {
	if !bm.isOpen {
		return
	}

	// Draw background panel (dark theme)
	panel := &views.UIPanel{
		X: bm.x, Y: bm.y, Width: bm.width, Height: bm.height,
		BgColor:     utils.Theme.PanelBgSolid,
		BorderColor: utils.Theme.PanelBorder,
	}
	panel.Draw(screen)

	lh := int(15.0 * utils.UIScale)

	// Draw title
	titleY := bm.y + lh
	views.DrawText(screen, "Build Menu", bm.x+10, titleY, utils.Theme.Accent)

	// Draw subtitle: tech level and credits
	subtitleY := titleY + lh
	techLvl := bm.getPlanetTechLevel()
	era := entities.TechEraName(techLvl)
	subtitle := fmt.Sprintf("Tech %.1f (%s)", techLvl, era)
	views.DrawText(screen, subtitle, bm.x+10, subtitleY, utils.Theme.Accent)

	// Draw player credits
	creditsY := subtitleY + lh
	creditsText := fmt.Sprintf("Credits: %d", bm.ctx.GetState().HumanPlayer.Credits)
	views.DrawText(screen, creditsText, bm.x+10, creditsY, utils.Theme.TextLight)

	// Draw separator line
	lineY := creditsY + lh
	views.DrawLine(screen, bm.x+10, lineY, bm.x+bm.width-10, lineY, utils.Theme.PanelBorder)

	// Draw building items (clipped to content area)
	contentTop := bm.getContentTop()
	contentBottom := bm.getContentBottom()
	itemY := contentTop - bm.scrollOffset

	_, _, maxScroll := bm.computeScrollMetrics()
	if bm.scrollOffset > maxScroll {
		bm.scrollOffset = maxScroll
	}

	// Create clipping region — items outside this area won't be visible
	clipRect := image.Rect(bm.x, contentTop, bm.x+bm.width, contentBottom)
	clipImg := screen.SubImage(clipRect).(*ebiten.Image)

	for i, item := range bm.items {
		itemX := bm.x + 10
		itemW := bm.width - 20

		item.Bounds.Width = 0
		item.Bounds.Height = 0

		// Skip items fully outside the visible area
		if itemY+buildMenuItemHeight < contentTop || itemY > contentBottom {
			itemY += buildMenuItemHeight + buildMenuItemPadding
			continue
		}

		item.Bounds.X = itemX
		item.Bounds.Y = itemY
		item.Bounds.Width = itemW
		item.Bounds.Height = buildMenuItemHeight

		// Check if this item can be built
		canBuild := true
		techLocked := false
		if item.TechRequired > 0 && bm.getPlanetTechLevel() < item.TechRequired {
			canBuild = false
			techLocked = true
		}
		if item.BuildingType == "Mine" && bm.attachmentType == "Resource" {
			if resource, ok := bm.attachedTo.(*entities.Resource); ok {
				resourceIDStr := fmt.Sprintf("%d", resource.GetID())

				// Check construction queue
				if bm.provider.HasMineQueued(resourceIDStr) {
					canBuild = false
				}

				// Check completed buildings
				if canBuild && !techLocked {
					planet := bm.findParentPlanet(resource)
					if planet != nil && bm.resourceHasMine(planet, resource.GetID()) {
						canBuild = false
					}
				}
			}
		}

		// Draw item background (highlight if selected, gray out if can't build)
		itemBg := utils.Theme.PanelBgLight
		if !canBuild {
			itemBg = color.RGBA{20, 20, 25, 230}
		} else if i == bm.selectedIndex {
			itemBg = utils.Theme.ButtonAccentBg
		}

		itemPanel := &views.UIPanel{
			X: itemX, Y: itemY, Width: itemW, Height: buildMenuItemHeight,
			BgColor: itemBg, BorderColor: utils.Theme.PanelBorder,
		}
		itemPanel.Draw(clipImg)

		// Draw building color indicator (dimmed if tech-locked)
		colorBoxSize := 12
		colorBoxX := itemX + 10
		colorBoxY := itemY + 10
		displayColor := item.Color
		if techLocked {
			displayColor = color.RGBA{item.Color.R / 3, item.Color.G / 3, item.Color.B / 3, 180}
		}
		colorBox := buildMenuRectCache.GetOrCreate(colorBoxSize, colorBoxSize, displayColor)
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(colorBoxX), float64(colorBoxY))
		clipImg.DrawImage(colorBox, opts)

		// Draw building name (dimmed if tech-locked)
		nameX := colorBoxX + colorBoxSize + 10
		nameY := itemY + lh - 4
		nameColor := utils.TextPrimary
		if techLocked {
			nameColor = utils.Theme.TextDim
		}
		views.DrawText(clipImg, item.Name, nameX, nameY, nameColor)

		// Draw cost or tech requirement
		costY := nameY + lh
		if techLocked {
			techText := fmt.Sprintf("Requires Tech %.1f", item.TechRequired)
			views.DrawText(clipImg, techText, nameX, costY, color.RGBA{200, 130, 60, 255})
		} else {
			costText := fmt.Sprintf("Cost: %d cr", item.Cost)
			costColor := utils.TextSecondary
			if bm.ctx.GetState().HumanPlayer.Credits < item.Cost {
				costColor = color.RGBA{200, 100, 100, 255}
			}
			views.DrawText(clipImg, costText, nameX, costY, costColor)
		}

		// Draw description (truncate to fit)
		desc := item.Description
		maxDescChars := (itemW - 40) / utils.CharWidth()
		if len(desc) > maxDescChars {
			desc = desc[:maxDescChars-2] + ".."
		}
		descY := costY + lh
		descColor := utils.TextSecondary
		if techLocked {
			descColor = utils.Theme.TextDim
		}
		views.DrawText(clipImg, desc, nameX, descY, descColor)

		// Draw build time or tech hint
		buildTimeY := descY + lh
		if techLocked {
			hint := "Buy/produce Electronics to advance"
			views.DrawText(clipImg, hint, nameX, buildTimeY, utils.Theme.TextDim)
		} else {
			// Estimate build time based on planet tech + happiness
			baseTicks := item.Cost / 2
			if baseTicks < 100 { baseTicks = 100 }
			techMult := 1.0 + bm.getPlanetTechLevel()*0.05
			prodMult := 1.0
			if planet, ok := bm.attachedTo.(*entities.Planet); ok && planet.ProductivityBonus > 0 {
				prodMult = planet.ProductivityBonus
			} else if resource, ok := bm.attachedTo.(*entities.Resource); ok {
				if p := bm.findParentPlanet(resource); p != nil && p.ProductivityBonus > 0 {
					prodMult = p.ProductivityBonus
				}
			}
			estTicks := float64(baseTicks) / (techMult * prodMult)
			estSec := estTicks / 10.0 // 10 ticks per second at 1x
			buildText := fmt.Sprintf("Build: %.0fs", estSec)
			views.DrawText(clipImg, buildText, nameX, buildTimeY, utils.TextSecondary)
		}

		itemY += buildMenuItemHeight + buildMenuItemPadding
	}

	// Draw scroll bar if content overflows
	if maxScroll > 0 {
		scrollAreaHeight := contentBottom - contentTop
		totalHeight := bm.totalContentHeight()
		if totalHeight <= 0 {
			totalHeight = scrollAreaHeight
		}
		scrollbarHeight := int(float64(scrollAreaHeight) * float64(scrollAreaHeight) / float64(totalHeight))
		if scrollbarHeight < 20 {
			scrollbarHeight = 20
		}
		scrollbarRange := scrollAreaHeight - scrollbarHeight
		scrollbarY := contentTop
		if scrollbarRange > 0 && maxScroll > 0 {
			scrollbarY = contentTop + int(float64(bm.scrollOffset)/float64(maxScroll)*float64(scrollbarRange))
		}

		scrollbarX := bm.x + bm.width - 14
		scrollbarColor := color.RGBA{120, 160, 210, 200}
		scrollbar := buildMenuRectCache.GetOrCreate(6, scrollbarHeight, scrollbarColor)

		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(scrollbarX), float64(scrollbarY))
		screen.DrawImage(scrollbar, opts)
	}

	// Draw notification message
	if bm.notificationTimer > 0 {
		notificationY := bm.y + bm.height - lh*2 - 4
		views.DrawText(screen, bm.notification, bm.x+10, notificationY, utils.SystemOrange)
	}

	// Draw instructions at bottom
	instructionsY := bm.y + bm.height - lh - 4
	instructionText := "Click to build | Esc cancel"
	if maxScroll > 0 {
		instructionText = "Scroll | Click build | Esc"
	}
	views.DrawText(screen, instructionText, bm.x+10, instructionsY, utils.Theme.TextDim)
}

func (bm *BuildMenu) totalContentHeight() int {
	count := len(bm.items)
	if count == 0 {
		return 0
	}
	return count*(buildMenuItemHeight+buildMenuItemPadding) - buildMenuItemPadding
}

func (bm *BuildMenu) computeScrollMetrics() (visibleHeight int, totalHeight int, maxScroll int) {
	top := bm.getContentTop()
	bottom := bm.getContentBottom()
	if bottom < top {
		bottom = top
	}
	visibleHeight = bottom - top
	totalHeight = bm.totalContentHeight()
	maxScroll = totalHeight - visibleHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	return
}

func (bm *BuildMenu) getContentTop() int {
	lh := int(15.0 * utils.UIScale)
	return bm.y + lh*4 + 10 // title + subtitle + credits + separator + gap
}

func (bm *BuildMenu) getContentBottom() int {
	lh := int(15.0 * utils.UIScale)
	return bm.y + bm.height - lh*2 - 10 // room for notification + instructions
}
