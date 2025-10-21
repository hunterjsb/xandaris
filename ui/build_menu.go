package ui

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/tickable"
	"github.com/hunterjsb/xandaris/utils"
	"github.com/hunterjsb/xandaris/views"
)

var (
	buildMenuRectCache = utils.NewRectImageCache()
)

const (
	buildMenuItemHeight  = 80
	buildMenuItemPadding = 5
	scrollStepPixels     = 40
)

// BuildMenuItem represents a single building option in the menu
type BuildMenuItem struct {
	BuildingType   string
	Name           string
	Description    string
	Cost           int
	AttachmentType string // "Planet" or "Resource"
	Color          color.RGBA
	Bounds         struct {
		X, Y, Width, Height int
	}
}

// BuildMenu displays available buildings and handles construction
type BuildMenu struct {
	ctx               UIContext
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
func NewBuildMenu(ctx UIContext) *BuildMenu {
	return &BuildMenu{
		ctx:           ctx,
		isOpen:        false,
		width:         400,
		height:        500,
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
	if bm.x+bm.width > 1280-10 {
		bm.x = 1280 - bm.width - 10
	}
	if bm.y+bm.height > 720-10 {
		bm.y = 720 - bm.height - 10
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
		AttachmentType: "Planet",
		Color:          color.RGBA{100, 180, 220, 255},
	})

	// Shipyard
	bm.items = append(bm.items, &BuildMenuItem{
		BuildingType:   "Shipyard",
		Name:           "Orbital Shipyard",
		Description:    "Enables ship construction (+100% speed)",
		Cost:           2000,
		AttachmentType: "Planet",
		Color:          color.RGBA{150, 160, 180, 255},
	})

	// Refinery
	bm.items = append(bm.items, &BuildMenuItem{
		BuildingType:   "Refinery",
		Name:           "Oil Refinery",
		Description:    "Converts Oil into Fuel (10 Oil/s → 5 Fuel/s)",
		Cost:           1500,
		AttachmentType: "Planet",
		Color:          utils.StationRefinery, // Orange color
	})

	// Trading Post
	bm.items = append(bm.items, &BuildMenuItem{
		BuildingType:   "Trading Post",
		Name:           "Trading Post",
		Description:    "Opens interstellar commerce and grants access to the market.",
		Cost:           1200,
		AttachmentType: "Planet",
		Color:          color.RGBA{210, 175, 95, 255},
	})

	// Future: Add more planet buildings here
	// - Research Lab
	// - Barracks
	// - Trade Port
	// - Defense Grid
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
		constructionSystem := tickable.GetSystemByName("Construction")
		if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
			if cs.HasMineInQueue(resourceIDStr) {
				description = "Mine already being built (1 per node max)"
			}
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
				bm.scrollOffset -= int(wheelY * scrollStepPixels)
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

// startConstruction queues a building for construction
func (bm *BuildMenu) startConstruction(itemIndex int) {
	if itemIndex < 0 || itemIndex >= len(bm.items) {
		return
	}

	item := bm.items[itemIndex]

	// Check if building a mine on a resource node that already has a mine or is being built
	if item.BuildingType == "Mine" && bm.attachmentType == "Resource" {
		if resource, ok := bm.attachedTo.(*entities.Resource); ok {
			resourceIDStr := fmt.Sprintf("%d", resource.GetID())

			// Check if there's already a mine in the construction queue for this resource
			constructionSystem := tickable.GetSystemByName("Construction")
			if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
				if cs.HasMineInQueue(resourceIDStr) {
					bm.notification = "Mine already being built on this node"
					bm.notificationTimer = 120 // 2 seconds at 60fps
					return
				}
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

	// Draw background panel
	panel := views.NewUIPanel(bm.x, bm.y, bm.width, bm.height)
	panel.Draw(screen)

	// Draw title
	titleY := bm.y + 15
	views.DrawCenteredText(screen, "Build Menu", bm.x+bm.width/2, titleY)

	// Draw subtitle based on attachment type
	subtitleY := titleY + 20
	subtitle := fmt.Sprintf("Building on: %s", bm.attachmentType)
	views.DrawCenteredText(screen, subtitle, bm.x+bm.width/2, subtitleY)

	// Draw player credits
	creditsY := subtitleY + 20
	creditsText := fmt.Sprintf("Credits: %d", bm.ctx.GetState().HumanPlayer.Credits)
	views.DrawCenteredText(screen, creditsText, bm.x+bm.width/2, creditsY)

	// Draw separator line
	lineY := creditsY + 15
	views.DrawLine(screen, bm.x+10, lineY, bm.x+bm.width-10, lineY, utils.PanelBorder)

	// Draw building items
	contentTop := bm.getContentTop()
	contentBottom := bm.getContentBottom()
	itemY := contentTop - bm.scrollOffset

	_, _, maxScroll := bm.computeScrollMetrics()
	if bm.scrollOffset > maxScroll {
		bm.scrollOffset = maxScroll
	}

	for i, item := range bm.items {
		itemX := bm.x + 10
		itemW := bm.width - 20

		item.Bounds.Width = 0
		item.Bounds.Height = 0

		if itemY+buildMenuItemHeight < contentTop {
			itemY += buildMenuItemHeight + buildMenuItemPadding
			continue
		}
		if itemY > contentBottom {
			itemY += buildMenuItemHeight + buildMenuItemPadding
			continue
		}

		item.Bounds.X = itemX
		item.Bounds.Y = itemY
		item.Bounds.Width = itemW
		item.Bounds.Height = buildMenuItemHeight

		// Check if this item can be built
		canBuild := true
		if item.BuildingType == "Mine" && bm.attachmentType == "Resource" {
			if resource, ok := bm.attachedTo.(*entities.Resource); ok {
				resourceIDStr := fmt.Sprintf("%d", resource.GetID())

				// Check construction queue
				constructionSystem := tickable.GetSystemByName("Construction")
				if cs, ok := constructionSystem.(*tickable.ConstructionSystem); ok {
					if cs.HasMineInQueue(resourceIDStr) {
						canBuild = false
					}
				}

				// Check completed buildings
				if canBuild {
					planet := bm.findParentPlanet(resource)
					if planet != nil && bm.resourceHasMine(planet, resource.GetID()) {
						canBuild = false
					}
				}
			}
		}

		// Draw item background (highlight if selected, gray out if can't build)
		itemBg := utils.PanelBg
		if !canBuild {
			itemBg = color.RGBA{30, 30, 30, 230} // Dark gray for disabled
		} else if i == bm.selectedIndex {
			itemBg = color.RGBA{40, 40, 80, 230}
		}

		itemPanel := views.NewUIPanel(itemX, itemY, itemW, buildMenuItemHeight)
		itemPanel.BgColor = itemBg
		itemPanel.Draw(screen)

		// Draw building color indicator
		colorBoxSize := 12
		colorBoxX := itemX + 10
		colorBoxY := itemY + 10
		colorBox := buildMenuRectCache.GetOrCreate(colorBoxSize, colorBoxSize, item.Color)
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(colorBoxX), float64(colorBoxY))
		screen.DrawImage(colorBox, opts)

		// Draw building name
		nameX := colorBoxX + colorBoxSize + 10
		nameY := itemY + 12
		views.DrawText(screen, item.Name, nameX, nameY, utils.TextPrimary)

		// Draw cost
		costText := fmt.Sprintf("Cost: %d credits", item.Cost)
		costY := nameY + 15
		costColor := utils.TextSecondary
		if bm.ctx.GetState().HumanPlayer.Credits < item.Cost {
			costColor = color.RGBA{200, 100, 100, 255} // Red if can't afford
		}
		views.DrawText(screen, costText, nameX, costY, costColor)

		// Draw description
		descY := costY + 15
		views.DrawText(screen, item.Description, nameX, descY, utils.TextSecondary)

		// Draw build time
		buildTimeText := "Build time: 60s"
		buildTimeY := descY + 15
		views.DrawText(screen, buildTimeText, nameX, buildTimeY, utils.TextSecondary)

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
		notificationY := bm.y + bm.height - 45
		views.DrawCenteredText(screen, bm.notification, bm.x+bm.width/2, notificationY)
	}

	// Draw instructions at bottom
	instructionsY := bm.y + bm.height - 25
	instructionText := "Click to build  |  ESC to cancel"
	if maxScroll > 0 {
		instructionText = "Scroll to browse  |  Click to build  |  ESC to cancel"
	}
	views.DrawCenteredText(screen, instructionText, bm.x+bm.width/2, instructionsY)
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
	return bm.y + 80
}

func (bm *BuildMenu) getContentBottom() int {
	return bm.y + bm.height - 60
}
