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

var mvRectCache = utils.NewRectImageCache()

type marketResourceRow struct {
	resource       string
	humanStock     int
	galaxySupply   int     // total supply galaxy-wide (all players)
	buyPrice       float64
	sellPrice      float64
	basePrice      float64 // base/equilibrium price
	bestBuyPrice   float64 // cheapest price anywhere in the galaxy
	bestSystem     string  // system with cheapest price
	demand         float64
	trend          float64
	importFee      float64 // dynamic fee rate (0.05-0.20)
	galaxyNetFlow  float64 // galaxy-wide production - consumption
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
	refreshCounter    int
}

type tableRowHit struct {
	buy  image.Rectangle
	sell image.Rectangle
}

const tradeLot = 10

// NewMarketView creates a new market view instance
func NewMarketView(ctx GameContext) *MarketView {
	panelMargin := 40
	background := &UIPanel{
		X:           panelMargin,
		Y:           panelMargin,
		Width:       ScreenWidth - panelMargin*2,
		Height:      ScreenHeight - panelMargin*2,
		BgColor:     utils.Theme.PanelBgSolid,
		BorderColor: utils.Theme.PanelBorder,
	}

	header := &UIPanel{
		X:           background.X + 15,
		Y:           background.Y + 12,
		Width:       background.Width - 30,
		Height:      55,
		BgColor:     color.RGBA{18, 22, 42, 250},
		BorderColor: utils.Theme.PanelBorder,
	}

	table := &UIPanel{
		X:           header.X,
		Y:           header.Y + header.Height + 6,
		Width:       header.Width,
		Height:      background.Height - header.Height - 80,
		BgColor:     color.RGBA{14, 18, 34, 240},
		BorderColor: utils.Theme.PanelBorder,
	}

	instructions := &UIPanel{
		X:           background.X + 15,
		Y:           table.Y + table.Height + 6,
		Width:       background.Width - 30,
		Height:      35,
		BgColor:     color.RGBA{18, 22, 42, 240},
		BorderColor: utils.Theme.PanelBorder,
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
	// Throttle refresh to every 30 frames (~0.5s) to avoid per-frame galaxy flow computation
	mv.refreshCounter++
	if mv.refreshCounter%30 == 0 {
		mv.refreshData()
	}

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

	title := "MARKET"
	if mv.tradingPlanet != nil {
		title = fmt.Sprintf("MARKET — %s", mv.tradingPlanet.Name)
	}
	DrawText(screen, title, mv.headerPanel.X+20, mv.headerPanel.Y+15, utils.Theme.Accent)
	credStr := fmt.Sprintf("Credits: %s", formatNumber(mv.humanCredits))
	DrawText(screen, credStr, mv.headerPanel.X+20, mv.headerPanel.Y+35, utils.Theme.TextLight)

	// Stock + trade mode (right side)
	if mv.tradingPlanet != nil {
		totalStock := 0
		for _, s := range mv.tradingPlanet.StoredResources {
			if s != nil {
				totalStock += s.Amount
			}
		}
		stockInfo := fmt.Sprintf("Stock: %s  Pop: %s", formatNumber(totalStock), formatPopulation(mv.tradingPlanet.Population))
		stockWidth := len(stockInfo) * utils.CharWidth()
		DrawText(screen, stockInfo, mv.headerPanel.X+mv.headerPanel.Width-stockWidth-20, mv.headerPanel.Y+15, utils.Theme.TextDim)

		// Trade mode indicator
		tradeMode := "Galaxy (fees apply)"
		tradeModeColor := utils.SystemOrange
		for _, sys := range mv.ctx.GetSystems() {
			for _, e := range sys.Entities {
				if p, ok := e.(*entities.Planet); ok && p.GetID() == mv.tradingPlanet.GetID() {
					for _, e2 := range sys.Entities {
						if p2, ok := e2.(*entities.Planet); ok && p2.Owner != "" && p2.Owner != mv.tradingPlanet.Owner {
							tradeMode = "Local (no fees)"
							tradeModeColor = utils.SystemGreen
							break
						}
					}
					break
				}
			}
		}
		tmWidth := len(tradeMode) * utils.CharWidth()
		DrawText(screen, tradeMode, mv.headerPanel.X+mv.headerPanel.Width-tmWidth-20, mv.headerPanel.Y+35, tradeModeColor)
	}

	// Status message
	if mv.statusMsg != "" {
		DrawText(screen, mv.statusMsg, mv.headerPanel.X+250, mv.headerPanel.Y+35, mv.statusColor)
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

	DrawText(screen, "Commodity", colResource, headerY, utils.Theme.TextDim)
	DrawText(screen, "Stock", colStock, headerY, utils.Theme.TextDim)
	DrawText(screen, "Galaxy", colGalSupply, headerY, utils.Theme.TextDim)
	DrawText(screen, "Buy @", colBuy, headerY, utils.Theme.TextDim)
	DrawText(screen, "Sell @", colSell, headerY, utils.Theme.TextDim)
	DrawText(screen, "Best@", colBase, headerY, utils.Theme.TextDim)
	DrawText(screen, "Flow", colDemand, headerY, utils.Theme.TextDim)
	DrawText(screen, "Status", colTrend, headerY, utils.Theme.TextDim)
	DrawText(screen, "Fee", colFee, headerY, utils.Theme.TextDim)
	DrawText(screen, "Trade", colAction, headerY, utils.Theme.TextDim)

	DrawLine(screen, mv.tablePanel.X+12, headerY+14, mv.tablePanel.X+mv.tablePanel.Width-12, headerY+14, utils.Theme.PanelBorder)

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

	mx, my := ebiten.CursorPosition()
	for _, row := range mv.rows[mv.scrollOffset:endIndex] {
		// Hover highlight
		if mx >= mv.tablePanel.X+12 && mx <= mv.tablePanel.X+mv.tablePanel.Width-12 &&
			my >= y-4 && my < y+rowHeight-4 {
			hoverBg := mvRectCache.GetOrCreate(mv.tablePanel.Width-24, rowHeight, utils.Theme.PanelHover)
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(mv.tablePanel.X+12), float64(y-4))
			screen.DrawImage(hoverBg, opts)
		}

		DrawText(screen, row.resource, colResource, y, utils.Theme.TextLight)

		// Stock: player's stock on trading planet (white = plenty, red = low)
		stockColor := utils.TextPrimary
		if row.humanStock < 20 {
			stockColor = utils.SystemRed
		}
		DrawText(screen, fmt.Sprintf("%d", row.humanStock), colStock, y, stockColor)

		// Galaxy supply (total across all players)
		DrawText(screen, fmt.Sprintf("%d", row.galaxySupply), colGalSupply, y, utils.TextSecondary)

		// Buy price colored by ratio to base: green=cheap, orange=fair, red=expensive
		buyColor := utils.TextPrimary
		if row.basePrice > 0 {
			ratio := row.buyPrice / row.basePrice
			if ratio < 0.5 {
				buyColor = utils.SystemGreen // bargain
			} else if ratio < 0.9 {
				buyColor = utils.SystemGreen // good deal
			} else if ratio > 2.0 {
				buyColor = utils.SystemRed // very expensive
			} else if ratio > 1.2 {
				buyColor = utils.SystemOrange // above average
			}
		}
		buyStr := fmt.Sprintf("%.0f", row.buyPrice)
		DrawText(screen, buyStr, colBuy, y, buyColor)
		// Trend arrow after buy price
		if row.trend > 1.0 {
			DrawText(screen, "^", colBuy+len(buyStr)*utils.CharWidth()+2, y, utils.SystemRed)
		} else if row.trend < -1.0 {
			DrawText(screen, "v", colBuy+len(buyStr)*utils.CharWidth()+2, y, utils.SystemGreen)
		}

		// Sell price colored by ratio to base: green=profitable, orange=below base
		sellColor := utils.TextPrimary
		if row.basePrice > 0 {
			ratio := row.sellPrice / row.basePrice
			if ratio > 1.5 {
				sellColor = utils.SystemGreen // very profitable
			} else if ratio > 0.9 {
				sellColor = utils.SystemGreen // profitable
			} else if ratio < 0.3 {
				sellColor = utils.SystemRed // terrible price
			} else {
				sellColor = utils.SystemOrange // below base
			}
		}
		DrawText(screen, fmt.Sprintf("%.0f", row.sellPrice), colSell, y, sellColor)

		// Best price across galaxy (shows where cheapest + system name)
		if row.bestBuyPrice > 0 && row.bestSystem != "" && row.bestBuyPrice < row.buyPrice*0.8 {
			bestStr := fmt.Sprintf("%.0f", row.bestBuyPrice)
			DrawText(screen, bestStr, colBase, y, utils.SystemGreen)
			// Show system name in smaller text after price
			sysLabel := row.bestSystem
			if len(sysLabel) > 5 {
				sysLabel = sysLabel[:5]
			}
			DrawText(screen, sysLabel, colBase+len(bestStr)*utils.CharWidth()+3, y, utils.TextSecondary)
		} else if row.bestBuyPrice > 0 {
			DrawText(screen, fmt.Sprintf("%.0f", row.bestBuyPrice), colBase, y, utils.TextSecondary)
		} else {
			DrawText(screen, fmt.Sprintf("%.0f", row.basePrice), colBase, y, utils.TextSecondary)
		}

		// Galaxy net flow (production - consumption)
		flowColor := utils.TextSecondary
		flowStr := "0"
		if row.galaxyNetFlow > 0.5 {
			flowColor = utils.SystemGreen
			flowStr = fmt.Sprintf("+%.0f", row.galaxyNetFlow)
		} else if row.galaxyNetFlow < -0.5 {
			flowColor = utils.SystemRed
			flowStr = fmt.Sprintf("%.0f", row.galaxyNetFlow)
		}
		DrawText(screen, flowStr, colDemand, y, flowColor)

		// Scarcity indicator (uses economy.ComputeScarcity, same thresholds as API)
		scarcity := economy.ComputeScarcity(float64(row.galaxySupply), row.demand)
		scarcityStr := "OK"
		scarcityColor := utils.TextSecondary
		switch scarcity {
		case "Abundant":
			scarcityStr = "Full"
			scarcityColor = utils.SystemGreen
		case "Moderate":
			scarcityStr = "OK"
		case "Scarce":
			scarcityStr = "Low"
			scarcityColor = utils.SystemOrange
		case "Critical":
			scarcityStr = "Crit"
			scarcityColor = utils.SystemRed
		case "Depleted":
			scarcityStr = "GONE"
			scarcityColor = utils.SystemRed
		}
		DrawText(screen, scarcityStr, colTrend, y, scarcityColor)

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
		actionColor := utils.TextSecondary
		actionStr := "bought"
		if t.Action == "sell" {
			actionStr = "sold"
		}
		// Highlight human player's trades
		human := mv.ctx.GetHumanPlayer()
		if human != nil && t.Player == human.Name {
			if t.Action == "sell" {
				actionColor = utils.SystemRed
			} else {
				actionColor = utils.SystemGreen
			}
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
	instr := fmt.Sprintf("Click Buy/Sell to trade %d units  |  Scroll to browse  |  Esc to close", tradeLot)
	DrawTextCentered(screen, instr, mv.instructionsPanel.X+mv.instructionsPanel.Width/2, mv.instructionsPanel.Y+12, utils.Theme.TextDim, 0.9)
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
		buyPrice    float64
		sellPrice   float64
		basePrice   float64
		totalSupply int
		demand      float64
		trend       float64
	}
	var snapshot map[string]priceInfo
	if market != nil {
		snap := market.GetSnapshot()
		snapshot = make(map[string]priceInfo, len(snap.Resources))
		for name, rm := range snap.Resources {
			snapshot[name] = priceInfo{rm.BuyPrice, rm.SellPrice, rm.BasePrice, int(rm.TotalSupply), rm.TotalDemand, rm.PriceVelocity}
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
				row.demand = s.demand
				row.trend = s.trend
				row.importFee = economy.ComputeImportFee(float64(s.totalSupply), s.demand)

				// Find cheapest price across inhabited systems
				bestPrice := row.buyPrice
				bestSys := ""
				for _, sys := range mv.ctx.GetSystems() {
					// Only check systems with owned planets
					hasOwned := false
					for _, e := range sys.Entities {
						if p, ok := e.(*entities.Planet); ok && p.Owner != "" {
							hasOwned = true
							break
						}
					}
					if !hasOwned {
						continue
					}
					lp := market.GetLocalBuyPrice(resource, sys.ID)
					if lp < bestPrice {
						bestPrice = lp
						bestSys = sys.Name
					}
				}
				row.bestBuyPrice = bestPrice
				row.bestSystem = bestSys
			}
		}

		if row.buyPrice == 0 {
			row.buyPrice = 50
			row.sellPrice = 40
		}

		mv.rows = append(mv.rows, row)
	}

	// Compute galaxy-wide net flows
	galaxyFlows := mv.computeGalaxyFlows()
	for i := range mv.rows {
		if flow, ok := galaxyFlows[mv.rows[i].resource]; ok {
			mv.rows[i].galaxyNetFlow = flow
		}
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

// computeGalaxyFlows calculates galaxy-wide net production - consumption per resource.
func (mv *MarketView) computeGalaxyFlows() map[string]float64 {
	flow := make(map[string]float64)
	for _, player := range mv.ctx.GetPlayers() {
		if player == nil {
			continue
		}
		for _, planet := range player.OwnedPlanets {
			if planet == nil {
				continue
			}
			// Mine production
			for _, resEntity := range planet.Resources {
				res, ok := resEntity.(*entities.Resource)
				if !ok || res.Abundance <= 0 {
					continue
				}
				resIDStr := fmt.Sprintf("%d", res.GetID())
				multiplier := 0.0
				for _, be := range planet.Buildings {
					if b, ok := be.(*entities.Building); ok {
						if b.BuildingType == "Mine" && b.AttachedTo == resIDStr && b.IsOperational {
							multiplier += b.GetStaffingRatio() * b.ProductionBonus
						}
					}
				}
				if multiplier > 0 {
					af := float64(res.Abundance) / 70.0
					if af > 1.0 { af = 1.0 }
					if af < 0.1 { af = 0.1 }
					flow[res.ResourceType] += 8.0 * res.ExtractionRate * multiplier * af
				}
			}
			// Refinery
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.BuildingType == "Refinery" && b.IsOperational {
					lm := 1.0 + float64(b.Level-1)*0.3
					flow["Fuel"] += 3.0 * lm
					flow["Oil"] -= 2.0 * lm
				}
			}
			// Population consumption
			for _, rate := range economy.PopulationConsumption {
				flow[rate.ResourceType] -= float64(planet.Population) / rate.PopDivisor * rate.PerPopulation
			}
			// Building upkeep
			for _, be := range planet.Buildings {
				if b, ok := be.(*entities.Building); ok && b.IsOperational {
					if upkeeps, found := economy.BuildingResourceUpkeep[b.BuildingType]; found {
						for _, u := range upkeeps {
							flow[u.ResourceType] -= float64(u.Amount)
						}
					}
				}
			}
		}
	}
	return flow
}

func pointInRect(x, y int, rect image.Rectangle) bool {
	return x >= rect.Min.X && x <= rect.Max.X && y >= rect.Min.Y && y <= rect.Max.Y
}

func makeLabelRect(x, y int, label string) image.Rectangle {
	width := len(label) * utils.CharWidth()
	return image.Rect(x-2, y-6, x+width+2, y+10)
}
