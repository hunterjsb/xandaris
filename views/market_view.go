package views

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/utils"
)

type marketContribution struct {
	planet  string
	amount  int
	owner   string
	isHuman bool
}

type marketResourceRow struct {
	resource      string
	totalAmount   int
	contributions []marketContribution
}

// MarketView presents a consolidated look at trade goods across Trading Posts
type MarketView struct {
	ctx               GameContext
	rows              []marketResourceRow
	scrollOffset      int
	maxVisibleRows    int
	tradingPostCount  int
	humanTradingPosts int
	aiTradingPosts    int
	returnTo          ViewType
	backgroundPanel   *UIPanel
	headerPanel       *UIPanel
	tablePanel        *UIPanel
	instructionsPanel *UIPanel
}

// NewMarketView creates a new market view instance
func NewMarketView(ctx GameContext) *MarketView {
	panelMargin := 60
	background := &UIPanel{
		X:           panelMargin,
		Y:           panelMargin,
		Width:       ScreenWidth - panelMargin*2,
		Height:      ScreenHeight - panelMargin*2,
		BgColor:     color.RGBA{15, 20, 30, 235},
		BorderColor: utils.PanelBorder,
	}

	header := &UIPanel{
		X:           background.X + 20,
		Y:           background.Y + 20,
		Width:       background.Width - 40,
		Height:      80,
		BgColor:     color.RGBA{25, 35, 55, 245},
		BorderColor: utils.PanelBorder,
	}

	table := &UIPanel{
		X:           header.X,
		Y:           header.Y + header.Height + 10,
		Width:       header.Width,
		Height:      background.Height - header.Height - 100,
		BgColor:     color.RGBA{18, 28, 45, 230},
		BorderColor: utils.PanelBorder,
	}

	instructions := &UIPanel{
		X:           background.X + 20,
		Y:           background.Y + background.Height - 70,
		Width:       background.Width - 40,
		Height:      50,
		BgColor:     color.RGBA{20, 30, 48, 235},
		BorderColor: utils.PanelBorder,
	}

	return &MarketView{
		ctx:               ctx,
		rows:              make([]marketResourceRow, 0),
		scrollOffset:      0,
		maxVisibleRows:    12,
		tradingPostCount:  0,
		humanTradingPosts: 0,
		aiTradingPosts:    0,
		returnTo:          ViewTypeGalaxy,
		backgroundPanel:   background,
		headerPanel:       header,
		tablePanel:        table,
		instructionsPanel: instructions,
	}
}

// GetType returns the view type
func (mv *MarketView) GetType() ViewType {
	return ViewTypeMarket
}

// OnEnter resets scrolling and refreshes cached data
func (mv *MarketView) OnEnter() {
	mv.scrollOffset = 0
	mv.refreshData()
}

// OnExit is part of the view interface (no-op for now)
func (mv *MarketView) OnExit() {}

// Update refreshes market data and handles scrolling
func (mv *MarketView) Update() error {
	mv.refreshData()

	kb := mv.ctx.GetKeyBindings()
	if kb != nil && kb.IsActionJustPressed(ActionEscape) {
		mv.ctx.GetViewManager().SwitchTo(mv.returnTo)
		return nil
	}

	_, wheelY := ebiten.Wheel()
	if wheelY != 0 {
		mv.scrollOffset -= int(wheelY)
		if mv.scrollOffset < 0 {
			mv.scrollOffset = 0
		}
		maxScroll := len(mv.rows) - mv.maxVisibleRows
		if maxScroll < 0 {
			maxScroll = 0
		}
		if mv.scrollOffset > maxScroll {
			mv.scrollOffset = maxScroll
		}
	}

	return nil
}

// Draw renders the market overview
func (mv *MarketView) Draw(screen *ebiten.Image) {
	// Panels
	mv.backgroundPanel.Draw(screen)
	mv.headerPanel.Draw(screen)
	mv.tablePanel.Draw(screen)
	mv.instructionsPanel.Draw(screen)

	// Header content
	titleY := mv.headerPanel.Y + 22
	DrawTextCentered(screen, "Market Overview", mv.headerPanel.X+mv.headerPanel.Width/2, titleY, color.RGBA{225, 230, 255, 255}, 1.8)

	subtitle := fmt.Sprintf("Trading Posts Online: %d (You: %d | NPC: %d)", mv.tradingPostCount, mv.humanTradingPosts, mv.aiTradingPosts)
	DrawTextCentered(screen, subtitle, mv.headerPanel.X+mv.headerPanel.Width/2, titleY+40, utils.TextSecondary, 1.2)

	// Table header
	headerY := mv.tablePanel.Y + 20
	leftX := mv.tablePanel.X + 20
	rightX := mv.tablePanel.X + mv.tablePanel.Width - 20

	DrawText(screen, "Resource", leftX, headerY, utils.TextPrimary)
	DrawText(screen, "Total Stock", mv.tablePanel.X+220, headerY, utils.TextPrimary)
	DrawText(screen, "Supply by Trading Post", mv.tablePanel.X+360, headerY, utils.TextPrimary)

	// Separator line
	DrawLine(screen, mv.tablePanel.X+15, headerY+12, rightX-5, headerY+12, utils.PanelBorder)

	rowStartY := headerY + 30
	rowHeight := 26

	if mv.tradingPostCount == 0 {
		DrawText(screen, "No Trading Posts constructed. Build one on a colony to access the market.",
			mv.tablePanel.X+20, rowStartY, utils.TextSecondary)
		return
	}

	if len(mv.rows) == 0 {
		DrawText(screen, "No tradable inventory yet. When Trading Posts store resources, they will appear here.",
			mv.tablePanel.X+20, rowStartY, utils.TextSecondary)
		return
	}

	endIndex := mv.scrollOffset + mv.maxVisibleRows
	if endIndex > len(mv.rows) {
		endIndex = len(mv.rows)
	}

	y := rowStartY
	for _, row := range mv.rows[mv.scrollOffset:endIndex] {
		DrawText(screen, row.resource, leftX, y, utils.TextPrimary)
		DrawText(screen, fmt.Sprintf("%d", row.totalAmount), mv.tablePanel.X+220, y, utils.TextPrimary)

		var builder strings.Builder
		for i, contribution := range row.contributions {
			if i > 0 {
				builder.WriteString(", ")
			}
			ownerLabel := contribution.owner
			if contribution.isHuman {
				ownerLabel = "You"
			}
			builder.WriteString(fmt.Sprintf("%s (%s: %d)", contribution.planet, ownerLabel, contribution.amount))
		}
		DrawText(screen, builder.String(), mv.tablePanel.X+360, y, utils.TextSecondary)

		y += rowHeight
	}

	// Scroll indicator
	if len(mv.rows) > mv.maxVisibleRows {
		scrollText := fmt.Sprintf("Showing %d-%d of %d entries",
			mv.scrollOffset+1, mv.scrollOffset+endIndex-mv.scrollOffset, len(mv.rows))
		DrawText(screen, scrollText, mv.tablePanel.X+20, mv.tablePanel.Y+mv.tablePanel.Height-30, utils.TextSecondary)
	}

	// Instructions
	instrY := mv.instructionsPanel.Y + 20
	DrawCenteredText(screen, "Use the mouse wheel to scroll, [Esc] to return. Trading Posts automatically list stored resources for all factions.",
		mv.instructionsPanel.X+mv.instructionsPanel.Width/2, instrY)
}

func (mv *MarketView) refreshData() {
	mv.rows = mv.rows[:0]
	mv.tradingPostCount = 0
	mv.humanTradingPosts = 0
	mv.aiTradingPosts = 0

	resourceMap := make(map[string]*marketResourceRow)

	players := mv.ctx.GetPlayers()
	if len(players) == 0 {
		return
	}

	for _, player := range players {
		if player == nil {
			continue
		}
		for _, planet := range player.OwnedPlanets {
			if planet == nil || planet.Owner != player.Name {
				continue
			}
			if !mv.planetHasTradingPost(planet) {
				continue
			}

			mv.tradingPostCount++
			if player.IsHuman() {
				mv.humanTradingPosts++
			} else {
				mv.aiTradingPosts++
			}

			for resourceType, storage := range planet.StoredResources {
				if storage == nil || storage.Amount <= 0 {
					continue
				}

				row, exists := resourceMap[resourceType]
				if !exists {
					row = &marketResourceRow{
						resource:      resourceType,
						totalAmount:   0,
						contributions: make([]marketContribution, 0),
					}
					resourceMap[resourceType] = row
				}

				row.totalAmount += storage.Amount
				row.contributions = append(row.contributions, marketContribution{
					planet:  planet.Name,
					owner:   player.Name,
					amount:  storage.Amount,
					isHuman: player.IsHuman(),
				})
			}
		}
	}

	for _, row := range resourceMap {
		sort.Slice(row.contributions, func(i, j int) bool {
			if row.contributions[i].amount == row.contributions[j].amount {
				if row.contributions[i].owner == row.contributions[j].owner {
					return row.contributions[i].planet < row.contributions[j].planet
				}
				return row.contributions[i].owner < row.contributions[j].owner
			}
			return row.contributions[i].amount > row.contributions[j].amount
		})
		mv.rows = append(mv.rows, *row)
	}

	sort.Slice(mv.rows, func(i, j int) bool {
		if mv.rows[i].totalAmount == mv.rows[j].totalAmount {
			return mv.rows[i].resource < mv.rows[j].resource
		}
		return mv.rows[i].totalAmount > mv.rows[j].totalAmount
	})
}

func (mv *MarketView) planetHasTradingPost(planet *entities.Planet) bool {
	for _, buildingEntity := range planet.Buildings {
		if building, ok := buildingEntity.(*entities.Building); ok {
			if building.BuildingType == "Trading Post" && building.IsOperational {
				return true
			}
		}
	}
	return false
}

// SetReturnView configures which view to go back to when exiting the market
func (mv *MarketView) SetReturnView(view ViewType) {
	mv.returnTo = view
}

// GetReturnView exposes which view the market should return to
func (mv *MarketView) GetReturnView() ViewType {
	return mv.returnTo
}
