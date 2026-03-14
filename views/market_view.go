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
	resource      string
	humanStock    int
	npcStock      int
	buyPrice      float64
	sellPrice     float64
	localBuyPrice float64 // per-system price
	localSellPrice float64
	demand        float64
	trend         float64
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

	// Show status message or commodity count
	if mv.statusMsg != "" {
		DrawText(screen, mv.statusMsg, mv.headerPanel.X+20, mv.headerPanel.Y+58, mv.statusColor)
	} else {
		DrawText(screen, fmt.Sprintf("Commodities: %d", len(mv.rows)), mv.headerPanel.X+20, mv.headerPanel.Y+58, utils.TextSecondary)
	}

	headerY := mv.tablePanel.Y + 18
	colResource := mv.tablePanel.X + 20
	colHuman := mv.tablePanel.X + 150
	colNpc := mv.tablePanel.X + 220
	colBuy := mv.tablePanel.X + 290
	colSell := mv.tablePanel.X + 370
	colGalaxy := mv.tablePanel.X + 450
	colTrend := mv.tablePanel.X + 530
	colAction := mv.tablePanel.X + mv.tablePanel.Width - 140

	DrawText(screen, "Commodity", colResource, headerY, utils.TextPrimary)
	DrawText(screen, "Stock", colHuman, headerY, utils.TextPrimary)
	DrawText(screen, "Supply", colNpc, headerY, utils.TextPrimary)
	DrawText(screen, "Buy @", colBuy, headerY, utils.TextPrimary)
	DrawText(screen, "Sell @", colSell, headerY, utils.TextPrimary)
	DrawText(screen, "Galaxy", colGalaxy, headerY, utils.TextPrimary)
	DrawText(screen, "Trend", colTrend, headerY, utils.TextPrimary)
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
		DrawText(screen, fmt.Sprintf("%d", row.humanStock), colHuman, y, utils.TextPrimary)
		DrawText(screen, fmt.Sprintf("%d", row.npcStock), colNpc, y, utils.TextSecondary)

		// Show effective local prices (fallback to galaxy if no local data)
		displayBuy := row.buyPrice
		displaySell := row.sellPrice
		hasLocal := row.localBuyPrice > 0
		if hasLocal {
			displayBuy = row.localBuyPrice
			displaySell = row.localSellPrice
		}

		// Color: green = favorable vs galaxy avg, red = unfavorable
		buyColor := utils.TextPrimary
		if hasLocal {
			if displayBuy < row.buyPrice*0.95 {
				buyColor = utils.SystemGreen // cheap locally
			} else if displayBuy > row.buyPrice*1.05 {
				buyColor = utils.SystemRed // expensive locally
			}
		}
		DrawText(screen, fmt.Sprintf("%.0f", displayBuy), colBuy, y, buyColor)

		sellColor := utils.TextPrimary
		if hasLocal {
			if displaySell > row.sellPrice*1.05 {
				sellColor = utils.SystemGreen // good sell price
			} else if displaySell < row.sellPrice*0.95 {
				sellColor = utils.SystemRed // bad sell price
			}
		}
		DrawText(screen, fmt.Sprintf("%.0f", displaySell), colSell, y, sellColor)

		// Galaxy average as reference
		DrawText(screen, fmt.Sprintf("%.0f", row.buyPrice), colGalaxy, y, utils.TextSecondary)

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

	mv.drawInstructions(screen)
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
			pi := priceInfo{rm.BuyPrice, rm.SellPrice, 0, 0, rm.TotalDemand, rm.PriceVelocity}
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
			resource:   resource,
			humanStock: entry.human,
			npcStock:   entry.npc,
		}

		if snapshot != nil {
			if s, ok := snapshot[resource]; ok {
				row.buyPrice = s.buyPrice
				row.sellPrice = s.sellPrice
				row.localBuyPrice = s.localBuyPrice
				row.localSellPrice = s.localSellPrice
				row.demand = s.demand
				row.trend = s.trend
			}
		}

		if row.buyPrice == 0 {
			row.buyPrice = 50
			row.sellPrice = 40
		}

		mv.rows = append(mv.rows, row)
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
