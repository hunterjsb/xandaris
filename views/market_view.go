package views

import (
	"fmt"
	"image"
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hunterjsb/xandaris/economy"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/utils"
)

type marketResourceRow struct {
	resource       string
	humanStock     int
	galaxySupply   int     // total supply galaxy-wide (all players)
	buyPrice       float64
	sellPrice      float64
	basePrice      float64 // base/equilibrium price
	localBuyPrice  float64 // per-system price
	localSellPrice float64
	demand         float64
	trend          float64
	importFee      float64 // dynamic fee rate (0.05-0.20)
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
	statusMsg         string
	statusColor       color.RGBA
	statusTicks       int
	tradingPlanet     *entities.Planet // the planet trades are scoped to
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
	mv.statusMsg = ""
	mv.selectTradingPlanet()
	mv.refreshData()
}

// selectTradingPlanet auto-selects the player's first planet with a Trading Post.
func (mv *MarketView) selectTradingPlanet() {
	player := mv.ctx.GetHumanPlayer()
	if player == nil {
		mv.tradingPlanet = nil
		return
	}
	// Prefer a planet with a Trading Post
	for _, planet := range player.OwnedPlanets {
		if planet == nil {
			continue
		}
		for _, be := range planet.Buildings {
			if b, ok := be.(*entities.Building); ok {
				if b.BuildingType == "Trading Post" && b.IsOperational {
					mv.tradingPlanet = planet
					return
				}
			}
		}
	}
	// Fallback: home planet or first owned planet
	if player.HomePlanet != nil {
		mv.tradingPlanet = player.HomePlanet
	} else if len(player.OwnedPlanets) > 0 {
		mv.tradingPlanet = player.OwnedPlanets[0]
	}
}

func (mv *MarketView) OnExit() {}

func (mv *MarketView) Update() error {
	mv.refreshData()

	if mv.statusTicks > 0 {
		mv.statusTicks--
		if mv.statusTicks <= 0 {
			mv.statusMsg = ""
		}
	}

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
	if mv.tradingPlanet != nil {
		title = fmt.Sprintf("Market at %s", mv.tradingPlanet.Name)
	}
	subtitle := fmt.Sprintf("Credits: %d", mv.humanCredits)
	DrawText(screen, title, mv.headerPanel.X+20, mv.headerPanel.Y+22, utils.TextPrimary)
	DrawText(screen, subtitle, mv.headerPanel.X+20, mv.headerPanel.Y+40, utils.TextSecondary)

	// Stock on this planet (right-aligned)
	if mv.tradingPlanet != nil {
		totalStock := 0
		for _, s := range mv.tradingPlanet.StoredResources {
			if s != nil {
				totalStock += s.Amount
			}
		}
		stockInfo := fmt.Sprintf("Stock: %d  Pop: %d", totalStock, mv.tradingPlanet.Population)
		stockWidth := len(stockInfo) * 6
		DrawText(screen, stockInfo, mv.headerPanel.X+mv.headerPanel.Width-stockWidth-20, mv.headerPanel.Y+22, utils.TextSecondary)
	}

	// Show trade mode (local vs galaxy-wide) with fee info
	tradeMode := "Galaxy-wide trading (import/export fees apply)"
	tradeModeColor := utils.SystemOrange
	// Check if there's an NPC in the same system for local trading
	if mv.tradingPlanet != nil {
		for _, sys := range mv.ctx.GetSystems() {
			for _, e := range sys.Entities {
				if p, ok := e.(*entities.Planet); ok && p.GetID() == mv.tradingPlanet.GetID() {
					// Check for NPC planets in this system
					for _, e2 := range sys.Entities {
						if p2, ok := e2.(*entities.Planet); ok && p2.Owner != "" && p2.Owner != mv.tradingPlanet.Owner {
							tradeMode = "Local market (no fees)"
							tradeModeColor = utils.SystemGreen
							break
						}
					}
					break
				}
			}
		}
	}
	DrawText(screen, tradeMode, mv.headerPanel.X+300, mv.headerPanel.Y+40, tradeModeColor)

	// Show status message or commodity count
	if mv.statusMsg != "" {
		DrawText(screen, mv.statusMsg, mv.headerPanel.X+20, mv.headerPanel.Y+58, mv.statusColor)
	} else {
		DrawText(screen, fmt.Sprintf("Commodities: %d", len(mv.rows)), mv.headerPanel.X+20, mv.headerPanel.Y+58, utils.TextSecondary)
	}

	headerY := mv.tablePanel.Y + 18
	colResource := mv.tablePanel.X + 20
	colStock := mv.tablePanel.X + 130
	colGalSupply := mv.tablePanel.X + 195
	colBuy := mv.tablePanel.X + 275
	colSell := mv.tablePanel.X + 365
	colBase := mv.tablePanel.X + 450
	colDemand := mv.tablePanel.X + 520
	colTrend := mv.tablePanel.X + 595
	colFee := mv.tablePanel.X + 645
	colAction := mv.tablePanel.X + mv.tablePanel.Width - 120

	DrawText(screen, "Commodity", colResource, headerY, utils.TextPrimary)
	DrawText(screen, "Stock", colStock, headerY, utils.TextPrimary)
	DrawText(screen, "Galaxy", colGalSupply, headerY, utils.TextSecondary)
	DrawText(screen, "Buy @", colBuy, headerY, utils.SystemGreen)
	DrawText(screen, "Sell @", colSell, headerY, utils.SystemOrange)
	DrawText(screen, "Base", colBase, headerY, utils.TextSecondary)
	DrawText(screen, "Demand", colDemand, headerY, utils.TextSecondary)
	DrawText(screen, "Trend", colTrend, headerY, utils.TextPrimary)
	DrawText(screen, "Fee", colFee, headerY, utils.TextSecondary)
	DrawText(screen, "Trade", colAction, headerY, utils.TextPrimary)

	DrawLine(screen, mv.tablePanel.X+15, headerY+12, mv.tablePanel.X+mv.tablePanel.Width-15, headerY+12, utils.PanelBorder)

	if len(mv.rows) == 0 {
		DrawText(screen, "No tradable inventory available yet. Build Trading Posts and accumulate goods.",
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

		// Stock: player's stock on trading planet (white = plenty, red = low)
		stockColor := utils.TextPrimary
		if row.humanStock < 20 {
			stockColor = utils.SystemRed
		}
		DrawText(screen, fmt.Sprintf("%d", row.humanStock), colStock, y, stockColor)

		// Galaxy supply (total across all players)
		DrawText(screen, fmt.Sprintf("%d", row.galaxySupply), colGalSupply, y, utils.TextSecondary)

		// Buy price — green if below base (good deal), red if above (expensive)
		buyColor := utils.SystemGreen
		if row.basePrice > 0 && row.buyPrice > row.basePrice*1.2 {
			buyColor = utils.SystemRed
		} else if row.basePrice > 0 && row.buyPrice > row.basePrice {
			buyColor = utils.SystemOrange
		}
		DrawText(screen, fmt.Sprintf("%.0f", row.buyPrice), colBuy, y, buyColor)

		// Sell price — green if above base (profit), orange if below
		sellColor := utils.SystemOrange
		if row.basePrice > 0 && row.sellPrice > row.basePrice {
			sellColor = utils.SystemGreen
		}
		DrawText(screen, fmt.Sprintf("%.0f", row.sellPrice), colSell, y, sellColor)

		// Base price for reference
		baseStr := fmt.Sprintf("%.0f", row.basePrice)
		if row.basePrice == 0 {
			baseStr = "--"
		}
		DrawText(screen, baseStr, colBase, y, utils.TextSecondary)

		// Demand signal
		demandColor := utils.TextSecondary
		if row.demand > 30 {
			demandColor = utils.SystemOrange
		}
		DrawText(screen, fmt.Sprintf("%.0f", row.demand), colDemand, y, demandColor)

		// Trend indicator
		trendStr := "--"
		trendColor := utils.TextSecondary
		if row.trend > 1.0 {
			trendStr = "^^"
			trendColor = utils.SystemGreen
		} else if row.trend > 0.1 {
			trendStr = "^"
			trendColor = utils.SystemGreen
		} else if row.trend < -1.0 {
			trendStr = "vv"
			trendColor = utils.SystemRed
		} else if row.trend < -0.1 {
			trendStr = "v"
			trendColor = utils.SystemRed
		}
		DrawText(screen, trendStr, colTrend, y, trendColor)

		// Import/export fee indicator
		feeStr := fmt.Sprintf("%.0f%%", row.importFee*100)
		feeColor := utils.TextSecondary
		if row.importFee >= 0.15 {
			feeColor = utils.SystemRed
		} else if row.importFee <= 0.05 {
			feeColor = utils.SystemGreen
		}
		DrawText(screen, feeStr, colFee, y, feeColor)

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
		scrollText := fmt.Sprintf("Showing %d-%d of %d",
			mv.scrollOffset+1, endIndex, len(mv.rows))
		DrawText(screen, scrollText, mv.tablePanel.X+20, mv.tablePanel.Y+mv.tablePanel.Height-24, utils.TextSecondary)
	}

	// Trade history section (below commodity rows)
	mv.drawTradeHistory(screen, y+10)

	mv.drawInstructions(screen)
}

func (mv *MarketView) drawTradeHistory(screen *ebiten.Image, startY int) {
	exec := mv.ctx.GetTradeExecutor()
	if exec == nil {
		return
	}

	history := exec.GetHistory(8)
	if len(history) == 0 {
		return
	}

	maxY := mv.tablePanel.Y + mv.tablePanel.Height - 10

	// Section header
	if startY+15 > maxY {
		return
	}
	DrawLine(screen, mv.tablePanel.X+15, startY, mv.tablePanel.X+mv.tablePanel.Width-15, startY, utils.PanelBorder)
	startY += 8
	DrawText(screen, "Recent Trades", mv.tablePanel.X+20, startY, utils.TextSecondary)
	startY += 18

	// Draw trades (newest first)
	for i := len(history) - 1; i >= 0; i-- {
		if startY+14 > maxY {
			break
		}
		t := history[i]
		actionColor := utils.SystemGreen
		actionStr := "bought"
		if t.Action == "sell" {
			actionColor = utils.SystemRed
			actionStr = "sold"
		}

		line := fmt.Sprintf("%s %s %d %s @ %.0f = %dcr",
			t.Player, actionStr, t.Quantity, t.Resource, t.UnitPrice, t.Total)
		DrawText(screen, line, mv.tablePanel.X+25, startY, actionColor)
		startY += 16
	}
}

func (mv *MarketView) handleTrade(resource string, buy bool) {
	exec := mv.ctx.GetTradeExecutor()
	if exec == nil {
		mv.setStatus("Market not available", utils.SystemRed)
		return
	}

	human := mv.ctx.GetHumanPlayer()
	players := mv.ctx.GetPlayers()
	qty := tradeLot

	if buy {
		record, err := exec.Buy(human, players, resource, qty, mv.tradingPlanet)
		if err != nil {
			mv.setStatus(err.Error(), utils.SystemRed)
			return
		}
		mv.setStatus(fmt.Sprintf("Bought %d %s for %d credits", record.Quantity, record.Resource, record.Total), utils.SystemGreen)
	} else {
		record, err := exec.Sell(human, players, resource, qty, mv.tradingPlanet)
		if err != nil {
			mv.setStatus(err.Error(), utils.SystemRed)
			return
		}
		mv.setStatus(fmt.Sprintf("Sold %d %s for %d credits", record.Quantity, record.Resource, record.Total), utils.SystemGreen)
	}
	mv.refreshData()
}

func (mv *MarketView) setStatus(msg string, c color.RGBA) {
	mv.statusMsg = msg
	mv.statusColor = c
	mv.statusTicks = 180 // ~3 seconds at 60fps
}

func (mv *MarketView) drawInstructions(screen *ebiten.Image) {
	instr := fmt.Sprintf("[Buy]/[Sell] trades %d units. Scroll with mouse wheel. [Esc] to return.", tradeLot)
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

	// Find the system ID of the trading planet for NPC stock scoping and local prices
	tradingSystemID := -1
	if mv.tradingPlanet != nil {
		for _, sys := range mv.ctx.GetSystems() {
			for _, entity := range sys.Entities {
				if p, ok := entity.(*entities.Planet); ok && p.GetID() == mv.tradingPlanet.GetID() {
					tradingSystemID = sys.ID
					break
				}
			}
			if tradingSystemID >= 0 {
				break
			}
		}
	}

	// Build set of planet IDs in the trading system for NPC stock scoping
	systemPlanetIDs := make(map[int]bool)
	if tradingSystemID >= 0 {
		for _, sys := range mv.ctx.GetSystems() {
			if sys.ID == tradingSystemID {
				for _, entity := range sys.Entities {
					if p, ok := entity.(*entities.Planet); ok {
						systemPlanetIDs[p.GetID()] = true
					}
				}
				break
			}
		}
	}

	// Get market snapshot for dynamic prices (after tradingSystemID is known)
	market := mv.ctx.GetMarket()
	type priceInfo struct {
		buyPrice       float64
		sellPrice      float64
		basePrice      float64
		totalSupply    int
		localBuyPrice  float64
		localSellPrice float64
		demand         float64
		trend          float64
	}
	var snapshot map[string]priceInfo
	if market != nil {
		snap := market.GetSnapshot()
		snapshot = make(map[string]priceInfo, len(snap.Resources))
		for name, rm := range snap.Resources {
			pi := priceInfo{rm.BuyPrice, rm.SellPrice, rm.BasePrice, int(rm.TotalSupply), 0, 0, rm.TotalDemand, rm.PriceVelocity}
			if tradingSystemID >= 0 {
				pi.localBuyPrice = market.GetLocalBuyPrice(name, tradingSystemID)
				pi.localSellPrice = market.GetLocalSellPrice(name, tradingSystemID)
			}
			snapshot[name] = pi
		}
	}

	type stockAccumulator struct {
		human int
		npc   int
	}
	acc := make(map[string]*stockAccumulator)

	for _, p := range players {
		if p == nil {
			continue
		}
		if p.IsHuman() {
			// "You" column: show only the trading planet's stock
			if mv.tradingPlanet != nil {
				for resourceType, storage := range mv.tradingPlanet.StoredResources {
					if storage == nil || storage.Amount <= 0 {
						continue
					}
					entry, exists := acc[resourceType]
					if !exists {
						entry = &stockAccumulator{}
						acc[resourceType] = entry
					}
					entry.human += storage.Amount
				}
			} else {
				// Fallback: aggregate all human planets
				for _, planet := range p.OwnedPlanets {
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
						entry.human += storage.Amount
					}
				}
			}
		} else {
			// NPC column: only planets in the same system as the trading planet
			for _, planet := range p.OwnedPlanets {
				if planet == nil {
					continue
				}
				// If we have system scoping, only count planets in the same system
				if tradingSystemID >= 0 && !systemPlanetIDs[planet.GetID()] {
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
					entry.npc += storage.Amount
				}
			}
		}
	}

	// Include resources with market prices but no stock
	if snapshot != nil {
		for name := range snapshot {
			if _, exists := acc[name]; !exists {
				acc[name] = &stockAccumulator{}
			}
		}
	}

	for resource, entry := range acc {
		row := marketResourceRow{
			resource:     resource,
			humanStock:   entry.human,
			galaxySupply: entry.human + entry.npc,
		}

		if snapshot != nil {
			if s, ok := snapshot[resource]; ok {
				row.buyPrice = s.buyPrice
				row.sellPrice = s.sellPrice
				row.basePrice = s.basePrice
				row.galaxySupply = s.totalSupply
				row.localBuyPrice = s.localBuyPrice
				row.localSellPrice = s.localSellPrice
				row.demand = s.demand
				row.trend = s.trend

				row.importFee = economy.ComputeImportFee(float64(s.totalSupply), s.demand)
			}
		}

		if row.buyPrice == 0 {
			row.buyPrice = 50
			row.sellPrice = 40
		}

		mv.rows = append(mv.rows, row)
	}

	sort.Slice(mv.rows, func(i, j int) bool {
		if mv.rows[i].galaxySupply == mv.rows[j].galaxySupply {
			return mv.rows[i].resource < mv.rows[j].resource
		}
		return mv.rows[i].galaxySupply > mv.rows[j].galaxySupply
	})
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
