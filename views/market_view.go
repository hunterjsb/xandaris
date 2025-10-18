package views

import (
	"fmt"
	"image"
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/utils"
)

type marketResourceRow struct {
	resource   string
	humanStock int
	npcStock   int
	avgValue   float64
}

// MarketView presents a simplified trading interface
type MarketView struct {
	ctx               GameContext
	rows              []marketResourceRow
	scrollOffset      int
	maxVisibleRows    int
	returnTo          ViewType
	backgroundPanel   *UIPanel
	headerPanel       *UIPanel
	tablePanel        *UIPanel
	instructionsPanel *UIPanel
	rowHits           map[string]tableRowHit
	humanCredits      int
}

type tableRowHit struct {
	buy  image.Rectangle
	sell image.Rectangle
}

const tradeLot = 10

// NewMarketView creates a new market view instance
func NewMarketView(ctx GameContext) *MarketView {
	panelMargin := 60
	background := &UIPanel{
		X:           panelMargin,
		Y:           panelMargin,
		Width:       ScreenWidth - panelMargin*2,
		Height:      ScreenHeight - panelMargin*2,
		BgColor:     color.RGBA{18, 20, 32, 235},
		BorderColor: utils.PanelBorder,
	}

	header := &UIPanel{
		X:           background.X + 20,
		Y:           background.Y + 20,
		Width:       background.Width - 40,
		Height:      70,
		BgColor:     color.RGBA{28, 38, 58, 245},
		BorderColor: utils.PanelBorder,
	}

	table := &UIPanel{
		X:           header.X,
		Y:           header.Y + header.Height + 10,
		Width:       header.Width,
		Height:      background.Height - header.Height - 120,
		BgColor:     color.RGBA{22, 30, 48, 235},
		BorderColor: utils.PanelBorder,
	}

	instructions := &UIPanel{
		X:           background.X + 20,
		Y:           table.Y + table.Height + 10,
		Width:       background.Width - 40,
		Height:      50,
		BgColor:     color.RGBA{22, 32, 48, 235},
		BorderColor: utils.PanelBorder,
	}

	return &MarketView{
		ctx:               ctx,
		rows:              make([]marketResourceRow, 0),
		scrollOffset:      0,
		maxVisibleRows:    12,
		returnTo:          ViewTypeGalaxy,
		backgroundPanel:   background,
		headerPanel:       header,
		tablePanel:        table,
		instructionsPanel: instructions,
		rowHits:           make(map[string]tableRowHit),
	}
}

func (mv *MarketView) GetType() ViewType { return ViewTypeMarket }

func (mv *MarketView) OnEnter() {
	mv.scrollOffset = 0
	mv.refreshData()
}

func (mv *MarketView) OnExit() {}

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

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		for resource, hits := range mv.rowHits {
			if pointInRect(mx, my, hits.buy) {
				mv.handleTrade(resource, true)
				return nil
			}
			if pointInRect(mx, my, hits.sell) {
				mv.handleTrade(resource, false)
				return nil
			}
		}
	}
	return nil
}

func (mv *MarketView) Draw(screen *ebiten.Image) {
	mv.backgroundPanel.Draw(screen)
	mv.headerPanel.Draw(screen)
	mv.tablePanel.Draw(screen)
	mv.instructionsPanel.Draw(screen)

	title := "Market Directory"
	subtitle := fmt.Sprintf("Credits Available: %d", mv.humanCredits)
	DrawText(screen, title, mv.headerPanel.X+20, mv.headerPanel.Y+22, utils.TextPrimary)
	DrawText(screen, subtitle, mv.headerPanel.X+20, mv.headerPanel.Y+40, utils.TextSecondary)
	DrawText(screen, fmt.Sprintf("Commodities Listed: %d", len(mv.rows)), mv.headerPanel.X+20, mv.headerPanel.Y+58, utils.TextSecondary)

	headerY := mv.tablePanel.Y + 18
	colResource := mv.tablePanel.X + 20
	colHuman := mv.tablePanel.X + 220
	colNpc := mv.tablePanel.X + 320
	colValue := mv.tablePanel.X + 420
	colAction := mv.tablePanel.X + mv.tablePanel.Width - 180

	DrawText(screen, "Commodity", colResource, headerY, utils.TextPrimary)
	DrawText(screen, "Your Stock", colHuman, headerY, utils.TextPrimary)
	DrawText(screen, "NPC Stock", colNpc, headerY, utils.TextPrimary)
	DrawText(screen, "Avg Value", colValue, headerY, utils.TextPrimary)
	DrawText(screen, "Trade", colAction, headerY, utils.TextPrimary)

	DrawLine(screen, mv.tablePanel.X+15, headerY+12, mv.tablePanel.X+mv.tablePanel.Width-15, headerY+12, utils.PanelBorder)

	if len(mv.rows) == 0 {
		DrawText(screen, "No tradable inventory available yet. Build Trading Posts and accumulate goods to access the market.",
			mv.tablePanel.X+20, headerY+40, utils.TextSecondary)
		mv.drawInstructions(screen)
		return
	}

	y := headerY + 24
	rowHeight := 28
	mv.rowHits = make(map[string]tableRowHit)

	endIndex := mv.scrollOffset + mv.maxVisibleRows
	if endIndex > len(mv.rows) {
		endIndex = len(mv.rows)
	}

	for _, row := range mv.rows[mv.scrollOffset:endIndex] {
		rowRect := image.Rect(mv.tablePanel.X+12, y-4, mv.tablePanel.X+mv.tablePanel.Width-12, y+rowHeight-4)

		rowPanel := NewUIPanel(rowRect.Min.X, rowRect.Min.Y, rowRect.Dx(), rowRect.Dy())
		rowPanel.BgColor = color.RGBA{34, 46, 70, 220}
		rowPanel.BorderColor = utils.PanelBorder
		rowPanel.Draw(screen)

		DrawText(screen, row.resource, colResource, y, utils.TextPrimary)
		DrawText(screen, fmt.Sprintf("%d", row.humanStock), colHuman, y, utils.TextPrimary)
		DrawText(screen, fmt.Sprintf("%d", row.npcStock), colNpc, y, utils.TextPrimary)
		DrawText(screen, fmt.Sprintf("%.0f", row.avgValue), colValue, y, utils.TextSecondary)

		buyLabel := "[Buy]"
		sellLabel := "[Sell]"
		DrawText(screen, buyLabel, colAction, y, utils.SystemGreen)
		DrawText(screen, sellLabel, colAction+60, y, utils.SystemRed)

		mv.rowHits[row.resource] = tableRowHit{
			buy:  makeLabelRect(colAction, y, buyLabel),
			sell: makeLabelRect(colAction+60, y, sellLabel),
		}

		y += rowHeight
	}

	if len(mv.rows) > mv.maxVisibleRows {
		scrollText := fmt.Sprintf("Showing %d-%d of %d commodities",
			mv.scrollOffset+1, mv.scrollOffset+endIndex-mv.scrollOffset, len(mv.rows))
		DrawText(screen, scrollText, mv.tablePanel.X+20, mv.tablePanel.Y+mv.tablePanel.Height-24, utils.TextSecondary)
	}

	mv.drawInstructions(screen)
}

func (mv *MarketView) handleTrade(resource string, buy bool) {
	row := mv.getRow(resource)
	if row == nil {
		return
	}
	qty := tradeLot
	if buy {
		if row.npcStock < qty {
			fmt.Println("[Market] Not enough NPC stock")
			return
		}
		cost := int(row.avgValue * float64(qty))
		if cost <= 0 {
			cost = qty * 50
		}
		human := mv.ctx.GetHumanPlayer()
		if human == nil || human.Credits < cost {
			fmt.Println("[Market] Insufficient credits")
			return
		}
		if !mv.transferStock(resource, qty, false) {
			fmt.Println("[Market] Failed to obtain stock from NPCs")
			return
		}
		if !mv.transferStock(resource, qty, true) {
			fmt.Println("[Market] Failed to add stock to human colony")
			return
		}
		human.Credits -= cost
		fmt.Printf("[Market] Purchased %d units of %s for %d credits\n", qty, resource, cost)
	} else {
		if row.humanStock < qty {
			fmt.Println("[Market] Not enough stock to sell")
			return
		}
		revenue := int(row.avgValue * float64(qty))
		if revenue <= 0 {
			revenue = qty * 40
		}
		if !mv.removeFromHuman(resource, qty) {
			fmt.Println("[Market] Failed to remove stock from player")
			return
		}
		mv.transferToNPC(resource, qty)
		if human := mv.ctx.GetHumanPlayer(); human != nil {
			human.Credits += revenue
		}
		fmt.Printf("[Market] Sold %d units of %s for %d credits\n", qty, resource, revenue)
	}
	mv.refreshData()
}

func (mv *MarketView) drawInstructions(screen *ebiten.Image) {
	instr := fmt.Sprintf("Click [Buy] or [Sell] to trade %d units. Scroll with the mouse wheel. Press [Tab] to focus home; press [Esc] to return.", tradeLot)
	DrawText(screen, instr, mv.instructionsPanel.X+20, mv.instructionsPanel.Y+20, utils.TextSecondary)
}

func (mv *MarketView) getRow(resource string) *marketResourceRow {
	for i := range mv.rows {
		if mv.rows[i].resource == resource {
			return &mv.rows[i]
		}
	}
	return nil
}

func (mv *MarketView) transferStock(resource string, qty int, toHuman bool) bool {
	players := mv.ctx.GetPlayers()
	if len(players) == 0 {
		return false
	}
	if toHuman {
		human := mv.ctx.GetHumanPlayer()
		if human == nil {
			return false
		}
		planet := firstPlanetWithTradingPost(human)
		if planet == nil {
			return false
		}
		planet.AddStoredResource(resource, qty)
		return true
	}

	for _, player := range players {
		if player == nil || player.IsHuman() {
			continue
		}
		for _, planet := range player.OwnedPlanets {
			if planet == nil {
				continue
			}
            if storage := planet.StoredResources[resource]; storage != nil && storage.Amount >= qty {
                planet.RemoveStoredResource(resource, qty)
                return true
            }
		}
	}
	return false
}

func (mv *MarketView) removeFromHuman(resource string, qty int) bool {
	human := mv.ctx.GetHumanPlayer()
	if human == nil {
		return false
	}
	for _, planet := range human.OwnedPlanets {
		if planet == nil {
			continue
		}
		if storage := planet.StoredResources[resource]; storage != nil && storage.Amount >= qty {
			removed := planet.RemoveStoredResource(resource, qty)
			return removed == qty
		}
	}
	return false
}

func (mv *MarketView) transferToNPC(resource string, qty int) {
	players := mv.ctx.GetPlayers()
	for _, player := range players {
		if player == nil || player.IsHuman() {
			continue
		}
		planet := firstPlanetWithTradingPost(player)
		if planet != nil {
			planet.AddStoredResource(resource, qty)
			return
		}
	}
}

func firstPlanetWithTradingPost(player *entities.Player) *entities.Planet {
	if player == nil {
		return nil
	}
	for _, planet := range player.OwnedPlanets {
		if planet == nil {
			continue
		}
		if hasTradingPost(planet) {
			return planet
		}
	}
	if len(player.OwnedPlanets) > 0 {
		return player.OwnedPlanets[0]
	}
	return nil
}

func hasTradingPost(planet *entities.Planet) bool {
	for _, buildingEntity := range planet.Buildings {
		if building, ok := buildingEntity.(*entities.Building); ok {
			if building.BuildingType == "Trading Post" && building.IsOperational {
				return true
			}
		}
	}
	return false
}

func (mv *MarketView) refreshData() {
	mv.rows = mv.rows[:0]
	mv.rowHits = make(map[string]tableRowHit)
	mv.humanCredits = 0

	player := mv.ctx.GetHumanPlayer()
	if player != nil {
		mv.humanCredits = player.Credits
	}

	players := mv.ctx.GetPlayers()
	if len(players) == 0 {
		return
	}

	type stockAccumulator struct {
		human int
		npc   int
		value float64
		count int
	}
	acc := make(map[string]*stockAccumulator)

	for _, player := range players {
		if player == nil {
			continue
		}
		for _, planet := range player.OwnedPlanets {
			if planet == nil {
				continue
			}
			for resourceType, storage := range planet.StoredResources {
				if storage == nil || storage.Amount <= 0 {
					continue
				}
				entry, exists := acc[resourceType]
				if !exists {
					entry = &stockAccumulator{}
					acc[resourceType] = entry
				}
				if player.IsHuman() {
					entry.human += storage.Amount
				} else {
					entry.npc += storage.Amount
				}
				if value := lookupResourceValue(planet, resourceType); value > 0 {
					entry.value += float64(value)
					entry.count++
				}
			}
		}
	}

	for resource, entry := range acc {
		avg := 0.0
		if entry.count > 0 {
			avg = entry.value / float64(entry.count)
		}
		mv.rows = append(mv.rows, marketResourceRow{
			resource:   resource,
			humanStock: entry.human,
			npcStock:   entry.npc,
			avgValue:   avg,
		})
	}

	sort.Slice(mv.rows, func(i, j int) bool {
		totalI := mv.rows[i].humanStock + mv.rows[i].npcStock
		totalJ := mv.rows[j].humanStock + mv.rows[j].npcStock
		if totalI == totalJ {
			return mv.rows[i].resource < mv.rows[j].resource
		}
		return totalI > totalJ
	})
}

// helper for pricing lookup
func lookupResourceValue(planet *entities.Planet, resourceType string) int {
	if planet == nil {
		return 0
	}
	for _, resEntity := range planet.Resources {
		if resource, ok := resEntity.(*entities.Resource); ok {
			if resource.ResourceType == resourceType {
				return resource.Value
			}
		}
	}
	return 0
}

// SetReturnView configures which view to go back to when exiting the market
func (mv *MarketView) SetReturnView(view ViewType) { mv.returnTo = view }

// GetReturnView exposes which view the market should return to
func (mv *MarketView) GetReturnView() ViewType { return mv.returnTo }

func pointInRect(x, y int, rect image.Rectangle) bool {
	return x >= rect.Min.X && x <= rect.Max.X && y >= rect.Min.Y && y <= rect.Max.Y
}

func makeLabelRect(x, y int, label string) image.Rectangle {
	width := len(label) * 6
	return image.Rect(x-2, y-6, x+width+2, y+10)
}
