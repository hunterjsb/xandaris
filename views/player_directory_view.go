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

type playerDirectoryEntry struct {
	player       *entities.Player
	isHuman      bool
	planets      int
	tradingPosts int
	totalStock   int
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
		BgColor:     color.RGBA{18, 24, 36, 240},
		BorderColor: utils.PanelBorder,
	}

	header := &UIPanel{
		X:           background.X + 20,
		Y:           background.Y + 20,
		Width:       background.Width - 40,
		Height:      80,
		BgColor:     color.RGBA{25, 32, 48, 245},
		BorderColor: utils.PanelBorder,
	}

	table := &UIPanel{
		X:           header.X,
		Y:           header.Y + header.Height + 10,
		Width:       header.Width,
		Height:      background.Height - header.Height - 100,
		BgColor:     color.RGBA{20, 28, 45, 235},
		BorderColor: utils.PanelBorder,
	}

	instructions := &UIPanel{
		X:           background.X + 20,
		Y:           background.Y + background.Height - 70,
		Width:       background.Width - 40,
		Height:      50,
		BgColor:     color.RGBA{22, 30, 50, 235},
		BorderColor: utils.PanelBorder,
	}

	return &PlayerDirectoryView{
		ctx:               ctx,
		returnTo:          ViewTypeGalaxy,
		entries:           make([]*playerDirectoryEntry, 0),
		scrollOffset:      0,
		maxVisibleRows:    14,
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
		mx, my := ebiten.CursorPosition()
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

	titleY := pd.headerPanel.Y + 20
	DrawTextCentered(screen, "Player Directory", pd.headerPanel.X+pd.headerPanel.Width/2, titleY, color.RGBA{235, 240, 255, 255}, 1.8)

	summary := fmt.Sprintf("Factions tracked: %d", len(pd.entries))
	DrawTextCentered(screen, summary, pd.headerPanel.X+pd.headerPanel.Width/2, titleY+40, utils.TextSecondary, 1.2)

	headerY := pd.tablePanel.Y + 18
	DrawText(screen, "Name", pd.tablePanel.X+20, headerY, utils.TextPrimary)
	DrawText(screen, "Type", pd.tablePanel.X+210, headerY, utils.TextPrimary)
	DrawText(screen, "Planets", pd.tablePanel.X+280, headerY, utils.TextPrimary)
	DrawText(screen, "Trading Posts", pd.tablePanel.X+360, headerY, utils.TextPrimary)
	DrawText(screen, "Stored Goods", pd.tablePanel.X+500, headerY, utils.TextPrimary)
	DrawText(screen, "Home System", pd.tablePanel.X+630, headerY, utils.TextPrimary)

	DrawLine(screen, pd.tablePanel.X+15, headerY+12, pd.tablePanel.X+pd.tablePanel.Width-15, headerY+12, utils.PanelBorder)

	if len(pd.entries) == 0 {
		DrawText(screen, "No known factions yet. Establish contact to populate this list.", pd.tablePanel.X+20, headerY+40, utils.TextSecondary)
	} else {
		contentTop := headerY + 20
		rowHeight := 26
		startIndex := pd.visibleStartIndex()
		endIndex := pd.visibleEndIndex()
		y := contentTop

		for _, entry := range pd.entries[startIndex:endIndex] {
			entry.bounds.X = pd.tablePanel.X + 15
			entry.bounds.Y = y - 8
			entry.bounds.W = pd.tablePanel.Width - 30
			entry.bounds.H = rowHeight

			nameColor := entry.player.Color
			DrawText(screen, entry.player.Name, pd.tablePanel.X+20, y, nameColor)
			playerType := "AI"
			if entry.isHuman {
				playerType = "Human"
			}
			DrawText(screen, playerType, pd.tablePanel.X+210, y, utils.TextSecondary)
			DrawText(screen, fmt.Sprintf("%d", entry.planets), pd.tablePanel.X+280, y, utils.TextPrimary)
			DrawText(screen, fmt.Sprintf("%d", entry.tradingPosts), pd.tablePanel.X+360, y, utils.TextPrimary)
			DrawText(screen, fmt.Sprintf("%d", entry.totalStock), pd.tablePanel.X+500, y, utils.TextPrimary)

			homeName := "Unknown"
			if entry.player.HomeSystem != nil {
				homeName = entry.player.HomeSystem.Name
			}
			DrawText(screen, homeName, pd.tablePanel.X+630, y, utils.TextSecondary)

			y += rowHeight
		}

		if len(pd.entries) > pd.maxVisibleRows {
			info := fmt.Sprintf("Showing %d-%d of %d", startIndex+1, endIndex, len(pd.entries))
			DrawText(screen, info, pd.tablePanel.X+20, pd.tablePanel.Y+pd.tablePanel.Height-25, utils.TextSecondary)
		}
	}

	instrY := pd.instructionsPanel.Y + 20
	DrawCenteredText(screen, "Click a faction to center its home system. Press [Esc] to return.", pd.instructionsPanel.X+pd.instructionsPanel.Width/2, instrY)
}

func (pd *PlayerDirectoryView) refreshEntries() {
	pd.entries = pd.entries[:0]

	for _, player := range pd.ctx.GetPlayers() {
		if player == nil {
			continue
		}

		entry := &playerDirectoryEntry{
			player:       player,
			isHuman:      player.IsHuman(),
			planets:      len(player.OwnedPlanets),
			tradingPosts: countTradingPosts(player.OwnedPlanets),
			totalStock:   totalStoredResources(player.OwnedPlanets),
		}
		pd.entries = append(pd.entries, entry)
	}

	sort.Slice(pd.entries, func(i, j int) bool {
		if pd.entries[i].isHuman == pd.entries[j].isHuman {
			return pd.entries[i].player.Name < pd.entries[j].player.Name
		}
		return pd.entries[i].isHuman
	})
}

func (pd *PlayerDirectoryView) focusPlayer(player *entities.Player) {
	if player == nil {
		return
	}

	if player.HomeSystem == nil {
		return
	}

	vm := pd.ctx.GetViewManager()
	if galaxyView, ok := vm.GetView(ViewTypeGalaxy).(*GalaxyView); ok {
		galaxyView.FocusSystem(player.HomeSystem)
		vm.SwitchTo(ViewTypeGalaxy)
	}
}

func (pd *PlayerDirectoryView) maxScroll() int {
	total := len(pd.entries) * 26
	visible := pd.tablePanel.Height - 60
	maxScroll := total - visible
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

func (pd *PlayerDirectoryView) visibleStartIndex() int {
	rowHeight := 26
	return pd.scrollOffset / rowHeight
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
