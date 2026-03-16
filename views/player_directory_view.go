package views

import (
	"fmt"
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/utils"
)

var pdRectCache = utils.NewRectImageCache()

type playerDirectoryEntry struct {
	player       *entities.Player
	isHuman      bool
	planets      int
	tradingPosts int
	totalStock   int
	mines        int
	buildings    int
	population   int64
	bounds       struct {
		X, Y, W, H int
	}
}

// PlayerDirectoryView displays all known factions with quick navigation helpers
type PlayerDirectoryView struct {
	ctx               GameContext
	returnTo          ViewType
	entries           []*playerDirectoryEntry
	scrollOffset      int
	maxVisibleRows    int
	hoveredRow        int
	backgroundPanel   *UIPanel
	headerPanel       *UIPanel
	tablePanel        *UIPanel
	instructionsPanel *UIPanel
}

// NewPlayerDirectoryView creates the directory UI
func NewPlayerDirectoryView(ctx GameContext) *PlayerDirectoryView {
	panelMargin := 60
	background := &UIPanel{
		X:           panelMargin,
		Y:           panelMargin,
		Width:       ScreenWidth - panelMargin*2,
		Height:      ScreenHeight - panelMargin*2,
		BgColor:     utils.Theme.PanelBgSolid,
		BorderColor: utils.Theme.PanelBorder,
	}

	header := &UIPanel{
		X:           background.X + 20,
		Y:           background.Y + 15,
		Width:       background.Width - 40,
		Height:      60,
		BgColor:     color.RGBA{18, 22, 42, 250},
		BorderColor: utils.Theme.PanelBorder,
	}

	table := &UIPanel{
		X:           header.X,
		Y:           header.Y + header.Height + 8,
		Width:       header.Width,
		Height:      background.Height - header.Height - 85,
		BgColor:     color.RGBA{14, 18, 34, 240},
		BorderColor: utils.Theme.PanelBorder,
	}

	instructions := &UIPanel{
		X:           background.X + 20,
		Y:           background.Y + background.Height - 50,
		Width:       background.Width - 40,
		Height:      35,
		BgColor:     color.RGBA{18, 22, 42, 240},
		BorderColor: utils.Theme.PanelBorder,
	}

	return &PlayerDirectoryView{
		ctx:               ctx,
		returnTo:          ViewTypeGalaxy,
		entries:           make([]*playerDirectoryEntry, 0),
		scrollOffset:      0,
		maxVisibleRows:    14,
		hoveredRow:        -1,
		backgroundPanel:   background,
		headerPanel:       header,
		tablePanel:        table,
		instructionsPanel: instructions,
	}
}

// GetType satisfies the View interface
func (pd *PlayerDirectoryView) GetType() ViewType {
	return ViewTypePlayers
}

// OnEnter refreshes data
func (pd *PlayerDirectoryView) OnEnter() {
	pd.scrollOffset = 0
	pd.refreshEntries()
}

// OnExit no-op
func (pd *PlayerDirectoryView) OnExit() {}

// Update handles input
func (pd *PlayerDirectoryView) Update() error {
	kb := pd.ctx.GetKeyBindings()
	if kb != nil && kb.IsActionJustPressed(ActionEscape) {
		pd.ctx.GetViewManager().SwitchTo(pd.returnTo)
		return nil
	}

	// Track hovered row
	mx, my := ebiten.CursorPosition()
	pd.hoveredRow = -1
	for i, entry := range pd.entries {
		if mx >= entry.bounds.X && mx <= entry.bounds.X+entry.bounds.W &&
			my >= entry.bounds.Y && my <= entry.bounds.Y+entry.bounds.H {
			pd.hoveredRow = i
			break
		}
	}

	if len(pd.entries) > 0 {
		_, wheelY := ebiten.Wheel()
		if wheelY != 0 {
			pd.scrollOffset -= int(wheelY * 30)
			if pd.scrollOffset < 0 {
				pd.scrollOffset = 0
			}
			if maxScroll := pd.maxScroll(); pd.scrollOffset > maxScroll {
				pd.scrollOffset = maxScroll
			}
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if mx >= pd.tablePanel.X && mx <= pd.tablePanel.X+pd.tablePanel.Width &&
			my >= pd.tablePanel.Y && my <= pd.tablePanel.Y+pd.tablePanel.Height {
			for _, entry := range pd.entries {
				if mx >= entry.bounds.X && mx <= entry.bounds.X+entry.bounds.W &&
					my >= entry.bounds.Y && my <= entry.bounds.Y+entry.bounds.H {
					pd.focusPlayer(entry.player)
					break
				}
			}
		}
	}

	return nil
}

// Draw renders the directory
func (pd *PlayerDirectoryView) Draw(screen *ebiten.Image) {
	pd.backgroundPanel.Draw(screen)
	pd.headerPanel.Draw(screen)
	pd.tablePanel.Draw(screen)
	pd.instructionsPanel.Draw(screen)

	// Refresh data every draw
	pd.refreshEntries()

	// Header
	titleY := pd.headerPanel.Y + 15
	DrawTextCentered(screen, "PLAYER DIRECTORY", pd.headerPanel.X+pd.headerPanel.Width/2, titleY, utils.Theme.Accent, 1.6)
	summary := fmt.Sprintf("%d factions", len(pd.entries))
	DrawTextCentered(screen, summary, pd.headerPanel.X+pd.headerPanel.Width/2, titleY+25, utils.Theme.TextDim, 1.0)

	// Column headers
	headerY := pd.tablePanel.Y + 16
	cols := pd.columnOffsets()

	DrawText(screen, "Faction", cols[0], headerY, utils.Theme.TextDim)
	DrawText(screen, "Credits", cols[1], headerY, utils.Theme.TextDim)
	DrawText(screen, "Pop", cols[2], headerY, utils.Theme.TextDim)
	DrawText(screen, "Planets", cols[3], headerY, utils.Theme.TextDim)
	DrawText(screen, "Mines", cols[4], headerY, utils.Theme.TextDim)
	DrawText(screen, "Ships", cols[5], headerY, utils.Theme.TextDim)
	DrawText(screen, "Stock", cols[6], headerY, utils.Theme.TextDim)

	DrawLine(screen, pd.tablePanel.X+12, headerY+14, pd.tablePanel.X+pd.tablePanel.Width-12, headerY+14, utils.Theme.PanelBorder)

	if len(pd.entries) == 0 {
		DrawTextCentered(screen, "No factions discovered.", pd.tablePanel.X+pd.tablePanel.Width/2, headerY+60, utils.Theme.TextDim, 1.0)
	} else {
		// Find max credits for wealth bar scaling
		maxCredits := 1
		for _, e := range pd.entries {
			if e.player.Credits > maxCredits {
				maxCredits = e.player.Credits
			}
		}

		contentTop := headerY + 22
		rowHeight := 30
		startIndex := pd.visibleStartIndex()
		endIndex := pd.visibleEndIndex()
		y := contentTop

		for idx, entry := range pd.entries[startIndex:endIndex] {
			globalIdx := startIndex + idx

			// Row background on hover
			if globalIdx == pd.hoveredRow {
				rowBg := pdRectCache.GetOrCreate(pd.tablePanel.Width-24, rowHeight, utils.Theme.PanelHover)
				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Translate(float64(pd.tablePanel.X+12), float64(y-6))
				screen.DrawImage(rowBg, opts)
			}

			entry.bounds.X = pd.tablePanel.X + 12
			entry.bounds.Y = y - 6
			entry.bounds.W = pd.tablePanel.Width - 24
			entry.bounds.H = rowHeight

			// Player color dot + name
			dotImg := utils.NewCircleImageCache().GetOrCreate(4, entry.player.Color)
			dotOpts := &ebiten.DrawImageOptions{}
			dotOpts.GeoM.Translate(float64(cols[0]-2), float64(y+2))
			screen.DrawImage(dotImg, dotOpts)

			nameColor := entry.player.Color
			label := entry.player.Name
			if entry.isHuman {
				label += " *"
			}
			DrawText(screen, truncate(label, 18), cols[0]+12, y, nameColor)

			// Credits with wealth bar
			credStr := formatNumber(entry.player.Credits)
			credColor := utils.Theme.TextLight
			if entry.player.Credits < 500 {
				credColor = color.RGBA{255, 100, 100, 255}
			} else if entry.player.Credits > 100000 {
				credColor = utils.Theme.Accent
			}
			DrawText(screen, credStr, cols[1], y, credColor)

			// Small wealth bar under credits
			barWidth := 70
			barHeight := 3
			barY := y + 12
			ratio := float64(entry.player.Credits) / float64(maxCredits)
			if ratio > 1 {
				ratio = 1
			}
			fillWidth := int(float64(barWidth) * ratio)
			if fillWidth < 1 {
				fillWidth = 1
			}
			barBgImg := pdRectCache.GetOrCreate(barWidth, barHeight, utils.Theme.BarBg)
			barBgOpts := &ebiten.DrawImageOptions{}
			barBgOpts.GeoM.Translate(float64(cols[1]), float64(barY))
			screen.DrawImage(barBgImg, barBgOpts)
			barFillImg := pdRectCache.GetOrCreate(fillWidth, barHeight, entry.player.Color)
			barFillOpts := &ebiten.DrawImageOptions{}
			barFillOpts.GeoM.Translate(float64(cols[1]), float64(barY))
			screen.DrawImage(barFillImg, barFillOpts)

			// Population
			popStr := formatPopulation(entry.population)
			DrawText(screen, popStr, cols[2], y, utils.Theme.TextLight)

			// Planets
			planetColor := utils.Theme.TextDim
			if entry.planets > 0 {
				planetColor = utils.Theme.TextLight
			}
			DrawText(screen, fmt.Sprintf("%d", entry.planets), cols[3], y, planetColor)

			// Mines
			mineColor := utils.Theme.TextDim
			if entry.mines > 0 {
				mineColor = utils.Theme.TextLight
			}
			DrawText(screen, fmt.Sprintf("%d", entry.mines), cols[4], y, mineColor)

			// Ships
			shipCount := len(entry.player.OwnedShips)
			if shipCount == 0 && entry.player.SyncedStock > 0 {
				// Remote player — we don't have local ship data
				shipCount = 0 // Can't determine from synced data yet
			}
			DrawText(screen, fmt.Sprintf("%d", shipCount), cols[5], y, utils.Theme.TextLight)

			// Stock
			stockColor := utils.Theme.TextDim
			if entry.totalStock > 0 {
				stockColor = utils.Theme.TextLight
			}
			DrawText(screen, formatNumber(entry.totalStock), cols[6], y, stockColor)

			y += rowHeight
		}

		// Scroll indicator
		if len(pd.entries) > pd.maxVisibleRows {
			info := fmt.Sprintf("%d-%d of %d", startIndex+1, endIndex, len(pd.entries))
			DrawText(screen, info, pd.tablePanel.X+pd.tablePanel.Width-120, pd.tablePanel.Y+pd.tablePanel.Height-20, utils.Theme.TextDim)
		}
	}

	// Instructions
	instrY := pd.instructionsPanel.Y + 12
	DrawTextCentered(screen, "Click faction to focus home system  |  Scroll to browse  |  Esc to close",
		pd.instructionsPanel.X+pd.instructionsPanel.Width/2, instrY, utils.Theme.TextDim, 0.9)
}

func (pd *PlayerDirectoryView) columnOffsets() []int {
	x := pd.tablePanel.X
	return []int{
		x + 20,  // Faction name
		x + 160, // Credits
		x + 250, // Population
		x + 320, // Planets
		x + 380, // Mines
		x + 430, // Ships
		x + 480, // Stock
	}
}

func (pd *PlayerDirectoryView) refreshEntries() {
	pd.entries = pd.entries[:0]

	for _, player := range pd.ctx.GetPlayers() {
		if player == nil {
			continue
		}

		mineCount := 0
		bldgCount := 0
		for _, planet := range player.OwnedPlanets {
			if planet == nil {
				continue
			}
			bldgCount += len(planet.Buildings)
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == "Mine" {
					mineCount++
				}
			}
		}

		pop := player.GetTotalPopulation()
		planets := len(player.OwnedPlanets)
		stock := totalStoredResources(player.OwnedPlanets)

		// Use synced data for remote factions
		if planets == 0 && player.SyncedPlanets > 0 {
			planets = player.SyncedPlanets
		}
		if mineCount == 0 && player.SyncedMines > 0 {
			mineCount = player.SyncedMines
		}
		if bldgCount == 0 && player.SyncedBuildings > 0 {
			bldgCount = player.SyncedBuildings
		}
		if stock == 0 && player.SyncedStock > 0 {
			stock = player.SyncedStock
		}

		entry := &playerDirectoryEntry{
			player:       player,
			isHuman:      player.IsHuman(),
			planets:      planets,
			tradingPosts: countTradingPosts(player.OwnedPlanets),
			totalStock:   stock,
			mines:        mineCount,
			buildings:    bldgCount,
			population:   pop,
		}
		pd.entries = append(pd.entries, entry)
	}

	// Sort by credits descending (richest first)
	sort.Slice(pd.entries, func(i, j int) bool {
		return pd.entries[i].player.Credits > pd.entries[j].player.Credits
	})
}

func (pd *PlayerDirectoryView) focusPlayer(player *entities.Player) {
	if player == nil || player.HomeSystem == nil {
		return
	}

	vm := pd.ctx.GetViewManager()
	if galaxyView, ok := vm.GetView(ViewTypeGalaxy).(*GalaxyView); ok {
		galaxyView.FocusSystem(player.HomeSystem)
		vm.SwitchTo(ViewTypeGalaxy)
	}
}

func (pd *PlayerDirectoryView) maxScroll() int {
	total := len(pd.entries) * 30
	visible := pd.tablePanel.Height - 60
	maxScroll := total - visible
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

func (pd *PlayerDirectoryView) visibleStartIndex() int {
	return pd.scrollOffset / 30
}

func (pd *PlayerDirectoryView) visibleEndIndex() int {
	start := pd.visibleStartIndex()
	end := start + pd.maxVisibleRows
	if end > len(pd.entries) {
		end = len(pd.entries)
	}
	return end
}

// SetReturnView configures the view to return to when closing
func (pd *PlayerDirectoryView) SetReturnView(view ViewType) {
	pd.returnTo = view
}

// GetReturnView exposes the target to return to
func (pd *PlayerDirectoryView) GetReturnView() ViewType {
	return pd.returnTo
}

// --- Formatting helpers ---

func formatNumber(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000.0)
	}
	if n >= 10000 {
		return fmt.Sprintf("%.0fk", float64(n)/1000.0)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000.0)
	}
	return fmt.Sprintf("%d", n)
}

func formatPopulation(pop int64) string {
	if pop >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(pop)/1000000.0)
	}
	if pop >= 10000 {
		return fmt.Sprintf("%.0fk", float64(pop)/1000.0)
	}
	if pop >= 1000 {
		return fmt.Sprintf("%.1fk", float64(pop)/1000.0)
	}
	return fmt.Sprintf("%d", pop)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-2] + ".."
}

